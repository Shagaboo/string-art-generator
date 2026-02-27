package main

import (
	"math"
	"math/rand"
)

// GenerateNails генерирует позиции гвоздей в зависимости от формы
func GenerateNails(width, height, count int, shape string) []Nail {
	nails := make([]Nail, 0, count)

	switch shape {
	case "circle":
		nails = generateCircleNails(width, height, count)
	case "rectangle":
		nails = generateRectangleNails(width, height, count)
	default: // random
		nails = generateRandomNails(width, height, count)
	}

	// Предвычисляем целочисленные координаты для оптимизации
	for i := range nails {
		nails[i].Xi = int(math.Round(nails[i].X))
		nails[i].Yi = int(math.Round(nails[i].Y))
	}

	return nails
}

// generateCircleNails генерирует гвозди по кругу
func generateCircleNails(width, height, count int) []Nail {
	nails := make([]Nail, 0, count)
	centerX := float64(width) / 2
	centerY := float64(height) / 2
	radius := math.Min(float64(width), float64(height)) / 2 * 0.9

	for i := 0; i < count; i++ {
		angle := 2 * math.Pi * float64(i) / float64(count)
		nails = append(nails, Nail{
			X: centerX + radius*math.Cos(angle),
			Y: centerY + radius*math.Sin(angle),
		})
	}

	return nails
}

// generateRectangleNails генерирует гвозди по периметру прямоугольника
func generateRectangleNails(width, height, count int) []Nail {
	nails := make([]Nail, 0, count)
	perimeter := 2*float64(width) + 2*float64(height)
	spacing := perimeter / float64(count)

	pos := 0.0
	for i := 0; i < count; i++ {
		if pos < float64(width) {
			nails = append(nails, Nail{X: pos, Y: 0})
		} else if pos < float64(width+height) {
			nails = append(nails, Nail{X: float64(width), Y: pos - float64(width)})
		} else if pos < float64(2*width+height) {
			nails = append(nails, Nail{X: float64(width) - (pos - float64(width+height)), Y: float64(height)})
		} else {
			nails = append(nails, Nail{X: 0, Y: float64(height) - (pos - float64(2*width+height))})
		}
		pos += spacing
	}

	return nails
}

// generateRandomNails генерирует случайные позиции гвоздей
func generateRandomNails(width, height, count int) []Nail {
	nails := make([]Nail, 0, count)
	for i := 0; i < count; i++ {
		nails = append(nails, Nail{
			X: rand.Float64() * float64(width),
			Y: rand.Float64() * float64(height),
		})
	}
	return nails
}

