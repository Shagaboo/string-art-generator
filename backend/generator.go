package main

import (
	"log"
	"runtime"
	"sync"
)

// Generator генерирует string art из изображения
// ERROR-BASED алгоритм: работаем с residual image (ошибкой) для предотвращения стагнации
type Generator struct {
	original  []int // Оригинальное изображение (не изменяется)
	current   []int // Текущее изображение (обновляется после каждой линии)
	error     []int // Ошибка: error = original - current (где нужно добавить детали)
	lineCache *LineCache
	nails     []Nail
	width     int
	height    int
	workers   int
}

// NewGenerator создает новый генератор с error-based алгоритмом
func NewGenerator(grayscale []int, lineCache *LineCache, nails []Nail, width, height int) *Generator {
	workers := runtime.NumCPU() * 4 // Максимальная параллельность
	if workers < 16 {
		workers = 16 // Минимум 16 воркеров
	}
	if workers > 64 {
		workers = 64 // Максимум 64 для скорости
	}

	// Создаем копии для original и current
	original := make([]int, len(grayscale))
	current := make([]int, len(grayscale))
	copy(original, grayscale)
	copy(current, grayscale)

	// Инициализируем error: error = original (темные области = большая ошибка)
	// В оригинале используется инвертированное изображение для error
	// Но так как мы ищем максимальную ошибку, нам нужны темные области с большой ошибкой
	// Поэтому: error = 255 - original (темные пиксели = большая ошибка)
	error := make([]int, len(grayscale))
	for i := 0; i < len(grayscale); i++ {
		// Темные области (маленькие значения) = большая ошибка
		// Это правильно, так как мы хотим рисовать линии в темных областях
		error[i] = 255 - original[i]
	}

	return &Generator{
		original:  original,
		current:   current,
		error:     error,
		lineCache: lineCache,
		nails:     nails,
		width:     width,
		height:    height,
		workers:   workers,
	}
}

