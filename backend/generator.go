package main

import (
	"runtime"
	"sync"
)

// Generator генерирует string art из изображения
// 1. Для текущего гвоздя проверяем ВСЕ остальные гвозди
// 2. Для каждого кандидата вычисляем среднюю яркость пикселей вдоль линии
// 3. Выбираем линию с минимальной средней яркостью (самую тёмную)
// 4. Прибавляем lineWeight к пикселям выбранной линии (делаем ярче)
// 5. Переходим к выбранному гвоздю и повторяем
type Generator struct {
	grayscale   []int  
	nails       []Nail
	width       int
	height      int
	workers     int
	currentNail int
}

// NewGenerator создает новый генератор
func NewGenerator(grayscale []int, nails []Nail, width, height int) *Generator {
	workers := runtime.NumCPU()
	if workers < 4 {
		workers = 4
	}
	if workers > 32 {
		workers = 32
	}

	return &Generator{
		grayscale:   grayscale,
		nails:       nails,
		width:       width,
		height:      height,
		workers:     workers,
		currentNail: 0,
	}
}

// bestResult хранит результат оценки одного воркера
type bestResult struct {
	nail       int
	brightness int // Средняя яркость (чем меньше, тем лучше)
	count      int // Количество пикселей в линии
}

// GenerateNextLine генерирует следующую линию string art
// - Перебирает ВСЕ гвозди (кроме текущего)
// - Вычисляет среднюю яркость (GetLineLightness)
// - Выбирает линию с минимальной яркостью
func (g *Generator) GenerateNextLine(lineWeight int) LineSegment {
	nailCount := len(g.nails)
	fromNail := g.currentNail
	fromX := g.nails[fromNail].Xi
	fromY := g.nails[fromNail].Yi

	// Параллельная оценка ВСЕХ гвоздей 
	chunkSize := (nailCount + g.workers - 1) / g.workers
	results := make([]bestResult, g.workers)

	var wg sync.WaitGroup
	for w := 0; w < g.workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > nailCount {
			end = nailCount
		}
		wg.Add(1)
		go func(start, end, idx int) {
			defer wg.Done()
			best := bestResult{nail: -1, brightness: 1 << 30}

			for j := start; j < end; j++ {
				if j == fromNail {
					continue
				}

				// Вычисляем среднюю яркость вдоль линии (inline Bresenham - без аллокации)
				sum, cnt := BresenhamBrightness(
					fromX, fromY,
					g.nails[j].Xi, g.nails[j].Yi,
					g.grayscale, g.width, g.height,
				)

				if cnt > 0 {
					avg := sum / cnt
					if avg < best.brightness {
						best = bestResult{nail: j, brightness: avg, count: cnt}
					}
				}
			}
			results[idx] = best
		}(start, end, w)
	}
	wg.Wait()

	// Находим лучший гвоздь (минимальная средняя яркость)
	bestNail := -1
	minBrightness := 1 << 30
	for _, r := range results {
		if r.nail >= 0 && r.brightness < minBrightness {
			minBrightness = r.brightness
			bestNail = r.nail
		}
	}

	// Fallback если не нашли подходящий гвоздь
	if bestNail < 0 {
		bestNail = (fromNail + 1) % nailCount
	}

	// Обновляем grayscale: прибавляем lineWeight к пикселям линии
	// Точно как в оригинале RemoveLine: pixels[index] = min(255, pixels[index] + lineWeight)
	BresenhamUpdate(
		fromX, fromY,
		g.nails[bestNail].Xi, g.nails[bestNail].Yi,
		g.grayscale, g.width, g.height, lineWeight,
	)

	g.currentNail = bestNail
	return LineSegment{From: fromNail, To: bestNail}
}
