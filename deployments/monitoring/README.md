# Monitoring Stack

Полный мониторинг для Chat Application во время нагрузочного тестирования.

## Быстрый старт

```powershell
# Запуск мониторинга
.\deployments\monitoring\start-monitoring.ps1

# Остановка
.\deployments\monitoring\start-monitoring.ps1 -Stop
```

## Компоненты

| Компонент | URL | Описание |
|-----------|-----|----------|
| **Grafana** | http://localhost:3000 | Визуализация метрик (admin/admin) |
| **Prometheus** | http://localhost:9090 | Сбор и хранение метрик |
| **RabbitMQ** | http://localhost:15672 | Management UI (chatapp/secret) |
| **Centrifugo** | http://localhost:8000 | Admin UI |

## Grafana Dashboards

### 1. Chat Services Overview
- API Gateway request rate и latency
- RabbitMQ messages published/consumed
- RabbitMQ queue depth
- Centrifugo client connections
- Centrifugo messages sent/received

### 2. Container Resources
- CPU usage по контейнерам
- Memory usage
- Network RX/TX
- PostgreSQL connections
- Redis clients и commands/sec

## Метрики

### API Gateway
- `api_gateway_http_requests_total` - количество запросов
- `api_gateway_http_request_duration_seconds` - latency
- `api_gateway_http_active_requests` - активные запросы

### RabbitMQ (Prometheus plugin)
- `rabbitmq_channel_messages_published_total`
- `rabbitmq_channel_messages_delivered_total`
- `rabbitmq_queue_messages` - глубина очереди

### Centrifugo
- `centrifugo_node_num_clients` - WebSocket клиенты
- `centrifugo_node_num_subscriptions`
- `centrifugo_messages_sent_count`

### PostgreSQL (via exporter)
- `pg_stat_activity_count` - активные соединения
- `pg_stat_database_*` - статистика БД

### Redis (via exporter)
- `redis_connected_clients`
- `redis_commands_processed_total`
- `redis_memory_used_bytes`

## Запуск нагрузочных тестов с мониторингом

```powershell
# 1. Запустить основные сервисы
docker compose up -d

# 2. Запустить мониторинг
.\deployments\monitoring\start-monitoring.ps1

# 3. Открыть Grafana: http://localhost:3000

# 4. Запустить нагрузочный тест
cd load-tests
.\run-extreme-test.ps1 -K6Users 100 -DurationSeconds 60

# 5. Наблюдать метрики в реальном времени в Grafana
```

## Ручной запуск через Docker Compose

```powershell
# Запуск всего стека
docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml up -d

# Только мониторинг (без cAdvisor на Windows)
docker compose -f docker-compose.yml -f deployments/monitoring/docker-compose.monitoring.yml up -d prometheus grafana redis-exporter postgres-exporter
```

## Prometheus Targets

Проверить статус сбора метрик: http://localhost:9090/targets

Targets:
- `prometheus` - self-monitoring
- `rabbitmq` - RabbitMQ Prometheus plugin (port 15692)
- `centrifugo` - встроенные метрики
- `api-gateway` - Go сервис /metrics
- `redis` - Redis exporter
- `postgres` - PostgreSQL exporter
