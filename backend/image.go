package main

import "math"

// ImageProcessor обрабатывает изображения: конвертация в grayscale, применение brightness/contrast
// ТОЧНО как в оригинале: используем целые числа и правильный порядок операций
type ImageProcessor struct {
	brightnessTable []int
	contrastTable   []int
}

// limitPixel ограничивает значение пикселя 0-255 и округляет (как LimitPixel в оригинале)
func limitPixel(value float64) int {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return int(math.Round(value))
}

// NewImageProcessor создает новый процессор изображений с заданными параметрами
// ТОЧНО как в оригинале: brightnessTable[i] = LimitPixel(i * brightness)
func NewImageProcessor(brightness, contrast float64) *ImageProcessor {
	ip := &ImageProcessor{
		brightnessTable: make([]int, 256),
		contrastTable:   make([]int, 256),
	}

	// Создаем таблицу brightness ТОЧНО как в оригинале
	brightnessFactor := 1.0 + brightness/100.0
	for i := 0; i < 256; i++ {
		ip.brightnessTable[i] = limitPixel(float64(i) * brightnessFactor)
	}

	// Создаем таблицу contrast ТОЧНО как в оригинале
	contrastFactor := 1.0 + contrast/100.0
	for i := 0; i < 256; i++ {
		ip.contrastTable[i] = limitPixel((float64(i) - 128.0) * contrastFactor + 128.0)
	}

	return ip
}

// ProcessToGrayscale конвертирует RGBA изображение в grayscale с применением настроек
// ТОЧНО как в оригинале: GetLightness -> invert -> brightnessTable -> contrastTable
func (ip *ImageProcessor) ProcessToGrayscale(imageData []uint8, width, height int, invert bool) []int {
	grayscale := make([]int, width*height)

	for i := 0; i < len(imageData); i += 4 {
		r := float64(imageData[i])
		g := float64(imageData[i+1])
		b := float64(imageData[i+2])

		// GetLightness: Math.floor(0.2126 * red + 0.7152 * green + 0.0722 * blue)
		lightness := int(math.Floor(0.2126*r + 0.7152*g + 0.0722*b))
		if lightness < 0 {
			lightness = 0
		}
		if lightness > 255 {
			lightness = 255
		}

		// Применяем инверсию ДО brightness/contrast (как в оригинале)
		if invert {
			lightness = 255 - lightness
		}

		// Применяем brightness и contrast через таблицы (ТОЧНО как в оригинале)
		lightness = ip.brightnessTable[lightness]
		lightness = ip.contrastTable[lightness]

		grayscale[i/4] = lightness
	}

	return grayscale
}

