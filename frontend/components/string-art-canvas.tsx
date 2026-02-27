"use client"

import { useRef, useEffect, useCallback, forwardRef, useImperativeHandle, useState } from "react"
import type { Nail, StringArtSettings, LineSegment } from "@/lib/string-art-engine"
import { generateNails, stringArtGenerator, hexToRgb } from "@/lib/string-art-engine"
import { generateStringArtWithGoBackend, checkGoBackendHealth } from "@/lib/go-backend-client"

export interface StringArtCanvasHandle {
  start: (image: HTMLImageElement) => void
  stop: () => void
  reset: () => void
  downloadPNG: () => void
  downloadSVG: () => void
  getLines: () => LineSegment[]
  getNails: () => Nail[]
}

interface StringArtCanvasProps {
  settings: StringArtSettings
  onProgress: (current: number, total: number) => void
  onComplete: () => void
  onRunningChange: (running: boolean) => void
}

const StringArtCanvas = forwardRef<StringArtCanvasHandle, StringArtCanvasProps>(
  ({ settings, onProgress, onComplete, onRunningChange }, ref) => {
    const canvasRef = useRef<HTMLCanvasElement>(null)
    const sourceCanvasRef = useRef<HTMLCanvasElement>(null)
    const animFrameRef = useRef<number>(0)
    const generatorRef = useRef<Generator<LineSegment, { lines: LineSegment[]; nails: Nail[] }, undefined> | null>(null)
    const asyncGeneratorRef = useRef<AsyncGenerator<LineSegment, any, undefined> | null>(null)
    const nailsRef = useRef<Nail[]>([])
    const linesRef = useRef<LineSegment[]>([])
    const runningRef = useRef(false)
    const canvasSizeRef = useRef({ width: 0, height: 0 })
    const [useGoBackend, setUseGoBackend] = useState(true)

    const drawNails = useCallback(
      (ctx: CanvasRenderingContext2D, nails: Nail[]) => {
        const rgb = hexToRgb(settings.lineColor)
        ctx.fillStyle = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.4)`
        for (const nail of nails) {
          ctx.beginPath()
          ctx.arc(nail.x, nail.y, 2, 0, Math.PI * 2)
          ctx.fill()
        }
      },
      [settings.lineColor]
    )

    const drawLine = useCallback(
      (
        ctx: CanvasRenderingContext2D,
        nails: Nail[],
        segment: LineSegment
      ) => {
        // Используем rgba формат для надежной совместимости
        // В оригинале используется hex с альфа, но rgba работает везде одинаково
        const rgb = hexToRgb(settings.lineColor)
        // Прозрачность: lineOpacity / 100 (как в оригинале lineWeight / 100)
        const alpha = settings.lineOpacity / 100
        
        ctx.strokeStyle = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`
        ctx.lineWidth = 1
        ctx.lineCap = "round"
        ctx.lineJoin = "round"
        ctx.beginPath()
        ctx.moveTo(nails[segment.from].x, nails[segment.from].y)
        ctx.lineTo(nails[segment.to].x, nails[segment.to].y)
        ctx.stroke()
      },
      [settings.lineColor, settings.lineOpacity]
    )

    const stop = useCallback(() => {
      runningRef.current = false
      onRunningChange(false)
      if (animFrameRef.current) {
        cancelAnimationFrame(animFrameRef.current)
        animFrameRef.current = 0
      }
    }, [onRunningChange])

    const reset = useCallback(() => {
      stop()
      generatorRef.current = null
      asyncGeneratorRef.current = null
      linesRef.current = []
      nailsRef.current = []
      const canvas = canvasRef.current
      if (canvas) {
        const ctx = canvas.getContext("2d")
        if (ctx) {
          ctx.clearRect(0, 0, canvas.width, canvas.height)
        }
      }
    }, [stop])

    // Check Go backend availability on mount
    useEffect(() => {
      checkGoBackendHealth().then((available) => {
        setUseGoBackend(available)
        if (!available) {
          console.warn("Go backend not available, using local generator")
        }
      })
    }, [])

    const start = useCallback(
      (image: HTMLImageElement) => {
        reset()

        const canvas = canvasRef.current
        const sourceCanvas = sourceCanvasRef.current
        if (!canvas || !sourceCanvas) return

        // Determine canvas size based on image while fitting container
        const container = canvas.parentElement
        const containerW = container ? container.clientWidth : 2000
        const containerH = container ? container.clientHeight : 1500
        
        // On mobile, ensure canvas fits both width and height with padding
        const isMobile = containerW < 768
        const isTablet = containerW >= 768 && containerW < 1024
        const padding = isMobile ? 8 : 16 // Уменьшили padding для экономии места
        const maxW = containerW - padding
        const maxH = containerH - padding - 40 // Дополнительный отступ сверху для UI
        
        let w = image.naturalWidth
        let h = image.naturalHeight

        // Calculate scale to fit both dimensions
        // Увеличиваем размер в 1.5 раза минимум, но НЕ выходим за рамки контейнера
        const minScale = isMobile ? 1.2 : isTablet ? 1.5 : 1.5 // Минимум 1.5x на десктопе
        const maxScale = isMobile ? 2 : isTablet ? 3 : 4 // Максимум 4x на десктопе
        
        // Вычисляем масштаб для вписывания в контейнер (приоритет высоте)
        const scaleW = maxW / w
        const scaleH = maxH / h
        // Используем минимальный масштаб, чтобы вписаться в оба измерения
        const calculatedScale = Math.min(scaleW, scaleH)
        
        // Применяем минимум 1.5x, но не больше maxScale и calculatedScale
        const scale = Math.max(minScale, Math.min(calculatedScale, maxScale))
        
        w = Math.round(w * scale)
        h = Math.round(h * scale)

        canvas.width = w
        canvas.height = h
        sourceCanvas.width = w
        sourceCanvas.height = h
        canvasSizeRef.current = { width: w, height: h }

        // Draw source image to hidden canvas
        const sCtx = sourceCanvas.getContext("2d", { willReadFrequently: true })
        if (!sCtx) return
        sCtx.drawImage(image, 0, 0, w, h)
        const imageData = sCtx.getImageData(0, 0, w, h)

        // Generate nail positions
        const nails = generateNails(w, h, settings.nailCount, settings.shape)
        nailsRef.current = nails

        // Fill background
        const ctx = canvas.getContext("2d")
        if (!ctx) return
        ctx.fillStyle = settings.backgroundColor
        ctx.fillRect(0, 0, w, h)
        
        // Устанавливаем режим композиции для правильного накопления линий
        ctx.globalCompositeOperation = "source-over"

        // Draw nails
        drawNails(ctx, nails)

        runningRef.current = true
        onRunningChange(true)

        // Use Go backend if available, otherwise use local generator
        if (useGoBackend) {
          // Use Go backend for faster generation
          const goGen = generateStringArtWithGoBackend(imageData, nails, settings)
          asyncGeneratorRef.current = goGen
          
          let lineCount = 0
          // Увеличенный batch size для ускорения - обрабатываем больше линий за кадр
          // Go backend быстрый, поэтому можем обрабатывать больше
          const batchSize = settings.lineCount > 30000 ? 200 : settings.lineCount > 10000 ? 150 : 100

          const animateAsync = async () => {
            if (!runningRef.current || !asyncGeneratorRef.current) return

            const ctx2 = canvas.getContext("2d")
            if (!ctx2) return

            try {
              // Go backend быстрый, обрабатываем больше линий за кадр
              for (let i = 0; i < batchSize; i++) {
                const result = await asyncGeneratorRef.current.next()
                
                if (result.done) {
                  // Final result received - all lines already drawn via streaming
                  const finalResult = result.value
                  if (finalResult && finalResult.nails) {
                    nailsRef.current = finalResult.nails
                  }

                  runningRef.current = false
                  onRunningChange(false)
                  onComplete()
                  onProgress(lineCount, settings.lineCount)
                  return
                }
                
                const segment = result.value
                if (segment && segment.from !== undefined && segment.to !== undefined) {
                  linesRef.current.push(segment)
                  drawLine(ctx2, nails, segment)
                  lineCount++
                }
              }

              onProgress(lineCount, settings.lineCount)
              // Use requestAnimationFrame for smooth 60fps animation
              animFrameRef.current = requestAnimationFrame(animateAsync)
            } catch (error) {
              console.error("Error in Go backend generation:", error)
              // Fallback to local generator
              setUseGoBackend(false)
              const localGen = stringArtGenerator(imageData, nails, settings)
              generatorRef.current = localGen
              let fallbackLineCount = linesRef.current.length

              const fallbackBatchSize = settings.lineCount > 30000 ? 150 : settings.lineCount > 10000 ? 100 : 50

              const fallbackAnimate = () => {
                if (!runningRef.current || !generatorRef.current) return
                const ctx3 = canvas.getContext("2d")
                if (!ctx3) return
                const startT = performance.now()
                for (let i = 0; i < fallbackBatchSize; i++) {
                  if (performance.now() - startT > 16) break
                  const result = generatorRef.current.next()
                  if (result.done) {
                    runningRef.current = false
                    onRunningChange(false)
                    onComplete()
                    onProgress(settings.lineCount, settings.lineCount)
                    return
                  }
                  const seg = result.value
                  linesRef.current.push(seg)
                  drawLine(ctx3, nails, seg)
                  fallbackLineCount++
                }
                onProgress(fallbackLineCount, settings.lineCount)
                animFrameRef.current = requestAnimationFrame(fallbackAnimate)
              }
              animFrameRef.current = requestAnimationFrame(fallbackAnimate)
            }
          }

          animateAsync()
        } else {
          // Use local generator
          const gen = stringArtGenerator(imageData, nails, settings)
          generatorRef.current = gen

          let lineCount = 0
          // Увеличенный batch size для ускорения локальной генерации
          const batchSize = settings.lineCount > 30000 ? 150 : settings.lineCount > 10000 ? 100 : 50

          const animate = () => {
            if (!runningRef.current || !generatorRef.current) return

            const ctx2 = canvas.getContext("2d")
            if (!ctx2) return

            // Use performance timing for smooth animation
            const startTime = performance.now()
            const maxTimePerFrame = 16 // ~60fps

            for (let i = 0; i < batchSize; i++) {
              // Check if we're taking too long - yield to browser for smooth animation
              if (performance.now() - startTime > maxTimePerFrame) {
                break
              }
              
              const result = generatorRef.current.next()
              if (result.done) {
                runningRef.current = false
                onRunningChange(false)
                onComplete()
                onProgress(settings.lineCount, settings.lineCount)
                return
              }
              const segment = result.value
              linesRef.current.push(segment)
              drawLine(ctx2, nails, segment)
              lineCount++
            }

            onProgress(lineCount, settings.lineCount)
            // Use requestAnimationFrame for smooth 60fps animation
            animFrameRef.current = requestAnimationFrame(animate)
          }

          animFrameRef.current = requestAnimationFrame(animate)
        }
      },
      [settings, drawNails, drawLine, onProgress, onComplete, onRunningChange, reset, useGoBackend]
    )

    const downloadPNG = useCallback(() => {
      const canvas = canvasRef.current
      if (!canvas) return
      const link = document.createElement("a")
      link.download = "string-art.png"
      link.href = canvas.toDataURL("image/png")
      link.click()
    }, [])

    const downloadSVG = useCallback(() => {
      const nails = nailsRef.current
      const lines = linesRef.current
      if (nails.length === 0 || lines.length === 0) return

      const { width, height } = canvasSizeRef.current
      
      // Формат как в оригинале: hex цвет + hex альфа-канал
      const opacityValue = Math.round((settings.lineOpacity / 100) * 255)
      const opacityHex = opacityValue.toString(16).padStart(2, '0')
      const lineColor = `${settings.lineColor}${opacityHex}`

      let svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">\n`
      svg += `<rect width="${width}" height="${height}" fill="${settings.backgroundColor}"/>\n`

      for (const line of lines) {
        // Используем path как в оригинале
        svg += `<path d="M ${nails[line.from].x.toFixed(1)} ${nails[line.from].y.toFixed(1)} L ${nails[line.to].x.toFixed(1)} ${nails[line.to].y.toFixed(1)}" stroke="${lineColor}" stroke-width="1" fill="none"/>\n`
      }

      svg += `</svg>`

      const blob = new Blob([svg], { type: "image/svg+xml" })
      const link = document.createElement("a")
      link.download = "string-art.svg"
      link.href = URL.createObjectURL(blob)
      link.click()
      URL.revokeObjectURL(link.href)
    }, [settings.lineColor, settings.lineOpacity, settings.backgroundColor])

    const getLines = useCallback(() => linesRef.current, [])
    const getNails = useCallback(() => nailsRef.current, [])

    useImperativeHandle(ref, () => ({
      start,
      stop,
      reset,
      downloadPNG,
      downloadSVG,
      getLines,
      getNails,
    }))

    useEffect(() => {
      return () => {
        if (animFrameRef.current) {
          cancelAnimationFrame(animFrameRef.current)
        }
      }
    }, [])

    return (
      <div className="relative flex items-center justify-center w-full min-h-[200px] md:min-h-[400px] py-0">
        <canvas
          ref={sourceCanvasRef}
          className="hidden"
        />
        <canvas
          ref={canvasRef}
          className="w-auto h-auto max-w-full max-h-[calc(89vh-39px)] rounded-lg shadow-2xl shadow-primary/5"
          style={{ imageRendering: "auto" }}
        />
        {nailsRef.current.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div className="text-center px-4">
              <div className="w-16 h-16 md:w-24 md:h-24 mx-auto mb-4 md:mb-6 rounded-full border-2 border-dashed border-muted-foreground/30 flex items-center justify-center">
                <svg
                  className="w-7 h-7 md:w-10 md:h-10 text-muted-foreground/40"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5a2.25 2.25 0 002.25-2.25V5.25a2.25 2.25 0 00-2.25-2.25H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z"
                  />
                </svg>
              </div>
              <p className="text-muted-foreground/60 text-xs md:text-sm font-medium max-w-[200px] md:max-w-none mx-auto text-balance">
                Загрузите изображение и нажмите &laquo;Запустить&raquo;
              </p>
            </div>
          </div>
        )}
      </div>
    )
  }
)

StringArtCanvas.displayName = "StringArtCanvas"

export default StringArtCanvas
