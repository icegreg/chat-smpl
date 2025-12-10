# Performance Notes

Документация по оптимизациям производительности и результатам нагрузочного тестирования.

## Выполненные оптимизации

### 1. Миграция с bcrypt на Argon2id

**Проблема:** bcrypt создавал bottleneck при массовой регистрации/авторизации пользователей.

**Решение:** Замена bcrypt на Argon2id с параметрами OWASP:
- Memory: 64 MB
- Iterations: 1
- Parallelism: 4 threads
- Key length: 32 bytes
- Salt length: 16 bytes

**Файлы:**
- `pkg/password/password.go` - новая реализация с поддержкой legacy bcrypt

**Результат:** Значительное ускорение операций авторизации. Поддержка обратной совместимости с существующими bcrypt хешами.

---

### 2. Горизонтальное масштабирование Users Service

**Проблема:** Один инстанс users-service не справлялся с нагрузкой при массовой авторизации.

**Решение:** Масштабирование до 4 реплик в Docker Compose.

**Файлы:**
- `docker-compose.yml` - `deploy: replicas: 4` для users-service
- `deployments/nginx/nginx.conf` - upstream с keepalive для load balancing

**Конфигурация nginx:**
```nginx
upstream users_service {
    server users-service:8081;
    keepalive 64;
    keepalive_requests 1000;
    keepalive_timeout 60s;
}
```

---

### 3. Горизонтальное масштабирование Centrifugo

**Проблема:** Один инстанс Centrifugo ограничивал количество WebSocket соединений.

**Решение:**
1. Настройка Redis как backend для синхронизации состояния между репликами
2. Масштабирование до 3 реплик

**Файлы:**
- `deployments/centrifugo/config.json` - Redis engine configuration
- `docker-compose.yml` - `deploy: replicas: 3` для centrifugo

**Конфигурация Centrifugo:**
```json
{
  "engine": "redis",
  "redis_address": "redis:6379",
  "redis_prefix": "centrifugo",

  "client_concurrency": 16,
  "client_channel_limit": 128,
  "client_queue_max_size": 10485760,
  "client_presence_update_interval": "27s",

  "websocket_compression": false,
  "websocket_write_timeout": "5s",
  "websocket_message_size_limit": 65536
}
```

**Важно:** В Centrifugo v5 параметры `broker` и `presence_manager` встроены в `engine`, отдельно указывать не нужно.

---

### 4. Оптимизация Nginx

**Изменения в `deployments/nginx/nginx.conf`:**

```nginx
worker_processes auto;
worker_rlimit_nofile 65535;

events {
    worker_connections 10240;
    use epoll;
    multi_accept on;
}

http {
    # Performance tuning
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;

    # Keepalive
    keepalive_timeout 65;
    keepalive_requests 1000;

    # Buffers
    proxy_buffer_size 16k;
    proxy_buffers 8 32k;
    proxy_busy_buffers_size 64k;

    # Docker DNS resolver
    resolver 127.0.0.11 valid=10s ipv6=off;
}
```

**WebSocket upstream:**
```nginx
upstream centrifugo {
    server centrifugo:8000;
    keepalive 256;
    keepalive_requests 10000;
    keepalive_timeout 3600s;
}
```

---

## Результаты нагрузочного тестирования

### Тест: 700 VUs, 70 msg/sec, 5 минут

| Метрика | Значение |
|---------|----------|
| VUs | 700 |
| Сообщений отправлено | 21,027 |
| Скорость | ~70 msg/sec |
| HTTP ошибки | 0.17% |
| WS ошибки | 1.7% (12 из 688) |
| WS сообщений получено | 14,726,890 |
| Средний HTTP latency | 129ms |

**Статус:** PASSED

### Тест: 1000 VUs, 50 msg/sec (с ограничениями Windows)

| Метрика | Значение |
|---------|----------|
| VUs | 1000 |
| Setup | 1000/1000 пользователей |
| WS ошибки | Начались на ~577-650 VUs |

**Причина ошибок:** Windows ephemeral port limit (16384 ports: 49152-65535)

---

## Известные ограничения

### Windows Docker Desktop

1. **Ephemeral ports:** Windows имеет лимит ~16384 динамических портов, что ограничивает количество одновременных соединений
2. **WSL2 NAT:** Дополнительный overhead при NAT между Windows и WSL2
3. **Рекомендация:** Для тестирования >700 VUs использовать Linux или увеличить диапазон портов:
   ```powershell
   netsh int ipv4 set dynamicport tcp start=1025 num=64510
   ```

---

## Предложения по дальнейшей оптимизации

### 1. Connection Pooling для PostgreSQL

Добавить PgBouncer перед PostgreSQL для эффективного управления соединениями:

```yaml
pgbouncer:
  image: edoburu/pgbouncer
  environment:
    DATABASE_URL: postgres://chatapp:secret@postgres:5432/chatapp
    POOL_MODE: transaction
    MAX_CLIENT_CONN: 1000
    DEFAULT_POOL_SIZE: 50
```

### 2. Read Replicas для PostgreSQL

Для read-heavy нагрузки добавить реплики для чтения:

```yaml
postgres-replica:
  image: postgres:16-alpine
  command: postgres -c hot_standby=on
  depends_on:
    - postgres
```

### 3. Rate Limiting

Добавить rate limiting на уровне nginx для защиты от DDoS:

```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;
limit_conn_zone $binary_remote_addr zone=conn:10m;

location /api/ {
    limit_req zone=api burst=200 nodelay;
    limit_conn conn 50;
}
```

### 4. Кеширование в Redis

Кешировать часто запрашиваемые данные:
- Информация о пользователях
- Метаданные чатов
- Списки участников

### 5. Message Batching

Группировать сообщения для отправки через Centrifugo:
- Уменьшение количества publish операций
- Снижение нагрузки на Redis

### 6. Async Message Processing

Использовать RabbitMQ для асинхронной обработки:
- Отправка уведомлений
- Обновление счетчиков
- Индексация для поиска

### 7. Мониторинг

Добавить Prometheus + Grafana для мониторинга:
- Latency percentiles (p50, p95, p99)
- Error rates
- Connection counts
- Queue depths

---

## Архитектура масштабирования

```
                    ┌─────────────┐
                    │   Nginx     │
                    │ (LB + Proxy)│
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ users-service │  │ api-gateway   │  │  centrifugo   │
│  (4 replicas) │  │ (1 instance)  │  │  (3 replicas) │
└───────┬───────┘  └───────┬───────┘  └───────┬───────┘
        │                  │                  │
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│  PostgreSQL   │  │   RabbitMQ    │  │     Redis     │
└───────────────┘  └───────────────┘  └───────────────┘
```

---

## Команды для тестирования

```bash
# Базовый тест
cd load-tests
docker compose run --rm \
  -e BASE_URL=http://host.docker.internal:8888 \
  -e WS_URL=ws://host.docker.internal:8888 \
  -e VUS=700 \
  -e TARGET_MPS=70 \
  -e DURATION=5m \
  k6 run extreme-load-test.js

# Проверка состояния сервисов
docker compose ps
docker stats

# Логи Centrifugo
docker compose logs centrifugo --tail 100

# Проверка Redis
docker exec chatapp-redis redis-cli info clients
```
