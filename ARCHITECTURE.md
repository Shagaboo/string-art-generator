# Архитектура проекта String Art Generator

## 📋 Оглавление
1. [Обзор системы](#обзор-системы)
2. [Компоненты](#компоненты)
3. [Взаимодействие компонентов](#взаимодействие-компонентов)
4. [Технологический стек](#технологический-стек)
5. [Поток данных](#поток-данных)
6. [API](#api)
7. [Деплой](#деплой)

---

## Обзор системы

Проект состоит из двух основных компонентов:
- **Frontend** (Next.js/React) — пользовательский интерфейс
- **Backend** (Go) — сервер генерации string art

```
┌─────────────┐         HTTP/SSE          ┌─────────────┐
│   Browser   │ ◄──────────────────────► │   Frontend  │
│  (Client)   │                           │  (Next.js)  │
└─────────────┘                           └──────┬──────┘
                                                 │
                                                 │ HTTP API
                                                 │
                                          ┌──────▼──────┐
                                          │   Backend   │
                                          │    (Go)     │
                                          └─────────────┘
```

---

## Компоненты

### Frontend (Next.js/React/TypeScript)

#### Структура
```
frontend/
├── app/                    # Next.js App Router
│   ├── page.tsx            # Главная страница
│   ├── layout.tsx          # Layout приложения
│   └── globals.css         # Глобальные стили
├── components/             # React компоненты
│   ├── string-art-canvas.tsx    # Canvas для отрисовки
│   ├── controls-panel.tsx       # Панель управления
│   └── ui/                      # UI компоненты (shadcn/ui)
├── lib/                    # Библиотеки и утилиты
│   ├── string-art-engine.ts     # Локальный генератор (fallback)
│   ├── go-backend-client.ts     # Клиент для Go backend
│   └── utils.ts                 # Утилиты
└── public/                 # Статические файлы
```

#### Основные компоненты

**StringArtCanvas**
- Управляет canvas для отрисовки string art
- Обрабатывает генерацию (Go backend или локальный fallback)
- Анимирует отрисовку линий через `requestAnimationFrame`
- Управляет состоянием генератора и прогрессом

**ControlsPanel**
- Форма с настройками генерации
- Загрузка изображения
- Параметры: гвозди, линии, яркость, контраст, гамма, форма
- Кнопки экспорта (G-code, JSON, CSV)

**GoBackendClient**
- HTTP клиент для взаимодействия с Go backend
- SSE (Server-Sent Events) для потоковой генерации
- Обработка ошибок и fallback на локальный генератор

### Backend (Go)

#### Структура
```
backend/
├── main.go              # Точка входа, HTTP сервер
├── handler.go           # HTTP handlers (generate, export, health)
├── generator.go         # Основной алгоритм генерации
├── image.go             # Обработка изображений (grayscale, brightness, contrast)
├── line.go              # Алгоритм Брезенхема для линий
├── nail.go              # Генерация гвоздей (круг, прямоугольник, случайное)
├── types.go             # Типы данных (запросы, ответы)
└── go.mod               # Зависимости
```

#### Основные модули

**Generator**
- `NewGenerator()` — создание генератора
- `GenerateNextLine()` — генерация одной линии
- Параллельная обработка кандидатов
- Управление состоянием (currentNail, grayscale)

**ImageProcessor**
- `NewImageProcessor()` — создание процессора с таблицами
- `ProcessToGrayscale()` — конвертация RGBA → grayscale
- Применение brightness, contrast, gamma через таблицы

**Nail Generator**
- `GenerateNails()` — генерация гвоздей по форме
- `generateCircleNails()` — круг
- `generateRectangleNails()` — прямоугольник
- `generateRandomNails()` — случайное размещение
- `generateGridNails()` — полу-сетка
- Проверка минимального расстояния

**Line Algorithms**
- `BresenhamBrightness()` — вычисление яркости вдоль линии
- `BresenhamUpdate()` — обновление яркости пикселей

---

## Взаимодействие компонентов

### Сценарий генерации

```
1. Пользователь загружает изображение и настраивает параметры
   ↓
2. Frontend отправляет POST /api/generate с данными изображения
   ↓
3. Backend обрабатывает изображение:
   - ImageProcessor → grayscale
   - GenerateNails → массив гвоздей
   - NewGenerator → создание генератора
   ↓
4. Backend начинает генерацию в цикле:
   - GenerateNextLine() → одна линия
   - Отправка через SSE: data: {line, index, total}
   ↓
5. Frontend получает SSE события:
   - Обновляет прогресс
   - Рисует линию на canvas через requestAnimationFrame
   ↓
6. После завершения:
   - Backend отправляет финальный ответ с массивом всех линий
   - Frontend завершает анимацию
```

### Сценарий экспорта

```
1. Пользователь нажимает "G-code" / "JSON" / "CSV"
   ↓
2. Frontend отправляет POST /api/export с:
   - nails (координаты гвоздей)
   - lines (последовательность линий)
   - format (gcode/json/csv)
   ↓
3. Backend генерирует файл в выбранном формате
   ↓
4. Frontend скачивает файл
```

---

## Технологический стек

### Frontend
- **Next.js 16** — React framework с App Router
- **React 19** — UI библиотека
- **TypeScript** — типизация
- **Tailwind CSS** — стилизация
- **shadcn/ui** — UI компоненты
- **Canvas API** — отрисовка string art

### Backend
- **Go 1.21+** — язык программирования
- **gorilla/mux** — HTTP router
- **Standard library** — все остальное (нет внешних зависимостей для алгоритма)

### Инфраструктура
- **Nginx** — reverse proxy
- **Systemd** — управление сервисами
- **SSH/SCP** — деплой

---

## Поток данных

### Генерация (SSE)

```
Client                    Backend
  │                          │
  │ POST /api/generate       │
  │─────────────────────────►│
  │                          │ Process image
  │                          │ Generate nails
  │                          │ Create generator
  │                          │
  │ SSE: data: {...}         │
  │◄─────────────────────────│ Line 1
  │                          │
  │ SSE: data: {...}         │
  │◄─────────────────────────│ Line 2
  │                          │
  │ ...                      │ ...
  │                          │
  │ SSE: data: {...}         │
  │◄─────────────────────────│ Line N (final)
  │                          │
```

### Экспорт

```
Client                    Backend
  │                          │
  │ POST /api/export         │
  │─────────────────────────►│
  │                          │ Generate file
  │                          │ (G-code/JSON/CSV)
  │                          │
  │ File download            │
  │◄─────────────────────────│
  │                          │
```

---

## API

### Endpoints

#### `POST /api/generate`
Генерация string art с потоковой отправкой прогресса через SSE.

**Request:**
```json
{
  "imageData": [uint8],      // RGBA массив
  "width": 800,
  "height": 600,
  "nailCount": 700,
  "lineCount": 20000,
  "lineOpacity": 6.0,
  "brightness": 0.0,
  "contrast": 20.0,
  "gamma": 0.7,
  "invertBrightness": false,
  "shape": "rectangle",
  "nails": []                // опционально
}
```

**Response (SSE):**
```
data: {"line": {"from": 0, "to": 5}, "index": 1, "total": 20000}

data: {"line": {"from": 5, "to": 12}, "index": 2, "total": 20000}

...

data: {"lines": [...], "nails": [...]}  // финальный ответ
```

#### `POST /api/export`
Экспорт результата в формат для ЧПУ.

**Request:**
```json
{
  "nails": [{"x": 100.5, "y": 200.3}, ...],
  "lines": [{"from": 0, "to": 5}, ...],
  "format": "gcode",  // или "json", "csv"
  "width": 800,
  "height": 600
}
```

**Response:**
- `Content-Type: text/plain` (G-code)
- `Content-Type: application/json` (JSON)
- `Content-Type: text/csv` (CSV)
- `Content-Disposition: attachment; filename=...`

#### `GET /api/health`
Проверка работоспособности сервера.

**Response:**
```json
{"status": "ok"}
```

---

## Деплой

### Структура на сервере

```
/var/www/string-art-generator/
├── backend/
│   ├── string-art-backend    # Скомпилированный бинарник
│   └── go.mod
├── frontend/
│   ├── .next/                # Production build
│   ├── public/
│   ├── package.json
│   └── node_modules/
└── deploy/
    └── start-frontend.sh
```

### Systemd сервисы

**string-art-backend.service**
```ini
[Unit]
Description=String Art Generator Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/var/www/string-art-generator/backend
ExecStart=/var/www/string-art-generator/backend/string-art-backend
Restart=always
```

**string-art-frontend.service**
```ini
[Unit]
Description=String Art Generator Frontend
After=network.target

[Service]
Type=simple
WorkingDirectory=/var/www/string-art-generator/frontend
ExecStart=/bin/bash /var/www/string-art-generator/deploy/start-frontend.sh
Restart=always
Environment=NODE_ENV=production
Environment=PORT=3000
```

### Nginx конфигурация

```nginx
server {
    listen 80;
    server_name 185.119.59.241;

    location / {
        proxy_pass http://localhost:3000;  # Frontend
    }

    location /api {
        proxy_pass http://localhost:9000;  # Backend
        proxy_read_timeout 300s;
    }
}
```

---

## Масштабируемость

### Текущая архитектура
- **Один сервер** с frontend и backend
- **Stateless backend** — можно масштабировать горизонтально
- **SSE** — требует постоянного соединения

### Возможные улучшения
- **Redis** — кэширование результатов
- **Queue (RabbitMQ/Kafka)** — асинхронная обработка больших задач
- **Load Balancer** — распределение нагрузки между несколькими backend
- **CDN** — для статических файлов frontend

---

## Безопасность

### Текущие меры
- CORS настроен для разрешения запросов
- Валидация входных данных
- Ограничение размера изображения (через параметры)

### Рекомендации для production
- HTTPS (SSL/TLS)
- Rate limiting
- Аутентификация/авторизация
- Валидация и санитизация всех входных данных
- Ограничение размера файлов

---

## Мониторинг

### Логирование
- Backend: стандартный `log` пакет Go
- Логи доступны через `journalctl -u string-art-backend -f`

### Метрики (потенциально)
- Время генерации
- Количество обработанных запросов
- Использование памяти
- Количество активных SSE соединений

