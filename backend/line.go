package main

// BresenhamBrightness вычисляет сумму яркости и количество пикселей вдоль линии
// используя алгоритм Брезенхема без выделения памяти (inline)
func BresenhamBrightness(x0, y0, x1, y1 int, grayscale []int, width, height int) (sum int, count int) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy
	cx, cy := x0, y0
	size := width * height

	for {
		if cx >= 0 && cx < width && cy >= 0 && cy < height {
			idx := cy*width + cx
			if idx >= 0 && idx < size {
				sum += grayscale[idx]
				count++
			}
		}
		if cx == x1 && cy == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			cx += sx
		}
		if e2 < dx {
			err += dx
			cy += sy
		}
	}
	return
}

// BresenhamUpdate прибавляет lineWeight к пикселям вдоль линии (с ограничением до 255)
// Это "удаляет" линию из изображения, делая пиксели ярче
func BresenhamUpdate(x0, y0, x1, y1 int, grayscale []int, width, height, lineWeight int) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}

	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy
	cx, cy := x0, y0
	size := width * height

	for {
		if cx >= 0 && cx < width && cy >= 0 && cy < height {
			idx := cy*width + cx
			if idx >= 0 && idx < size {
				v := grayscale[idx] + lineWeight
				if v > 255 {
					v = 255
				}
				grayscale[idx] = v
			}
		}
		if cx == x1 && cy == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			cx += sx
		}
		if e2 < dx {
			err += dx
			cy += sy
		}
	}
}
