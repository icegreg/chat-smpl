# Admin Service

REST API сервис для управления и мониторинга chat-smpl системы.

## Описание

Admin Service предоставляет REST API для:
- Мониторинга конференций и участников
- Проверки статуса микросервисов
- Сбора метрик для Prometheus/Grafana
- Взаимодействия с CLI инструментом `rtuccli`

## API Endpoints

### Conferences

#### GET /api/conferences
Список всех конференций с опциональной фильтрацией по статусу.

**Query параметры:**
- `status` (optional) - фильтр по статусу: `active`, `scheduled`, `ended`

**Пример:**
```bash
curl http://localhost:8086/api/conferences
curl http://localhost:8086/api/conferences?status=active
```

**Ответ:**
```json
{
  "conferences": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "Team Meeting",
      "event_type": "adhoc",
      "status": "active",
      "participants_count": 5,
      "started_at": "2024-01-20T10:00:00Z",
      "duration_seconds": 1800
    }
  ],
  "total": 1
}
```

#### GET /api/conferences/{id}
Получить детали конкретной конференции.

**Пример:**
```bash
curl http://localhost:8086/api/conferences/123e4567-e89b-12d3-a456-426614174000
```

#### GET /api/conferences/{id}/participants
Список участников конференции.

**Пример:**
```bash
curl http://localhost:8086/api/conferences/123e4567-e89b-12d3-a456-426614174000/participants
```

**Ответ:**
```json
{
  "participants": [
    {
      "id": "...",
      "conference_id": "...",
      "user_id": "...",
      "username": "user1",
      "extension": "1001",
      "status": "connected",
      "joined_at": "2024-01-20T10:00:00Z",
      "duration_seconds": 1200
    }
  ],
  "total": 5
}
```

#### POST /api/conferences/{id}/end
Завершить конференцию.

**Body:**
```json
{
  "ended_by": "user-guid"
}
```

### Services

#### GET /api/services
Список всех отслеживаемых сервисов и их статус.

**Пример:**
```bash
curl http://localhost:8086/api/services
```

**Ответ:**
```json
{
  "services": [
    {
      "id": "voice-service",
      "name": "Voice Service",
      "type": "voice",
      "status": "running",
      "health": "healthy",
      "last_check": "2024-01-20T10:30:00Z"
    },
    {
      "id": "api-gateway",
      "name": "API Gateway",
      "type": "gateway",
      "status": "running",
      "health": "healthy",
      "last_check": "2024-01-20T10:30:00Z"
    }
  ],
  "total": 5
}
```

#### GET /api/services/{id}
Получить детали конкретного сервиса.

**Пример:**
```bash
curl http://localhost:8086/api/services/voice-service
```

### Health & Metrics

#### GET /health
Health check endpoint.

```bash
curl http://localhost:8086/health
```

#### GET /metrics
Prometheus метрики.

```bash
curl http://localhost:8086/metrics
```

## Запуск

### Docker Compose

```bash
# Запуск всех сервисов включая admin-service
docker-compose up -d

# Только admin-service
docker-compose up -d admin-service
```

### Локальная разработка

```bash
cd services/admin

# Установка зависимостей
go mod download

# Запуск
DATABASE_URL=postgres://chatapp:secret@localhost:5435/chatapp \
VOICE_SERVICE_ADDR=localhost:50054 \
HTTP_PORT=8086 \
go run cmd/server/main.go
```

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://chatuser:chatpass@postgres:5432/chatdb?sslmode=disable` |
| `VOICE_SERVICE_ADDR` | Voice service gRPC address | `voice-service:50054` |
| `HTTP_PORT` | HTTP port для REST API | `8086` |

## Интеграция с Prometheus

Admin service автоматически экспортирует метрики на `/metrics`.

Добавьте в `prometheus.yml`:
```yaml
- job_name: 'admin-service'
  static_configs:
    - targets: ['admin-service:8086']
  metrics_path: /metrics
```

## Grafana Dashboard

Готовый дашборд доступен в:
`deployments/monitoring/grafana/dashboards/admin-service.json`

Дашборд включает:
- Количество активных конференций
- Распределение конференций по типам
- Статус микросервисов
- Количество участников в конференциях
- Request rate для Admin API

## CLI Клиент

См. [rtuccli README](../../cmd/rtuccli/README.md) для информации по использованию CLI.
