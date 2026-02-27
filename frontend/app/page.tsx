"use client"

import { useState, useRef, useCallback } from "react"
import ControlsPanel from "@/components/controls-panel"
import StringArtCanvas, {
  type StringArtCanvasHandle,
} from "@/components/string-art-canvas"
import { DEFAULT_SETTINGS, type StringArtSettings } from "@/lib/string-art-engine"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet"
import { Settings2 } from "lucide-react"

export default function Home() {
  const [settings, setSettings] = useState<StringArtSettings>(DEFAULT_SETTINGS)
  const [isRunning, setIsRunning] = useState(false)
  const [hasImage, setHasImage] = useState(false)
  const [progress, setProgress] = useState({ current: 0, total: 0 })
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const [mobileOpen, setMobileOpen] = useState(false)
  const canvasHandleRef = useRef<StringArtCanvasHandle>(null)
  const loadedImageRef = useRef<HTMLImageElement | null>(null)

  const handleImageLoad = useCallback((img: HTMLImageElement) => {
    loadedImageRef.current = img
    setHasImage(true)

    const previewCanvas = document.createElement("canvas")
    const ctx = previewCanvas.getContext("2d")
    if (ctx) {
      previewCanvas.width = 200
      previewCanvas.height = 200
      const scale = Math.min(200 / img.naturalWidth, 200 / img.naturalHeight)
      const w = img.naturalWidth * scale
      const h = img.naturalHeight * scale
      ctx.drawImage(img, (200 - w) / 2, (200 - h) / 2, w, h)
      setImagePreview(previewCanvas.toDataURL())
    }
  }, [])

  const handleStart = useCallback(() => {
    if (loadedImageRef.current && canvasHandleRef.current) {
      setProgress({ current: 0, total: settings.lineCount })
      canvasHandleRef.current.start(loadedImageRef.current)
      setMobileOpen(false)
    }
  }, [settings.lineCount])

  const handleStop = useCallback(() => {
    canvasHandleRef.current?.stop()
  }, [])

  const handleReset = useCallback(() => {
    canvasHandleRef.current?.reset()
    setProgress({ current: 0, total: 0 })
  }, [])

  const handleProgress = useCallback((current: number, total: number) => {
    setProgress({ current, total })
  }, [])

  const handleComplete = useCallback(() => {
    // Generation complete
  }, [])

  const handleRunningChange = useCallback((running: boolean) => {
    setIsRunning(running)
  }, [])

  const handleDownloadPNG = useCallback(() => {
    canvasHandleRef.current?.downloadPNG()
  }, [])

  const handleDownloadSVG = useCallback(() => {
    canvasHandleRef.current?.downloadSVG()
  }, [])

  const controlsPanelContent = (
    <ControlsPanel
      settings={settings}
      onSettingsChange={setSettings}
      onImageLoad={handleImageLoad}
      onStart={handleStart}
      onStop={handleStop}
      onReset={handleReset}
      onDownloadPNG={handleDownloadPNG}
      onDownloadSVG={handleDownloadSVG}
      isRunning={isRunning}
      hasImage={hasImage}
      progress={progress}
      imagePreview={imagePreview}
    />
  )

  return (
    <main className="flex flex-col h-dvh overflow-hidden bg-background">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-3 md:px-6 md:py-4 border-b border-border/50 shrink-0">
        <div className="flex items-center gap-3">
          {/* Mobile menu button */}
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden shrink-0"
            onClick={() => setMobileOpen(true)}
          >
            <Settings2 className="w-5 h-5" />
            <span className="sr-only">Open settings</span>
          </Button>

          <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
            <svg
              className="w-4 h-4 text-primary"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
            >
              <circle cx="12" cy="12" r="10" />
              <path d="M2 12 L22 6 M4 18 L20 8 M6 4 L18 20 M12 2 L12 22" />
            </svg>
          </div>
          <div className="min-w-0">
            <h1 className="text-sm md:text-base font-semibold text-foreground tracking-tight truncate">
              String Art Generator
            </h1>
            <p className="text-xs text-muted-foreground hidden sm:block">
              From photo to thread art
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 md:gap-4 shrink-0">
          {isRunning && (
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
              <span className="text-xs font-mono text-muted-foreground">
                {progress.current} / {progress.total}
              </span>
            </div>
          )}
          {!isRunning && progress.current > 0 && (
            <span className="text-xs font-mono text-primary/80">
              {progress.current} lines
            </span>
          )}
        </div>
      </header>

      {/* Main Content */}
      <div className="flex flex-col lg:flex-row flex-1 overflow-hidden">
        {/* Desktop Sidebar */}
        <aside className="hidden lg:flex w-80 min-w-80 border-r border-border/50 overflow-hidden flex-col bg-card/50">
          {controlsPanelContent}
        </aside>

        {/* Mobile Sheet */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetContent side="left" className="w-[320px] sm:w-[360px] p-0 overflow-hidden">
            <SheetHeader className="px-4 pt-4 pb-0">
              <SheetTitle className="text-foreground">Settings</SheetTitle>
              <SheetDescription className="text-muted-foreground">
                Configure your string art parameters
              </SheetDescription>
            </SheetHeader>
            <div className="flex-1 overflow-hidden">
              {controlsPanelContent}
            </div>
          </SheetContent>
        </Sheet>

        {/* Canvas Area */}
        <section className="flex-1 flex items-center justify-center p-1 sm:p-2 md:p-4 bg-background overflow-auto relative min-h-0 max-h-screen">
          {/* Subtle grid background */}
          <div
            className="absolute inset-0 opacity-[0.03]"
            style={{
              backgroundImage:
                "radial-gradient(circle at 1px 1px, currentColor 1px, transparent 0)",
              backgroundSize: "32px 32px",
            }}
          />

          <div className="relative z-10 w-full min-h-full flex items-center justify-center py-2">
            <StringArtCanvas
              ref={canvasHandleRef}
              settings={settings}
              onProgress={handleProgress}
              onComplete={handleComplete}
              onRunningChange={handleRunningChange}
            />
          </div>
        </section>
      </div>
    </main>
  )
}
