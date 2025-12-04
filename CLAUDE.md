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

See `arch/swagget.txt` for full OpenAPI spec. Key endpoints:
- `/api/auth/login`, `/api/auth/refresh`, `/api/auth/logout`
- `/api/users`, `/api/users/{userGUID}`
- `/api/chats`, `/api/chats/{chatID}/messages`
- `/api/files/upload`, `/api/files/{id}/download`

## Testing

Always write tests. Run `make test` before committing. Use `testify` for assertions and mocks.
