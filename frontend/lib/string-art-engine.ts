export interface StringArtSettings {
  shape: "circle" | "rectangle" | "grid" | "random"
  invertBrightness: boolean
  contrast: number
  brightness: number
  gamma: number // Гамма-коррекция для усиления темных областей (0.5-2.0)
  nailCount: number
  lineCount: number
  lineOpacity: number
  lineColor: string
  backgroundColor: string
}

export interface Nail {
  x: number
  y: number
}

export interface LineSegment {
  from: number
  to: number
}

export const DEFAULT_SETTINGS: StringArtSettings = {
  shape: "grid",
  invertBrightness: false,
  contrast: 0,
  brightness: 0,
  gamma: 0.7, // По умолчанию 0.7 для усиления темных областей
  nailCount: 2500,
  lineCount: 30000,
  lineOpacity: 10,
  lineColor: "#ffffff",
  backgroundColor: "#000000",
}

export function generateNails(
  width: number,
  height: number,
  count: number,
  shape: StringArtSettings["shape"]
): Nail[] {
  const nails: Nail[] = []
  const cx = width / 2
  const cy = height / 2

  if (shape === "circle") {
    const radius = Math.min(width, height) / 2 - 4
    for (let i = 0; i < count; i++) {
      const angle = (2 * Math.PI * i) / count
      nails.push({
        x: cx + radius * Math.cos(angle),
        y: cy + radius * Math.sin(angle),
      })
    }
  } else if (shape === "rectangle") {
    const margin = 4
    const perimeter = 2 * (width - 2 * margin) + 2 * (height - 2 * margin)
    const step = perimeter / count

    for (let i = 0; i < count; i++) {
      let d = (i * step) % perimeter
      let x: number, y: number

      if (d < width - 2 * margin) {
        x = margin + d
        y = margin
      } else {
        d -= width - 2 * margin
        if (d < height - 2 * margin) {
          x = width - margin
          y = margin + d
        } else {
          d -= height - 2 * margin
          if (d < width - 2 * margin) {
            x = width - margin - d
            y = height - margin
          } else {
            d -= width - 2 * margin
            x = margin
            y = height - margin - d
          }
        }
      }
      nails.push({ x, y })
    }
  } else if (shape === "grid") {
    // Полу-сетка: периметр + равномерное распределение внутри
    const margin = 4
    const perimeterCount = Math.floor(count * 0.4)
    const innerCount = count - perimeterCount
    
    // Гвозди по периметру
    const perimeter = 2 * (width - 2 * margin) + 2 * (height - 2 * margin)
    const spacing = perimeter / perimeterCount
    let pos = 0
    
    for (let i = 0; i < perimeterCount; i++) {
      let x: number, y: number
      if (pos < width - 2 * margin) {
        x = margin + pos
        y = margin
      } else if (pos < width - 2 * margin + height - 2 * margin) {
        x = width - margin
        y = margin + (pos - (width - 2 * margin))
      } else if (pos < 2 * (width - 2 * margin) + height - 2 * margin) {
        x = width - margin - (pos - (width - 2 * margin + height - 2 * margin))
        y = height - margin
      } else {
        x = margin
        y = height - margin - (pos - (2 * (width - 2 * margin) + height - 2 * margin))
      }
      nails.push({ x, y })
      pos += spacing
    }
    
    // Гвозди внутри (сетка с небольшим случайным смещением)
    const cols = Math.floor(Math.sqrt(innerCount * width / height))
    const rows = Math.ceil(innerCount / cols)
    const cellW = (width - 2 * margin) / cols
    const cellH = (height - 2 * margin) / rows
    
    for (let i = 0; i < innerCount; i++) {
      const col = i % cols
      const row = Math.floor(i / cols)
      const centerX = margin + col * cellW + cellW / 2
      const centerY = margin + row * cellH + cellH / 2
      const offsetX = (Math.random() - 0.5) * cellW * 0.3
      const offsetY = (Math.random() - 0.5) * cellH * 0.3
      nails.push({
        x: Math.max(margin, Math.min(width - margin, centerX + offsetX)),
        y: Math.max(margin, Math.min(height - margin, centerY + offsetY)),
      })
    }
  } else {
    // random
    for (let i = 0; i < count; i++) {
      nails.push({
        x: Math.random() * (width - 8) + 4,
        y: Math.random() * (height - 8) + 4,
      })
    }
  }

  return nails
}

export interface GeneratorResult {
  lines: LineSegment[]
  nails: Nail[]
}

// Get line pixels as Set of indices (like in original)
function getLinePixelsSet(
  x0: number,
  y0: number,
  x1: number,
  y1: number,
  width: number
): Set<number> {
  const line = new Set<number>()
  let x1_work = x0
  let y1_work = y0
  const x2 = x1
  const y2 = y1
  
  const delta_x = Math.abs(x2 - x1_work)
  const delta_y = Math.abs(y2 - y1_work)
  const sign_x = Math.sign(x2 - x1_work)
  const sign_y = Math.sign(y2 - y1_work)
  let error = delta_x - delta_y

  while (x1_work !== x2 || y1_work !== y2) {
    line.add(y1_work * width + x1_work)
    const error2 = error * 2

    if (error2 > -delta_y) {
      error -= delta_y
      x1_work += sign_x
    }

    if (error2 < delta_x) {
      error += delta_x
      y1_work += sign_y
    }
  }

  line.add(y2 * width + x2)
  return line
}

