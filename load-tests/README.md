# Load Testing для Chat Application

Полный набор инструментов для нагрузочного и стресс-тестирования чат-приложения.

## Быстрый старт

### Установка k6

**Windows (Chocolatey):**
```powershell
choco install k6
```

**Или скачать:** https://dl.k6.io/msi/k6-latest-amd64.msi

### Запуск тестов

```powershell
cd load-tests

# Smoke test (быстрая проверка)
.\run-load-test.ps1 -Scenario smoke -Type api

# Load test (50 VU, 5 минут)
.\run-load-test.ps1 -Scenario load -Type api

# Stress test
.\run-load-test.ps1 -Scenario stress -Type combined
```

## Экстремальный тест (400 клиентов)

Полный сценарий:
- **400 k6 VU** генерируют 10 msg/sec
- **10 браузеров** с медленной сетью (SLOW_3G)
- **20 браузеров** с обрывающейся сетью
- **20% сообщений** с файлами (UUID внутри)

### Запуск полного теста

```powershell
# Полный тест (5 минут)
.\run-extreme-test.ps1

# Укороченный тест (1 минута)
.\run-extreme-test.ps1 -DurationSeconds 60

# Только k6 (без браузеров)
.\run-extreme-test.ps1 -K6Only

# Только браузеры (без k6)
.\run-extreme-test.ps1 -BrowsersOnly

# Кастомные параметры
.\run-extreme-test.ps1 -K6Users 100 -TargetMPS 5 -SlowBrowsers 5 -FlakyBrowsers 10
```

### Параметры extreme теста

| Параметр | Default | Описание |
|----------|---------|----------|
| `-K6Users` | 400 | Количество k6 виртуальных пользователей |
| `-TargetMPS` | 10 | Целевое количество сообщений в секунду |
| `-SlowBrowsers` | 10 | Браузеры с медленной сетью |
| `-FlakyBrowsers` | 20 | Браузеры с обрывающейся сетью |
| `-DurationSeconds` | 300 | Длительность теста (секунды) |
| `-FileRatio` | 0.2 | Доля сообщений с файлами (0.2 = 20%) |
| `-K6Only` | - | Запустить только k6 |
| `-BrowsersOnly` | - | Запустить только браузеры |
| `-DryRun` | - | Показать конфигурацию без запуска |

## Типы тестов

### 1. API Load Test (`api-load-test.js`)
Тестирование REST API:
- Регистрация/логин
- Создание чатов
- Отправка/получение сообщений

### 2. WebSocket Load Test (`websocket-load-test.js`)
Тестирование WebSocket (Centrifugo):
- Множество одновременных подключений
- Real-time получение сообщений

### 3. Extreme Load Test (`extreme-load-test.js`)
Стресс-тест одного чата:
- 400 пользователей в одном чате
- 10 msg/sec с файлами
- Координированная нагрузка

### 4. Browser Clients (`extreme-browser-clients.spec.ts`)
Selenium тесты с плохой сетью:
- 10 клиентов на SLOW_3G
- 20 клиентов с обрывами соединения
- Chrome DevTools Protocol для эмуляции сети

## Сценарии нагрузки

| Сценарий | VUs | Длительность | Описание |
|----------|-----|--------------|----------|
| smoke | 5 | 30s | Проверка работоспособности |
| load | 50 | 5m | Нормальная нагрузка |
| stress | 100-300 | 10m | Стрессовая нагрузка |
| spike | 500 | 3m | Резкий всплеск |
| soak | 50 | 30m | Длительная нагрузка |

## Визуализация (Grafana)

```powershell
# Запуск инфраструктуры
.\run-load-test.ps1 -StartInfra

# Открыть Grafana: http://localhost:3001
# Логин: admin / admin

# Запуск тестов (метрики идут в InfluxDB)
.\run-load-test.ps1 -Scenario load -Type api

# Остановка
.\run-load-test.ps1 -StopInfra
```

## Метрики

### HTTP метрики:
- `http_req_duration` - время ответа
- `http_req_failed` - процент ошибок
- `http_reqs` - количество запросов

### Кастомные метрики:
- `messages_sent` - отправленные сообщения
- `messages_with_files` - сообщения с файлами
- `files_uploaded` - загруженные файлы
- `message_latency` - латентность отправки
- `file_upload_latency` - латентность загрузки файлов

## Thresholds (критерии успеха)

```javascript
thresholds: {
  http_req_duration: ['p(95)<500', 'p(99)<1000'],
  http_req_failed: ['rate<0.01'],
  api_errors: ['rate<0.15'],
}
```

## Результаты примеров

### Smoke test (10 VU, 30s):
```
messages_sent: 64
files_uploaded: 19
http_req_failed: 0.00%
p95 latency: 75ms
```

### Load test (50 VU, 5m):
```
messages_sent: ~1500
http_req_failed: 0.00%
p95 latency: ~100ms
```

## Структура файлов

```
load-tests/
├── config.js                    # Конфигурация
├── api-load-test.js             # REST API тесты
├── websocket-load-test.js       # WebSocket тесты
├── combined-load-test.js        # Комбинированные
├── extreme-load-test.js         # Экстремальный тест
├── run-load-test.ps1            # Стандартный runner
├── run-extreme-test.ps1         # Extreme тест runner
├── docker-compose.yml           # Grafana + InfluxDB
├── grafana/                     # Dashboards
└── README.md                    # Документация
```

## Требования

- Docker Desktop
- k6 (опционально, можно через Docker)
- Node.js 18+ (для browser тестов)
- Chrome/Chromium (для Selenium)
