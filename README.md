# Chat-SMPL

Микросервисное чат-приложение с JWT авторизацией, gRPC коммуникацией между сервисами и real-time обновлениями через Centrifugo.

## Особенности

- **Микросервисная архитектура** — независимые сервисы с чёткими границами ответственности
- **Real-time сообщения** — мгновенная доставка через WebSocket (Centrifugo)
- **Файловые вложения** — загрузка файлов с контролем доступа на уровне участников чата
- **Пересылка сообщений** — с изоляцией прав доступа к файлам между чатами
- **Ролевая модель** — owner, moderator, user, guest
- **Vue 3 SPA** — современный интерфейс на TypeScript + Pinia + TailwindCSS

## Архитектура

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Browser   │────▶│    Nginx    │────▶│ API Gateway │
└─────────────┘     └─────────────┘     └──────┬──────┘
       ▲                                       │
       │ WebSocket                    ┌────────┼────────┐
       │                              │        │        │
┌──────┴──────┐                 ┌─────▼─┐ ┌────▼───┐ ┌──▼───┐
│  Centrifugo │◀────────────────│ Users │ │  Chat  │ │Files │
└─────────────┘                 │Service│ │Service │ │Service│
       ▲                        └───────┘ └────┬───┘ └──────┘
       │                                       │ gRPC
┌──────┴──────┐     ┌─────────────┐            │
│  WebSocket  │◀────│  RabbitMQ   │◀───────────┘
│   Service   │     └─────────────┘
└─────────────┘
```

### Сервисы

| Сервис | Порт | Описание |
|--------|------|----------|
| **nginx** | 8888 | Reverse proxy, точка входа |
| **api-gateway** | 9180 | REST API + маршрутизация |
| **users-service** | 8181 | Аутентификация, JWT, управление пользователями |
| **chat-service** | 50051 | gRPC, чаты и сообщения |
| **files-service** | 8082, 50053 | Файловое хранилище (HTTP + gRPC) |
| **websocket-service** | — | Доставка событий в Centrifugo |
| **presence-service** | 50052 | gRPC, статусы пользователей online/offline |
| **centrifugo** | 8000 | WebSocket сервер |

### Инфраструктура

- **PostgreSQL** — основная БД (порт 5435)
- **RabbitMQ** — очередь событий (порт 5672, UI: 15672)
- **Redis** — кэш для presence (порт 6379)

## Быстрый старт

### Требования

- Docker и Docker Compose
- Go 1.23+ (для локальной разработки)
- Node.js 20+ (для Vue SPA и e2e тестов)
- Make

### Запуск всего стека

```bash
# Клонировать репозиторий
git clone https://github.com/icegreg/chat-smpl.git
cd chat-smpl

# Запустить все сервисы
make docker-up

# Или через docker-compose напрямую
docker-compose up -d
```

Приложение будет доступно по адресу: **http://localhost:8888**

### Проверка статуса

```bash
# Статус контейнеров
docker-compose ps

# Логи всех сервисов
make docker-logs

# Логи конкретного сервиса
docker logs chatapp-gateway -f
```

### Остановка

```bash
make docker-down

# Полная очистка (включая volumes)
make docker-clean
```

## API Документация

После запуска приложения доступны следующие страницы документации:

| Страница | URL | Описание |
|----------|-----|----------|
| **Swagger UI** | http://localhost:8888/swagger/index.html | REST API документация (OpenAPI) |
| **WebSocket Events** | http://localhost:8888/api/docs/events.html | Документация WebSocket событий Centrifugo |
| **Events JSON Schema** | http://localhost:8888/api/docs/events | JSON схема всех WebSocket событий |

### WebSocket Events

Все события доставляются через Centrifugo в персональный канал пользователя `user:{userId}`.

Типы событий:
- `chat.*` — создание, обновление, удаление чатов
- `message.*` — сообщения (created, updated, deleted, restored)
- `reaction.*` — реакции на сообщения
- `thread.*` — треды
- `typing` — индикатор набора текста
- `conference.*`, `participant.*`, `call.*` — голосовые вызовы

## Разработка

### Локальный запуск сервисов

```bash
# Запустить только инфраструктуру
make docker-infra

# Собрать все сервисы
make build

# Запустить конкретный сервис
make run-users
make run-chat
make run-files
make run-gateway
```

### Генерация сертификатов
```bash
# Сгенерировать SSL сертификаты (обязательно для nginx и FreeSWITCH)
docker-compose --profile ssl-gen run --rm ssl-gen
```
### Vue SPA разработка

```bash
# Установить зависимости
make web-install

# Dev сервер с hot-reload
make web-dev