// Generate генерирует линии string art используя ERROR-BASED алгоритм
// Вместо поиска минимальной яркости ищем линию с максимальной ошибкой
// Это предотвращает стагнацию и позволяет продолжать улучшать изображение
func (g *Generator) Generate(lineCount int, lineWeight int) []LineSegment {
	lines := make([]LineSegment, 0, lineCount)
	currentNail := 0
	nailCount := len(g.nails)

	// Ограничение на повторное использование гвоздей (как last_pins в оригинале)
	lastNails := make([]int, 0, 20)
	minDistance := nailCount / 20 // Минимальное расстояние между гвоздями
	if minDistance < 5 {
		minDistance = 5
	}

	// Тип для результата работы воркера
	type workerResult struct {
		nail      int
		lineError int // Сумма ошибки вдоль линии (чем больше, тем лучше)
	}

	// КРИТИЧЕСКАЯ ОПТИМИЗАЦИЯ: Ограничиваем кандидатов для скорости
	// Проверяем только часть гвоздей, но умно - ближайшие + случайные
	maxCandidates := 300 // Оптимальный баланс скорость/качество
	if nailCount < maxCandidates {
		maxCandidates = nailCount
	}

	for lineIdx := 0; lineIdx < lineCount; lineIdx++ {
		bestNail := currentNail
		var bestLine []int
		maxError := -1

		// Умный выбор кандидатов: ближайшие + равномерно распределенные
		candidates := make([]int, 0, maxCandidates)
		
		// Берем ближайшие гвозди (самые важные для качества)
		nearestCount := maxCandidates / 3
		for offset := minDistance; offset < nailCount-minDistance && len(candidates) < nearestCount; offset++ {
			nailIdx := (currentNail + offset) % nailCount
			if nailIdx != currentNail {
				candidates = append(candidates, nailIdx)
			}
		}
		
		// Добавляем равномерно распределенные гвозди
		step := nailCount / (maxCandidates - len(candidates))
		if step < 1 {
			step = 1
		}
		for i := 0; i < nailCount && len(candidates) < maxCandidates; i += step {
			if i != currentNail {
				// Проверяем, нет ли уже
				found := false
				for _, c := range candidates {
					if c == i {
						found = true
						break
					}
				}
				if !found {
					candidates = append(candidates, i)
				}
			}
		}

		// Быстрая проверка без worker pool для малого количества
		if len(candidates) < 50 {
			// Прямая проверка - быстрее чем worker pool overhead
			for _, nailIdx := range candidates {
				// Проверяем минимальное расстояние
				dist := nailIdx - currentNail
				if dist < 0 {
					dist = -dist
				}
				if dist > nailCount/2 {
					dist = nailCount - dist
				}
				if dist < minDistance {
					continue
				}

				// Проверяем недавно использованные
				recentlyUsed := false
				for _, lastNail := range lastNails {
					if nailIdx == lastNail {
						recentlyUsed = true
						break
					}
				}
				if recentlyUsed {
					continue
				}

				line := g.lineCache.Get(currentNail, nailIdx)
				if line == nil {
					continue
				}

				// Быстрое вычисление яркости
				lineLightness := 0
				count := 0
				for _, idx := range line {
					if idx >= 0 && idx < g.width*g.height {
						lineLightness += g.current[idx]
						count++
					}
				}

				if count > 0 {
					avgLightness := lineLightness / count
					lineError := 255 - avgLightness
					if lineError > maxError {
						maxError = lineError
						bestNail = nailIdx
						bestLine = line
					}
				}
			}
		} else {
			// Для большого количества - используем worker pool
			jobs := make(chan int, len(candidates))
			results := make(chan workerResult, len(candidates))

			var wg sync.WaitGroup
			currentNailLocal := currentNail

			for w := 0; w < g.workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for nailIdx := range jobs {
						// Проверяем минимальное расстояние
						dist := nailIdx - currentNailLocal
						if dist < 0 {
							dist = -dist
						}
						if dist > nailCount/2 {
							dist = nailCount - dist
						}
						if dist < minDistance {
							continue
						}

						line := g.lineCache.Get(currentNailLocal, nailIdx)
						if line == nil {
							continue
						}

						// Быстрое вычисление яркости
						lineLightness := 0
						count := 0
						for _, idx := range line {
							if idx >= 0 && idx < g.width*g.height {
								lineLightness += g.current[idx]
								count++
							}
						}

						if count > 0 {
							avgLightness := lineLightness / count
							lineError := 255 - avgLightness
							if lineError > 0 {
								results <- workerResult{
									nail:      nailIdx,
									lineError: lineError,
								}
							}
						}
					}
				}()
			}

			// Заполняем jobs
			for _, nailIdx := range candidates {
				jobs <- nailIdx
			}
			close(jobs)

			go func() {
				wg.Wait()
				close(results)
			}()

			// Собираем результаты
			for res := range results {
				// Проверяем недавно использованные
				recentlyUsed := false
				for _, lastNail := range lastNails {
					if res.nail == lastNail {
						recentlyUsed = true
						break
					}
				}
				if recentlyUsed {
					continue
				}

				if res.lineError > maxError {
					maxError = res.lineError
					bestNail = res.nail
					bestLine = g.lineCache.Get(currentNail, res.nail)
				}
			}
		}

		// Логируем каждые 10000 линий (реже для скорости)
		if lineIdx%10000 == 0 && lineIdx > 0 {
			log.Printf("🔍 [LINE %d] Найден лучший гвоздь: %d -> %d | Ошибка: %d",
				lineIdx+1, currentNail, bestNail, maxError)
		}

		// Если не нашли линию (fallback)
		if maxError < 0 || bestLine == nil {
			bestNail = (currentNail + 1) % nailCount
			bestLine = g.lineCache.Get(currentNail, bestNail)
		}

		// Обновляем current и error после рисования линии
		// ТОЧНО как в оригинале: error[v] = error[v] - LINE_WEIGHT
		for _, idx := range bestLine {
			if idx >= 0 && idx < g.width*g.height {
				// Обновляем current
				v := g.current[idx] + lineWeight
				if v > 255 {
					v = 255
				}
				g.current[idx] = v

				// Уменьшаем error на lineWeight (как в оригинальном Go коде)
				e := g.error[idx] - lineWeight
				if e < 0 {
					e = 0
				}
				g.error[idx] = e
			}
		}

		// Обновляем список последних использованных гвоздей
		lastNails = append(lastNails, bestNail)
		if len(lastNails) > 20 {
			lastNails = lastNails[1:] // Удаляем самый старый
		}

		segment := LineSegment{From: currentNail, To: bestNail}
		lines = append(lines, segment)
		currentNail = bestNail
	}

	return lines
}

