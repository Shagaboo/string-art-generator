package main

// LineCache кэширует линии между всеми парами гвоздей
type LineCache struct {
	cache [][][]int
	width int
}

// NewLineCache создает новый кэш линий для всех пар гвоздей
func NewLineCache(nails []Nail, width int) *LineCache {
	nailCount := len(nails)
	cache := make([][][]int, nailCount)
	
	for i := 0; i < nailCount; i++ {
		cache[i] = make([][]int, nailCount)
		for j := 0; j < nailCount; j++ {
			if i != j {
				cache[i][j] = GetLinePixels(
					nails[i].Xi,
					nails[i].Yi,
					nails[j].Xi,
					nails[j].Yi,
					width,
				)
			}
		}
	}

	return &LineCache{
		cache: cache,
		width: width,
	}
}

// Get возвращает линию между двумя гвоздями из кэша
func (lc *LineCache) Get(from, to int) []int {
	if from < 0 || from >= len(lc.cache) || to < 0 || to >= len(lc.cache) {
		return nil
	}
	return lc.cache[from][to]
}

// GetLinePixels использует алгоритм Bresenham для получения всех пикселей линии
func GetLinePixels(x0, y0, x1, y1, width int) []int {
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
	estimatedLen := dx + dy + 1
	if estimatedLen < 0 {
		estimatedLen = 100 // Fallback
	}
	pixels := make([]int, 0, estimatedLen)

	cx, cy := x0, y0

	for {
		idx := cy*width + cx
		pixels = append(pixels, idx)

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

	return pixels
}

