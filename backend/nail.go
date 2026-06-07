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
	case "grid":
		nails = generateGridNails(width, height, count)
	default: // random
		nails = generateRandomNailsWithSpacing(width, height, count)
	}

	// Применяем проверку минимального расстояния для всех режимов
	nails = enforceMinDistance(nails, width, height, 3.0) // Минимум 3 пикселя между гвоздями

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

// generateGridNails генерирует гвозди в виде полу-сетки (по периметру + равномерно внутри)
func generateGridNails(width, height, count int) []Nail {
	nails := make([]Nail, 0, count)
	margin := 4.0
	
	// Вычисляем количество гвоздей для периметра (примерно 40%)
	perimeterCount := int(float64(count) * 0.4)
	innerCount := count - perimeterCount
	
	// Генерируем гвозди по периметру
	perimeter := 2.0*(float64(width)-2*margin) + 2.0*(float64(height)-2*margin)
	spacing := perimeter / float64(perimeterCount)
	
	pos := 0.0
	for i := 0; i < perimeterCount; i++ {
		var x, y float64
		if pos < float64(width)-2*margin {
			x = margin + pos
			y = margin
		} else if pos < float64(width)-2*margin+float64(height)-2*margin {
			x = float64(width) - margin
			y = margin + (pos - (float64(width) - 2*margin))
		} else if pos < 2*(float64(width)-2*margin)+float64(height)-2*margin {
			x = float64(width) - margin - (pos - (float64(width)-2*margin+float64(height)-2*margin))
			y = float64(height) - margin
		} else {
			x = margin
			y = float64(height) - margin - (pos - (2*(float64(width)-2*margin)+float64(height)-2*margin))
		}
		nails = append(nails, Nail{X: x, Y: y})
		pos += spacing
	}
	
	// Генерируем гвозди внутри равномерно (сетка с небольшим случайным смещением)
	cols := int(math.Sqrt(float64(innerCount) * float64(width) / float64(height)))
	rows := (innerCount + cols - 1) / cols
	
	cellW := (float64(width) - 2*margin) / float64(cols)
	cellH := (float64(height) - 2*margin) / float64(rows)
	
	for i := 0; i < innerCount; i++ {
		col := i % cols
		row := i / cols
		
		// Центр ячейки с небольшим случайным смещением
		centerX := margin + float64(col)*cellW + cellW/2
		centerY := margin + float64(row)*cellH + cellH/2
		
		// Случайное смещение до 30% размера ячейки
		offsetX := (rand.Float64() - 0.5) * cellW * 0.3
		offsetY := (rand.Float64() - 0.5) * cellH * 0.3
		
		nails = append(nails, Nail{
			X: math.Max(margin, math.Min(float64(width)-margin, centerX+offsetX)),
			Y: math.Max(margin, math.Min(float64(height)-margin, centerY+offsetY)),
		})
	}
	
	return nails
}

// generateRandomNailsWithSpacing генерирует случайные позиции гвоздей с проверкой расстояния
func generateRandomNailsWithSpacing(width, height, count int) []Nail {
	nails := make([]Nail, 0, count)
	minDist := 5.0 // Минимальное расстояние между гвоздями
	maxAttempts := 100
	
	for i := 0; i < count; i++ {
		attempts := 0
		var x, y float64
		valid := false
		
		for !valid && attempts < maxAttempts {
			x = rand.Float64() * float64(width)
			y = rand.Float64() * float64(height)
			
			valid = true
			for _, nail := range nails {
				dx := x - nail.X
				dy := y - nail.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < minDist {
					valid = false
					break
				}
			}
			attempts++
		}
		
		nails = append(nails, Nail{X: x, Y: y})
	}
	
	return nails
}

// enforceMinDistance применяет проверку минимального расстояния между гвоздями
func enforceMinDistance(nails []Nail, width, height int, minDist float64) []Nail {
	if len(nails) == 0 {
		return nails
	}
	
	result := make([]Nail, 0, len(nails))
	result = append(result, nails[0]) // Первый гвоздь всегда добавляем
	
	for i := 1; i < len(nails); i++ {
		nail := nails[i]
		tooClose := false
		
		// Проверяем расстояние до всех уже добавленных гвоздей
		for _, existing := range result {
			dx := nail.X - existing.X
			dy := nail.Y - existing.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			
			if dist < minDist {
				tooClose = true
				// Пытаемся сдвинуть гвоздь
				if dist > 0 {
					shiftX := (dx / dist) * minDist
					shiftY := (dy / dist) * minDist
					nail.X = existing.X + shiftX
					nail.Y = existing.Y + shiftY
					
					// Проверяем границы
					nail.X = math.Max(2.0, math.Min(float64(width)-2.0, nail.X))
					nail.Y = math.Max(2.0, math.Min(float64(height)-2.0, nail.Y))
				}
				break
			}
		}
		
		// Если гвоздь слишком близко, пытаемся найти новую позицию
		if tooClose {
			// Пробуем найти свободное место рядом
			found := false
			for attempt := 0; attempt < 10; attempt++ {
				angle := rand.Float64() * 2 * math.Pi
				newX := nail.X + math.Cos(angle)*minDist*1.5
				newY := nail.Y + math.Sin(angle)*minDist*1.5
				
				if newX >= 2 && newX < float64(width)-2 && newY >= 2 && newY < float64(height)-2 {
					valid := true
					for _, existing := range result {
						dx := newX - existing.X
						dy := newY - existing.Y
						dist := math.Sqrt(dx*dx + dy*dy)
						if dist < minDist {
							valid = false
							break
						}
					}
					if valid {
						nail.X = newX
						nail.Y = newY
						found = true
						break
					}
				}
			}
			if !found {
				// Если не нашли место, пропускаем этот гвоздь
				continue
			}
		}
		
		result = append(result, nail)
	}
	
	return result
}


