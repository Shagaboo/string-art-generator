package main

// Nail представляет гвоздь с предвычисленными целочисленными координатами
type Nail struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Xi int     // Предвычисленная X координата (без Round в цикле)
	Yi int     // Предвычисленная Y координата
}

// LineSegment представляет сегмент линии между двумя гвоздями
type LineSegment struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// GenerateRequest содержит параметры запроса на генерацию
type GenerateRequest struct {
	ImageData        []uint8 `json:"imageData"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	Nails            []Nail  `json:"nails"`
	NailCount        int     `json:"nailCount"`
	LineCount        int     `json:"lineCount"`
	LineOpacity      float64 `json:"lineOpacity"`
	Shape            string  `json:"shape"`
	Brightness       float64 `json:"brightness"`
	Contrast         float64 `json:"contrast"`
	Gamma            float64 `json:"gamma"` // Гамма-коррекция для усиления темных областей (0.5-2.0, по умолчанию 0.7)
	InvertBrightness bool    `json:"invertBrightness"`
}

// GenerateResponse содержит результат генерации
type GenerateResponse struct {
	Lines []LineSegment `json:"lines"`
	Nails []Nail        `json:"nails"`
}

// ProgressUpdate содержит информацию о прогрессе генерации
type ProgressUpdate struct {
	Line  LineSegment `json:"line"`
	Index int         `json:"index"`
	Total int         `json:"total"`
}

// ExportRequest содержит данные для экспорта в формат ЧПУ
type ExportRequest struct {
	Nails []Nail        `json:"nails"`
	Lines []LineSegment `json:"lines"`
	Width int           `json:"width"`
	Height int          `json:"height"`
	Format string       `json:"format"` // "gcode", "json", "csv"
}


