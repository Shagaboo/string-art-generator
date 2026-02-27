package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

// healthHandler обрабатывает запросы на проверку здоровья сервера
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Добавляем CORS заголовки
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// generateHandler обрабатывает запросы на генерацию string art
func generateHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	startTime := time.Now()
	log.Printf("🚀 [GENERATE] Начало генерации - размер: %dx%d, гвоздей: %d, линий: %d", 
		req.Width, req.Height, req.NailCount, req.LineCount)
	log.Printf("📊 [GENERATE] Параметры: brightness=%.1f, contrast=%.1f, invert=%v, opacity=%.1f", 
		req.Brightness, req.Contrast, req.InvertBrightness, req.LineOpacity)

	// Обработка изображения
	log.Printf("🖼️  [IMAGE] Обработка изображения...")
	processor := NewImageProcessor(req.Brightness, req.Contrast)
	grayscale := processor.ProcessToGrayscale(
		req.ImageData,
		req.Width,
		req.Height,
		req.InvertBrightness,
	)
	log.Printf("✅ [IMAGE] Изображение обработано, пикселей: %d", len(grayscale))

	// Генерация гвоздей
	nails := req.Nails
	if len(nails) == 0 {
		nails = GenerateNails(req.Width, req.Height, req.NailCount, req.Shape)
	} else {
		// Предвычисляем целочисленные координаты если гвозди пришли из запроса
		for i := range nails {
			nails[i].Xi = int(nails[i].X + 0.5) // Быстрый round
			nails[i].Yi = int(nails[i].Y + 0.5)
		}
	}

	// GetLineWeight ТОЧНО как в оригинале: LimitPixel(value / 100 * 255)
	lineWeight := int(math.Round(req.LineOpacity / 100.0 * 255.0))
	if lineWeight < 0 {
		lineWeight = 0
	}
	if lineWeight > 255 {
		lineWeight = 255
	}

	// Создаем кэш линий
	log.Printf("🔧 [CACHE] Создание кэша линий для %d гвоздей...", len(nails))
	cacheStart := time.Now()
	lineCache := NewLineCache(nails, req.Width)
	log.Printf("✅ [CACHE] Кэш создан за %v", time.Since(cacheStart))

	// Setup SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Создаем генератор
	generator := NewGenerator(grayscale, lineCache, nails, req.Width, req.Height)

	// Генерируем линии с потоковой отправкой прогресса
	lines := make([]LineSegment, 0, req.LineCount)
	currentNail := 0

	// Определяем batch size для flush (максимально для скорости)
	batchSize := 200
	if req.LineCount > 10000 {
		batchSize = 500
	}
	if req.LineCount > 30000 {
		batchSize = 1000
	}

	// Генерируем по одной линии для потоковой отправки
	log.Printf("🎨 [GENERATE] Начало генерации %d линий...", req.LineCount)
	lastLogTime := time.Now()
	
	for lineIdx := 0; lineIdx < req.LineCount; lineIdx++ {
		segment := generator.generateNextLine(&currentNail, lineWeight)
		lines = append(lines, segment)
		
		// Логируем прогресс каждые 10000 линий или каждые 20 секунд (реже для скорости)
		now := time.Now()
		if lineIdx%10000 == 0 || now.Sub(lastLogTime) >= 20*time.Second {
			elapsed := now.Sub(startTime)
			linesPerSec := float64(lineIdx+1) / elapsed.Seconds()
			remaining := req.LineCount - (lineIdx + 1)
			eta := time.Duration(float64(remaining)/linesPerSec) * time.Second
			log.Printf("📈 [PROGRESS] Линия %d/%d (%.1f%%) | Скорость: %.1f лин/сек | Осталось: ~%v", 
				lineIdx+1, req.LineCount, float64(lineIdx+1)/float64(req.LineCount)*100, 
				linesPerSec, eta.Round(time.Second))
			lastLogTime = now
		}

		// Отправляем прогресс
		update := ProgressUpdate{
			Line:  segment,
			Index: lineIdx + 1,
			Total: req.LineCount,
		}
		data, _ := json.Marshal(update)
		fmt.Fprintf(w, "data: %s\n\n", data)

		// Flush батчами
		if lineIdx%batchSize == 0 || lineIdx == req.LineCount-1 {
			flusher.Flush()
		}
	}

	// Финальный ответ
	totalTime := time.Since(startTime)
	log.Printf("✅ [GENERATE] Генерация завершена за %v | Всего линий: %d | Средняя скорость: %.1f лин/сек", 
		totalTime, len(lines), float64(len(lines))/totalTime.Seconds())
	
	response := GenerateResponse{
		Lines: lines,
		Nails: nails,
	}
	data, _ := json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// generateNextLine генерирует следующую линию используя ERROR-BASED алгоритм
func (g *Generator) generateNextLine(currentNail *int, lineWeight int) LineSegment {
	nailCount := len(g.nails)
	bestNail := *currentNail
	var bestLine []int
	maxError := -1

	// КРИТИЧЕСКАЯ ОПТИМИЗАЦИЯ: Ограничиваем кандидатов для скорости
	maxCandidates := 300
	if nailCount < maxCandidates {
		maxCandidates = nailCount
	}

	// Умный выбор кандидатов: ближайшие + равномерно распределенные
	candidates := make([]int, 0, maxCandidates)
	minDistance := nailCount / 20
	if minDistance < 5 {
		minDistance = 5
	}

	// Берем ближайшие гвозди
	nearestCount := maxCandidates / 3
	for offset := minDistance; offset < nailCount-minDistance && len(candidates) < nearestCount; offset++ {
		nailIdx := (*currentNail + offset) % nailCount
		if nailIdx != *currentNail {
			candidates = append(candidates, nailIdx)
		}
	}

	// Добавляем равномерно распределенные
	step := nailCount / (maxCandidates - len(candidates))
	if step < 1 {
		step = 1
	}
	for i := 0; i < nailCount && len(candidates) < maxCandidates; i += step {
		if i != *currentNail {
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

	// Быстрая прямая проверка (без worker pool overhead для скорости)
	for _, nailIdx := range candidates {
		line := g.lineCache.Get(*currentNail, nailIdx)
		if line == nil {
			continue
		}

		// Быстрое вычисление яркости (оригинальный алгоритм)
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
	// Fallback если не нашли
	if maxError < 0 || bestLine == nil {
		bestNail = (*currentNail + 1) % nailCount
		bestLine = g.lineCache.Get(*currentNail, bestNail)
	}

	// Обновляем current и error после рисования линии
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

	fromNail := *currentNail
	*currentNail = bestNail
	return LineSegment{From: fromNail, To: bestNail}
}

