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
	log.Printf("[GENERATE] Start - size: %dx%d, nails: %d, lines: %d",
		req.Width, req.Height, req.NailCount, req.LineCount)

	// Обработка изображения
	processor := NewImageProcessor(req.Brightness, req.Contrast)
	grayscale := processor.ProcessToGrayscale(
		req.ImageData,
		req.Width,
		req.Height,
		req.InvertBrightness,
	)

	// Генерация гвоздей
	nails := req.Nails
	if len(nails) == 0 {
		nails = GenerateNails(req.Width, req.Height, req.NailCount, req.Shape)
	} else {
		for i := range nails {
			nails[i].Xi = int(nails[i].X + 0.5)
			nails[i].Yi = int(nails[i].Y + 0.5)
		}
	}

	// lineWeight точно как в оригинале: LimitPixel(value / 100 * 255)
	lineWeight := int(math.Round(req.LineOpacity / 100.0 * 255.0))
	if lineWeight < 0 {
		lineWeight = 0
	}
	if lineWeight > 255 {
		lineWeight = 255
	}

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

	// Создаем генератор (без кэша линий - используем inline Bresenham)
	generator := NewGenerator(grayscale, nails, req.Width, req.Height)

	// Генерируем линии с потоковой отправкой прогресса
	lines := make([]LineSegment, 0, req.LineCount)

	// Batch size для flush
	batchSize := 200
	if req.LineCount > 10000 {
		batchSize = 500
	}
	if req.LineCount > 30000 {
		batchSize = 1000
	}

	log.Printf("[GENERATE] Starting generation of %d lines (workers: %d)...", req.LineCount, generator.workers)

	for lineIdx := 0; lineIdx < req.LineCount; lineIdx++ {
		segment := generator.GenerateNextLine(lineWeight)
		lines = append(lines, segment)

		// Логируем прогресс
		if lineIdx%5000 == 0 && lineIdx > 0 {
			elapsed := time.Since(startTime)
			linesPerSec := float64(lineIdx) / elapsed.Seconds()
			remaining := req.LineCount - lineIdx
			eta := time.Duration(float64(remaining)/linesPerSec) * time.Second
			log.Printf("[PROGRESS] Line %d/%d (%.1f%%) | Speed: %.0f lines/sec | ETA: %v",
				lineIdx, req.LineCount, float64(lineIdx)/float64(req.LineCount)*100,
				linesPerSec, eta.Round(time.Second))
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
	log.Printf("[GENERATE] Done in %v | Lines: %d | Speed: %.0f lines/sec",
		totalTime, len(lines), float64(len(lines))/totalTime.Seconds())

	response := GenerateResponse{
		Lines: lines,
		Nails: nails,
	}
	data, _ := json.Marshal(response)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
