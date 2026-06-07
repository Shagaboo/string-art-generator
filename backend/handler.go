package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
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
	// Используем гамма = 0.7 для усиления темных областей (по умолчанию)
	gamma := req.Gamma
	if gamma <= 0 {
		gamma = 0.7
	}
	processor := NewImageProcessor(req.Brightness, req.Contrast, gamma)
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

// exportHandler обрабатывает запросы на экспорт в формат ЧПУ
func exportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Nails) == 0 || len(req.Lines) == 0 {
		http.Error(w, "No nails or lines to export", http.StatusBadRequest)
		return
	}

	format := strings.ToLower(req.Format)
	if format == "" {
		format = "json"
	}

	switch format {
	case "gcode":
		exportGCode(w, req)
	case "csv":
		exportCSV(w, req)
	case "json":
		exportJSON(w, req)
	default:
		http.Error(w, "Unsupported format. Use: gcode, csv, or json", http.StatusBadRequest)
	}
}

// exportGCode экспортирует в формат G-code для ЧПУ станка
func exportGCode(w http.ResponseWriter, req ExportRequest) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=string-art.gcode")

	// Заголовок G-code
	fmt.Fprintf(w, "; String Art G-code Export\n")
	fmt.Fprintf(w, "; Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "; Nails: %d, Lines: %d\n", len(req.Nails), len(req.Lines))
	fmt.Fprintf(w, "; Dimensions: %dx%d\n\n", req.Width, req.Height)

	// Инициализация
	fmt.Fprintf(w, "G21 ; Set units to millimeters\n")
	fmt.Fprintf(w, "G90 ; Absolute positioning\n")
	fmt.Fprintf(w, "G28 ; Home all axes\n")
	fmt.Fprintf(w, "G0 Z5 ; Raise tool\n\n")

	// Перемещаемся к первому гвоздю
	if len(req.Nails) > 0 {
		firstNail := req.Nails[req.Lines[0].From]
		fmt.Fprintf(w, "G0 X%.2f Y%.2f ; Move to first nail\n", firstNail.X, firstNail.Y)
		fmt.Fprintf(w, "G0 Z0 ; Lower tool\n\n")
	}

	// Рисуем линии
	for i, line := range req.Lines {
		fromNail := req.Nails[line.From]
		toNail := req.Nails[line.To]

		// Перемещаемся к началу линии (если нужно)
		if i == 0 || req.Lines[i-1].To != line.From {
			fmt.Fprintf(w, "G0 Z5 ; Raise tool\n")
			fmt.Fprintf(w, "G0 X%.2f Y%.2f ; Move to nail %d\n", fromNail.X, fromNail.Y, line.From)
			fmt.Fprintf(w, "G0 Z0 ; Lower tool\n")
		}

		// Рисуем линию
		fmt.Fprintf(w, "G1 X%.2f Y%.2f ; Draw line to nail %d\n", toNail.X, toNail.Y, line.To)
	}

	// Завершение
	fmt.Fprintf(w, "\nG0 Z5 ; Raise tool\n")
	fmt.Fprintf(w, "G28 ; Home all axes\n")
	fmt.Fprintf(w, "M30 ; End of program\n")
}

// exportCSV экспортирует в CSV формат (координаты гвоздей и последовательность линий)
func exportCSV(w http.ResponseWriter, req ExportRequest) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=string-art.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Заголовок
	writer.Write([]string{"Type", "Index", "X", "Y", "From", "To"})

	// Экспортируем гвозди
	for i, nail := range req.Nails {
		writer.Write([]string{
			"NAIL",
			strconv.Itoa(i),
			fmt.Sprintf("%.2f", nail.X),
			fmt.Sprintf("%.2f", nail.Y),
			"",
			"",
		})
	}

	// Экспортируем линии
	for i, line := range req.Lines {
		writer.Write([]string{
			"LINE",
			strconv.Itoa(i),
			"",
			"",
			strconv.Itoa(line.From),
			strconv.Itoa(line.To),
		})
	}
}

// exportJSON экспортирует в JSON формат с координатами
func exportJSON(w http.ResponseWriter, req ExportRequest) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=string-art.json")

	type ExportData struct {
		Metadata struct {
			GeneratedAt string `json:"generatedAt"`
			NailCount   int    `json:"nailCount"`
			LineCount   int    `json:"lineCount"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
		} `json:"metadata"`
		Nails []struct {
			Index int     `json:"index"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
		} `json:"nails"`
		Lines []struct {
			Index int `json:"index"`
			From  struct {
				Index int     `json:"index"`
				X     float64 `json:"x"`
				Y     float64 `json:"y"`
			} `json:"from"`
			To struct {
				Index int     `json:"index"`
				X     float64 `json:"x"`
				Y     float64 `json:"y"`
			} `json:"to"`
		} `json:"lines"`
	}

	var export ExportData
	export.Metadata.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")
	export.Metadata.NailCount = len(req.Nails)
	export.Metadata.LineCount = len(req.Lines)
	export.Metadata.Width = req.Width
	export.Metadata.Height = req.Height

	// Экспортируем гвозди
	export.Nails = make([]struct {
		Index int     `json:"index"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
	}, len(req.Nails))
	for i, nail := range req.Nails {
		export.Nails[i] = struct {
			Index int     `json:"index"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
		}{Index: i, X: nail.X, Y: nail.Y}
	}

	// Экспортируем линии с координатами
	export.Lines = make([]struct {
		Index int `json:"index"`
		From  struct {
			Index int     `json:"index"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
		} `json:"from"`
		To struct {
			Index int     `json:"index"`
			X     float64 `json:"x"`
			Y     float64 `json:"y"`
		} `json:"to"`
	}, len(req.Lines))
	for i, line := range req.Lines {
		fromNail := req.Nails[line.From]
		toNail := req.Nails[line.To]
		export.Lines[i].Index = i
		export.Lines[i].From.Index = line.From
		export.Lines[i].From.X = fromNail.X
		export.Lines[i].From.Y = fromNail.Y
		export.Lines[i].To.Index = line.To
		export.Lines[i].To.X = toNail.X
		export.Lines[i].To.Y = toNail.Y
	}

	json.NewEncoder(w).Encode(export)
}