export function* stringArtGenerator(
  sourceImageData: ImageData,
  nails: Nail[],
  settings: StringArtSettings
): Generator<LineSegment, GeneratorResult, undefined> {
  const width = sourceImageData.width
  const height = sourceImageData.height
  const lines: LineSegment[] = []

  // Create a copy of brightness values as a working buffer
  const grayscale = new Float32Array(width * height)
  const data = sourceImageData.data

  // ТОЧНОЕ применение brightness/contrast/inversion как в оригинале
  // В оригинале используются таблицы brightnessTable и contrastTable (строки 64-65, 75-76 init.js)
  // brightnessTable[i] = LimitPixel(i * brightness), где brightness = 1 + value / 100
  // contrastTable[i] = LimitPixel((i - 128) * contrast + 128), где contrast = 1 + value / 100
  
  // Создаем таблицы как в оригинале
  const brightnessTable: number[] = []
  const brightness = 1 + settings.brightness / 100
  for (let i = 0; i < 256; i++) {
    brightnessTable[i] = Math.max(0, Math.min(255, Math.round(i * brightness)))
  }
  
  const contrastTable: number[] = []
  const contrast = 1 + settings.contrast / 100
  for (let i = 0; i < 256; i++) {
    contrastTable[i] = Math.max(0, Math.min(255, Math.round((i - 128) * contrast + 128)))
  }

  // ТОЧНАЯ формула из оригинала (строка 6 draw.js): 0.2126 * red + 0.7152 * green + 0.0722 * blue
  for (let i = 0; i < data.length; i += 4) {
    let gray = Math.floor(0.2126 * data[i] + 0.7152 * data[i + 1] + 0.0722 * data[i + 2])
    
    // Применяем инверсию ДО brightness/contrast (как в оригинале, строка 58-59)
    if (settings.invertBrightness) {
      gray = 255 - gray
    }
    
    // Применяем brightness и contrast через таблицы (как в оригинале, строки 61-62)
    gray = brightnessTable[gray]
    gray = contrastTable[gray]
    
    grayscale[i / 4] = gray
  }

  // ТОЧНЫЙ алгоритм из оригинала - поиск минимальной яркости
  // В оригинале lineWeight вычисляется как: lineWeight = lineOpacity / 100 * 255 (строка 172)
  // Но затем умножается на dpr (device pixel ratio) при обновлении пикселей (строка 131)
  // У нас dpr = 1, поэтому просто используем lineWeight напрямую
  const lineWeight = Math.round((settings.lineOpacity / 100) * 255)
  
  let currentNail = 0

  for (let lineIdx = 0; lineIdx < settings.lineCount; lineIdx++) {
    let bestNail = currentNail
    let bestLine: Set<number> | null = null
    let minLightness = Infinity

    // ТОЧНЫЙ алгоритм из оригинала - проверяем ВСЕ гвозди (строка 109)
    // Ищем линию с минимальной яркостью (самую темную область)
    let foundAny = false
    
    for (let i = 0; i < nails.length; i++) {
      if (i === currentNail) continue

      // В оригинале порядок: LineRasterization(this.nails[i].x, this.nails[i].y, this.nails[nail].x, this.nails[nail].y)
      // То есть от nails[i] к nails[nail] (от i к currentNail)
      const line = getLinePixelsSet(
        Math.round(nails[i].x),
        Math.round(nails[i].y),
        Math.round(nails[currentNail].x),
        Math.round(nails[currentNail].y),
        width
      )

      // Вычисляем среднюю яркость линии, как в оригинале (строки 95-102)
      let lightness = 0
      let count = 0
      for (const index of line) {
        if (index >= 0 && index < width * height) {
          lightness += grayscale[index]
          count++
        }
      }

      if (count > 0) {
        const avgLightness = lightness / count
        
        // Инициализируем первым найденным гвоздем (на случай если все яркости одинаковые)
        if (!foundAny) {
          foundAny = true
          bestNail = i
          bestLine = line
          minLightness = avgLightness
        } else if (avgLightness < minLightness) {
          minLightness = avgLightness
          bestNail = i
          bestLine = line
        }
      }
    }

    // Если не нашли ни одной линии (не должно происходить, но на всякий случай)
    if (!foundAny || bestLine === null) {
      bestNail = (currentNail + 1) % nails.length
      bestLine = getLinePixelsSet(
        Math.round(nails[bestNail].x),
        Math.round(nails[bestNail].y),
        Math.round(nails[currentNail].x),
        Math.round(nails[currentNail].y),
        width
      )
    }

    // "Remove" the line from the source image by adding brightness
    // Как в оригинале (строка 131): pixels[index] = Math.min(255, pixels[index] + lineWeight * dpr)
    // Здесь dpr = 1, поэтому просто lineWeight
    // ВАЖНО: обновляем grayscale ПОСЛЕ выбора линии, чтобы следующая итерация использовала обновленные значения
    for (const index of bestLine) {
      if (index >= 0 && index < width * height) {
        grayscale[index] = Math.min(255, grayscale[index] + lineWeight)
      }
    }

    const segment: LineSegment = { from: currentNail, to: bestNail }
    lines.push(segment)
    currentNail = bestNail // Continue from the end of the line - ensures continuous line

    yield segment
  }

  return { lines, nails }
}

export function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : { r: 0, g: 0, b: 0 }
}
