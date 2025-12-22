# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Микросервисное чат-приложение с JWT авторизацией, gRPC коммуникацией и real-time обновлениями через Centrifugo.

## Build & Run Commands

```bash
# Start all services with Docker Compose
make docker-up

# Start only infrastructure (postgres, rabbitmq, centrifugo)
make docker-infra

# Build all Go services
make build

# Run tests
make test
make test-users      # только users-service
make test-coverage   # с coverage report

# Generate gRPC code from proto
make proto

# Run migrations
make migrate-up

# CLI для управления пользователями
./bin/users-cli user add --username admin --email admin@example.com --password secret --role owner
./bin/users-cli user list
./bin/users-cli user set-role --id <uuid> --role moderator

# Vue SPA development
make web-install
make web-dev
make web-build
```

## Architecture

```
├── pkg/                    # Shared Go packages (jwt, postgres, rabbitmq, logger, validator)
├── proto/chat/             # gRPC definitions for chat-service
├── migrations/             # SQL migrations for con_test schema
├── services/
│   ├── users/              # JWT auth, user management, CLI (HTTP :8081)
│   ├── chat/               # Chat & messages via gRPC (:50051), RabbitMQ events
│   ├── files/              # File storage & sharing (HTTP :8082)
│   └── api-gateway/        # REST API + Centrifugo integration + Vue SPA (:8080)
├── deployments/
│   ├── nginx/              # Reverse proxy config
│   └── centrifugo/         # WebSocket server config
```

## Key Technologies

- **Go**: chi (HTTP), grpc, pgx (PostgreSQL), amqp091-go (RabbitMQ)
- **Frontend**: Vue 3 + TypeScript + Pinia + Vite
- **Infrastructure**: PostgreSQL, RabbitMQ, Centrifugo, Nginx

## Database

- Schema: `con_test`
- Users: users, refresh_tokens, groups, group_members
- Chat: chats, chat_participants, messages, message_reactions, polls
- Files: files, file_links, file_link_permissions, file_share_links

## Role Model

- `owner` - полный доступ
- `moderator` - модерация чатов и пользователей
- `user` - стандартный доступ (чтение/запись)
- `guest` - только чтение (не может отправлять сообщения)

## RabbitMQ Events

Exchange: `chat.events` (topic)
- `chat.created`, `chat.deleted`
- `message.created`, `message.updated`, `message.deleted`

## API Endpoints

Key endpoints:
- `/api/auth/login`, `/api/auth/refresh`, `/api/auth/logout`
- `/api/users`, `/api/users/{userGUID}`
- `/api/chats`, `/api/chats/{chatID}/messages`
- `/api/files/upload`, `/api/files/{id}/download`

## Swagger & API Documentation

### Swagger UI (REST API)
- **URL**: http://localhost:8888/swagger/index.html
- **OpenAPI JSON**: http://localhost:8888/swagger/doc.json

```bash
# Генерация swagger документации
make swagger-install    # установить swag CLI (один раз)
make swagger            # сгенерировать docs из аннотаций
make swagger-fmt        # форматировать аннотации
```

Swagger генерируется автоматически при сборке Docker образа api-gateway.

### gRPC Documentation
```bash
# Интерактивный UI для тестирования gRPC
go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
grpcui -plaintext localhost:50051  # chat-service
grpcui -plaintext localhost:50052  # presence-service
grpcui -plaintext localhost:50053  # files-service

# Генерация HTML документации из proto файлов
make proto-doc-install  # установить protoc-gen-doc (один раз)
make proto-doc          # сгенерировать docs/proto-docs.html
```

## Monitoring & Observability

### Endpoints
| Service | URL | Description |
|---------|-----|-------------|
| Prometheus | http://localhost:9090 | Metrics storage & queries |
| Grafana | http://localhost:3000 | Dashboards (admin/admin) |
| RabbitMQ | http://localhost:15672 | Queue management (guest/guest) |

### Prometheus Metrics
Каждый сервис экспортирует метрики на `/metrics`:
- `api-gateway`: http://localhost:9180/metrics
- `users-service`: http://localhost:8081/metrics
- `files-service`: http://localhost:8082/metrics

### Полезные Prometheus запросы
```promql
# HTTP request rate
rate(http_requests_total[5m])

# Request latency 95th percentile
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# gRPC request rate
rate(grpc_server_handled_total[5m])

# RabbitMQ messages published
rate(rabbitmq_messages_published_total[5m])
```

### Health Checks
```bash
curl http://localhost:8888/health              # api-gateway (через nginx)
curl http://localhost:9180/health              # api-gateway (напрямую)
curl http://localhost:8081/health              # users-service
curl http://localhost:8082/health              # files-service
```

## Testing

Always write tests. Run `make test` before committing. Use `testify` for assertions and mocks.

### Important Rules for Claude

1. **После создания теста - запусти его!** Нельзя считать тест готовым, пока он не был запущен и не прошёл успешно.
2. **После изменения Vue компонентов** - запусти `make web-build` или `npm run build` для проверки TypeScript.
3. **Перед коммитом** - убедись что Docker образ собирается: `docker-compose build --no-cache <service>`
4. **После изменений фронтенда** - ОБЯЗАТЕЛЬНО пересобрать и задеплоить в контейнер:
   ```bash
   docker-compose build --no-cache api-gateway && docker-compose up -d api-gateway
   ```
   Если кэш не сбрасывается, использовать: `docker builder prune -f` перед сборкой.

### Запуск команд на разных платформах

- **Windows**: использовать `powershell.exe -NoProfile -ExecutionPolicy Bypass -File script.ps1` или Docker
- **Linux**: стандартная консоль bash или Docker
- **Важно**: На Windows НЕ использовать cmd.exe через WSL - вывод теряется

### E2E Tests (Selenium)

```powershell
# Запуск конкретного теста
cd services/api-gateway/web
.\run-reply-forward-test.ps1        # reply/forward тесты
.\run-threads-reply-test.ps1        # threads тесты
.\run-file-upload-test.ps1          # file upload тесты
```
