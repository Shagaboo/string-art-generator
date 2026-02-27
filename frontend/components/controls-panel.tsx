"use client"

import { useCallback, useRef } from "react"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import type { StringArtSettings } from "@/lib/string-art-engine"
import {
  Upload,
  Play,
  Square,
  RotateCcw,
  Download,
  Image as ImageIcon,
  FileCode,
} from "lucide-react"

interface ControlsPanelProps {
  settings: StringArtSettings
  onSettingsChange: (settings: StringArtSettings) => void
  onImageLoad: (image: HTMLImageElement) => void
  onStart: () => void
  onStop: () => void
  onReset: () => void
  onDownloadPNG: () => void
  onDownloadSVG: () => void
  isRunning: boolean
  hasImage: boolean
  progress: { current: number; total: number }
  imagePreview: string | null
}

export default function ControlsPanel({
  settings,
  onSettingsChange,
  onImageLoad,
  onStart,
  onStop,
  onReset,
  onDownloadPNG,
  onDownloadSVG,
  isRunning,
  hasImage,
  progress,
  imagePreview,
}: ControlsPanelProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return

      const img = new Image()
      img.crossOrigin = "anonymous"
      img.onload = () => {
        onImageLoad(img)
      }
      img.src = URL.createObjectURL(file)
    },
    [onImageLoad]
  )

  const updateSetting = <K extends keyof StringArtSettings>(
    key: K,
    value: StringArtSettings[K]
  ) => {
    onSettingsChange({ ...settings, [key]: value })
  }

  const progressPercent =
    progress.total > 0
      ? Math.round((progress.current / progress.total) * 100)
      : 0

  return (
    <div className="flex flex-col gap-4 md:gap-6 p-4 md:p-6 h-full overflow-y-auto">
      {/* Header -- hidden on mobile since sheet already has a title */}
      <div className="hidden lg:block">
        <h2 className="text-lg font-semibold text-foreground tracking-tight">
          Настройки
        </h2>
        <p className="text-xs text-muted-foreground mt-1">
          Загрузите изображение и настройте параметры
        </p>
      </div>

      <Separator className="bg-border/50 hidden lg:block" />

      {/* Image Upload */}
      <div className="flex flex-col gap-3">
        <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Изображение
        </Label>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleFileChange}
          className="hidden"
        />

        {imagePreview ? (
          <div className="relative group">
            <img
              src={imagePreview}
              alt="Preview"
              className="w-full h-24 md:h-32 object-cover rounded-lg border border-border/50"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              className="absolute inset-0 flex items-center justify-center bg-background/80 opacity-0 group-hover:opacity-100 transition-opacity rounded-lg cursor-pointer"
            >
              <Upload className="w-5 h-5 text-muted-foreground" />
            </button>
          </div>
        ) : (
          <button
            onClick={() => fileInputRef.current?.click()}
            className="flex flex-col items-center justify-center gap-2 h-24 md:h-32 rounded-lg border-2 border-dashed border-border/50 hover:border-primary/50 transition-colors cursor-pointer bg-secondary/30"
          >
            <Upload className="w-6 h-6 text-muted-foreground" />
            <span className="text-xs text-muted-foreground">
              Выбрать изображение
            </span>
          </button>
        )}
      </div>

      <Separator className="bg-border/50" />

      {/* Shape */}
      <div className="flex flex-col gap-2">
        <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Форма
        </Label>
        <Select
          value={settings.shape}
          onValueChange={(val) =>
            updateSetting("shape", val as StringArtSettings["shape"])
          }
          disabled={isRunning}
        >
          <SelectTrigger className="w-full bg-secondary/50 border-border/50">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="circle">Круг</SelectItem>
            <SelectItem value="rectangle">Прямоугольник</SelectItem>
            <SelectItem value="random">Случайные</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Invert */}
      <div className="flex items-center gap-3">
        <Checkbox
          id="invert"
          checked={settings.invertBrightness}
          onCheckedChange={(checked) =>
            updateSetting("invertBrightness", checked as boolean)
          }
          disabled={isRunning}
        />
        <Label
          htmlFor="invert"
          className="text-sm text-foreground/80 cursor-pointer"
        >
          Инвертировать яркость
        </Label>
      </div>

      <Separator className="bg-border/50" />

      {/* Image Adjustments */}
      <div className="flex flex-col gap-4">
        <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Настройка изображения
        </Label>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground/80">Контрастность</span>
            <span className="text-xs font-mono text-primary">
              {settings.contrast}
            </span>
          </div>
          <Slider
            value={[settings.contrast]}
            onValueChange={([val]) => updateSetting("contrast", val)}
            min={-100}
            max={100}
            step={1}
            disabled={isRunning}
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground/80">Яркость</span>
            <span className="text-xs font-mono text-primary">
              {settings.brightness}
            </span>
          </div>
          <Slider
            value={[settings.brightness]}
            onValueChange={([val]) => updateSetting("brightness", val)}
            min={-100}
            max={100}
            step={1}
            disabled={isRunning}
          />
        </div>
      </div>

      <Separator className="bg-border/50" />

      {/* Art Settings */}
      <div className="flex flex-col gap-4">
        <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Параметры арта
        </Label>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground/80">
              Количество гвоздей
            </span>
            <span className="text-xs font-mono text-primary">
              {settings.nailCount}
            </span>
          </div>
          <Slider
            value={[settings.nailCount]}
            onValueChange={([val]) => updateSetting("nailCount", val)}
            min={100}
            max={3000}
            step={50}
            disabled={isRunning}
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground/80">
              Количество линий
            </span>
            <span className="text-xs font-mono text-primary">
              {settings.lineCount}
            </span>
          </div>
          <Slider
            value={[settings.lineCount]}
            onValueChange={([val]) => updateSetting("lineCount", val)}
            min={1000}
            max={50000}
            step={500}
            disabled={isRunning}
          />
        </div>

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground/80">
              Непрозрачность линий
            </span>
            <span className="text-xs font-mono text-primary">
              {settings.lineOpacity}%
            </span>
          </div>
          <Slider
            value={[settings.lineOpacity]}
            onValueChange={([val]) => updateSetting("lineOpacity", val)}
            min={1}
            max={100}
            step={1}
            disabled={isRunning}
          />
        </div>
      </div>

      <Separator className="bg-border/50" />

      {/* Colors */}
      <div className="flex flex-col gap-3">
        <Label className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Цвета
        </Label>

        <div className="flex items-center gap-3">
          <div className="relative">
            <input
              type="color"
              value={settings.lineColor}
              onChange={(e) => updateSetting("lineColor", e.target.value)}
              className="w-10 h-10 rounded-lg border border-border/50 cursor-pointer bg-transparent p-0.5"
              disabled={isRunning}
            />
          </div>
          <div>
            <span className="text-sm text-foreground/80">Цвет линии</span>
            <span className="block text-xs font-mono text-muted-foreground">
              {settings.lineColor}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <div className="relative">
            <input
              type="color"
              value={settings.backgroundColor}
              onChange={(e) =>
                updateSetting("backgroundColor", e.target.value)
              }
              className="w-10 h-10 rounded-lg border border-border/50 cursor-pointer bg-transparent p-0.5"
              disabled={isRunning}
            />
          </div>
          <div>
            <span className="text-sm text-foreground/80">Цвет фона</span>
            <span className="block text-xs font-mono text-muted-foreground">
              {settings.backgroundColor}
            </span>
          </div>
        </div>
      </div>

      <Separator className="bg-border/50" />

      {/* Progress */}
      {(isRunning || progress.current > 0) && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Прогресс</span>
            <span className="text-xs font-mono text-primary">
              {progress.current} / {progress.total}
            </span>
          </div>
          <div className="w-full h-1.5 bg-secondary rounded-full overflow-hidden">
            <div
              className="h-full bg-primary transition-all duration-150 rounded-full"
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex flex-col gap-2 mt-auto pb-2 md:pb-0">
        <div className="flex gap-2">
          {!isRunning ? (
            <Button
              onClick={onStart}
              disabled={!hasImage}
              className="flex-1 bg-primary text-primary-foreground hover:bg-primary/90"
            >
              <Play className="w-4 h-4 mr-2" />
              Запустить
            </Button>
          ) : (
            <Button
              onClick={onStop}
              variant="secondary"
              className="flex-1"
            >
              <Square className="w-4 h-4 mr-2" />
              Остановить
            </Button>
          )}
          <Button
            onClick={onReset}
            variant="outline"
            size="icon"
            className="border-border/50"
          >
            <RotateCcw className="w-4 h-4" />
          </Button>
        </div>

        <div className="flex gap-2">
          <Button
            onClick={onDownloadPNG}
            variant="outline"
            className="flex-1 border-border/50 text-foreground/70"
            disabled={progress.current === 0}
          >
            <ImageIcon className="w-4 h-4 mr-2" />
            PNG
          </Button>
          <Button
            onClick={onDownloadSVG}
            variant="outline"
            className="flex-1 border-border/50 text-foreground/70"
            disabled={progress.current === 0}
          >
            <FileCode className="w-4 h-4 mr-2" />
            SVG
          </Button>
        </div>
      </div>
    </div>
  )
}