# Production сборка
make web-build
```

### Генерация Proto

```bash
# Установить protoc плагины
make proto-install

# Сгенерировать Go код из .proto файлов
make proto
```

### Миграции БД

```bash
# Применить все миграции
make migrate-up

# Создать новую миграцию
make migrate-create name=add_some_field service=chat
```

## Тестирование

### Unit тесты

```bash
# Все тесты
make test

# Тесты конкретного сервиса
make test-users
make test-chat
make test-files
make test-gateway
make test-pkg

# С покрытием
make test-coverage
```

### E2E тесты (Selenium)

E2E тесты находятся в `services/api-gateway/web/e2e-selenium/` и используют Selenium WebDriver + Mocha + Chai.

#### Требования для e2e

- Chrome/Chromium браузер
- ChromeDriver (должен соответствовать версии Chrome)

#### Запуск e2e тестов

```bash
cd services/api-gateway/web

# Установить зависимости
npm install

# Запуск всех e2e тестов (headless)
npm run test:e2e

# Запуск с открытым браузером (для отладки)
HEADLESS=false BASE_URL=http://127.0.0.1:8888 npm run test:e2e

# Запуск конкретного теста
npm run test:e2e -- --grep "Login"
npm run test:e2e -- --grep "File Upload"
npm run test:e2e -- --grep "Message Forward"
```

#### Доступные e2e тест-сьюты

| Тест | Описание |
|------|----------|
| `register.spec.ts` | Регистрация пользователя |
| `login.spec.ts` | Вход в систему |
| `auth-flow.spec.ts` | Полный флоу авторизации |
| `chat.spec.ts` | Создание чатов, базовые операции |
| `multiuser-chat.spec.ts` | Многопользовательские сценарии |
| `file-upload.spec.ts` | Загрузка файлов |
| `file-security.spec.ts` | Безопасность доступа к файлам |
| `message-forward.spec.ts` | Пересылка сообщений с файлами |
| `message-display.spec.ts` | Отображение сообщений |

#### E2E тесты в Docker

```bash
# Запуск e2e тестов в контейнере
docker-compose --profile e2e up e2e-tests
```

### Нагрузочное тестирование

```bash
# Собрать генераторы
make build-generators

# Генерация тестовых пользователей
make gen-users count=1000

# Генерация тестовых чатов
make gen-chats count=100 users=users.json

# Нагрузочный тест
make load-test duration=10m users=500

# Или в Docker
docker-compose --profile loadtest up loadgen
```

## CLI для управления пользователями

```bash
# Собрать CLI
make build-users

# Добавить пользователя
./bin/users-cli user add --username admin --email admin@test.local --password secret --role owner

# Список пользователей
./bin/users-cli user list

# Изменить роль
./bin/users-cli user set-role --id <uuid> --role moderator
```

## API

### Основные эндпоинты

#### Аутентификация
- `POST /api/auth/register` — регистрация
- `POST /api/auth/login` — вход
- `POST /api/auth/refresh` — обновление токена
- `POST /api/auth/logout` — выход

#### Пользователи
- `GET /api/users` — список пользователей
- `GET /api/users/{id}` — информация о пользователе
- `GET /api/users/me` — текущий пользователь

#### Чаты
- `POST /api/chats` — создать чат
- `GET /api/chats` — список чатов
- `GET /api/chats/{id}` — информация о чате
- `POST /api/chats/{id}/participants` — добавить участника
- `DELETE /api/chats/{id}/participants/{userId}` — удалить участника

#### Сообщения
- `GET /api/chats/{id}/messages` — сообщения чата
- `POST /api/chats/{id}/messages` — отправить сообщение
- `POST /api/chats/messages/{id}/forward` — переслать сообщение

#### Файлы
- `POST /api/files/upload` — загрузить файл
- `GET /api/files/{linkId}` — скачать файл
- `GET /api/files/{linkId}/info` — метаданные файла

## Ролевая модель

| Роль | Права |
|------|-------|
| `owner` | Полный доступ, управление ролями |
| `moderator` | Модерация чатов и сообщений |
| `user` | Чтение и отправка сообщений |
| `guest` | Только чтение |

## Документация

- [Архитектурные диаграммы](docs/diagrams/file-system-diagrams.md) — последовательности работы с файлами, модель данных

## Конфигурация

Переменные окружения настраиваются в `docker-compose.yml`. Ключевые:

| Переменная | Описание |
|------------|----------|
| `JWT_SECRET` | Секрет для подписи JWT токенов |
| `DATABASE_URL` | Подключение к PostgreSQL |
| `RABBITMQ_URL` | Подключение к RabbitMQ |
| `CENTRIFUGO_SECRET` | Секрет для Centrifugo |

## Лицензия

MIT
