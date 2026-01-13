# rtuccli - Chat-SMPL CLI Tool

Консольная утилита для управления и мониторинга chat-smpl системы.

## Установка

### Из исходников

```bash
cd cmd/rtuccli
go build -o rtuccli
```

### В Docker контейнере

```bash
# Запустить CLI внутри контейнера admin-service
docker-compose exec admin-service sh

# Или собрать отдельный образ для CLI
docker build -t rtuccli -f cmd/rtuccli/Dockerfile .
```

## Конфигурация

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `ADMIN_SERVICE_URL` | URL admin-service API | `http://localhost:8086` |

### Флаги командной строки

```bash
rtuccli --server http://admin-service:8086 conf list
```

## Команды

### Conference Management (conf)

#### Список конференций

```bash
# Все конференции
rtuccli conf list

# Только активные
rtuccli conf list --status active

# Только завершённые
rtuccli conf list --status ended
```

**Вывод:**
```
CONFERENCES (3):

┌──────────────┬─────────────────┬─────────┬────────┬──────────────┬──────────┬─────────────────┐
│ Conference ID│ Name            │ Type    │ Status │ Participants │ Duration │ Started         │
├──────────────┼─────────────────┼─────────┼────────┼──────────────┼──────────┼─────────────────┤
│ 123e4567     │ Team Meeting    │ adhoc   │ active │ 5            │ 30m 15s  │ 2024-01-20 10:00│
│ 234f5678     │ Daily Standup   │ scheduled│ ended │ 8            │ 15m 30s  │ 2024-01-20 09:00│
└──────────────┴─────────────────┴─────────┴────────┴──────────────┴──────────┴─────────────────┘
```

#### Детали конференции

```bash
rtuccli conf get <conference-id>
```

**Вывод:**
```
CONFERENCE: 123e4567-e89b-12d3-a456-426614174000
Name:         Team Meeting
Type:         adhoc
Status:       active
Participants: 5
Started:      2024-01-20 10:00:00
Duration:     30m 15s
Chat ID:      abc12345-6789-...

Use 'rtuccli conf clients 123e4567-e89b-12d3-a456-426614174000' to see participants
```

#### Список участников

```bash
rtuccli conf clients <conference-id>
```

**Вывод:**
```
CONFERENCE PARTICIPANTS (5):

┌───────────┬───────────┬───────────┬──────────┬──────────┐
│ Username  │ Extension │ Status    │ Joined   │ Duration │
├───────────┼───────────┼───────────┼──────────┼──────────┤
│ user1     │ 1001      │ connected │ 10:00:15 │ 30m 0s   │
│ user2     │ 1002      │ connected │ 10:01:22 │ 29m 0s   │
│ user3     │ 1003      │ connected │ 10:05:10 │ 25m 5s   │
└───────────┴───────────┴───────────┴──────────┴──────────┘
```

### Service Monitoring (service)

#### Список сервисов

```bash
rtuccli service list
```

**Вывод:**
```
SERVICES (5):

┌──────────────┬─────────────────┬─────────┬───────────┬─────────┬────────────┐
│ Service ID   │ Name            │ Type    │ Status    │ Health  │ Last Check │
├──────────────┼─────────────────┼─────────┼───────────┼─────────┼────────────┤
│ voice-service│ Voice Service   │ voice   │ ● running │ healthy │ 5s ago     │
│ api-gateway  │ API Gateway     │ gateway │ ● running │ healthy │ 5s ago     │
│ users-service│ Users Service   │ users   │ ● running │ healthy │ 5s ago     │
│ files-service│ Files Service   │ files   │ ● running │ healthy │ 5s ago     │
└──────────────┴─────────────────┴─────────┴───────────┴─────────┴────────────┘
```

#### Детали сервиса

```bash
rtuccli service get <service-id>
```

**Вывод:**
```
SERVICE: voice-service
Name:        Voice Service
Type:        voice
Status:      ● running
Health:      healthy
Last Check:  2024-01-20 10:30:15 (10s ago)
```

## Примеры использования

### Мониторинг активных конференций

```bash
# Проверить активные конференции каждые 5 секунд
watch -n 5 rtuccli conf list --status active
```

### Проверка статуса всех сервисов

```bash
# Проверить статус всех сервисов
rtuccli service list

# Детали конкретного сервиса
rtuccli service get voice-service
```

### Проверка участников конференции

```bash
# Получить ID активной конференции
CONF_ID=$(rtuccli conf list --status active -o json | jq -r '.conferences[0].id')

# Показать участников
rtuccli conf clients $CONF_ID
```

### Скрипты автоматизации

```bash
#!/bin/bash
# check-conferences.sh - Проверка конференций и алерт

# Получить количество активных конференций
ACTIVE=$(rtuccli conf list --status active | grep -c "active")

if [ $ACTIVE -gt 10 ]; then
  echo "ALERT: Too many active conferences: $ACTIVE"
  # Отправить алерт
fi
```

## Интеграция с Docker

### Запуск CLI в контейнере

```bash
# Выполнить команду в контейнере admin-service
docker-compose exec admin-service rtuccli conf list

# Или с алиасом
alias rtuccli='docker-compose exec admin-service rtuccli'
```

### Создание отдельного CLI контейнера

```dockerfile
# cmd/rtuccli/Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY cmd/rtuccli/ ./
RUN go build -o rtuccli

FROM alpine:latest
COPY --from=builder /app/rtuccli /usr/local/bin/
ENTRYPOINT ["rtuccli"]
```

Использование:
```bash
docker build -t rtuccli -f cmd/rtuccli/Dockerfile .
docker run --rm --network chatapp-network rtuccli conf list
```

## Расширение функционала

### Добавление новых команд

Создайте новый файл в `cmd/rtuccli/cmd/`:

```go
// cmd/rtuccli/cmd/mycommand.go
package cmd

import (
    "github.com/spf13/cobra"
)

var myCmd = &cobra.Command{
    Use:   "my",
    Short: "My custom command",
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(myCmd)
}
```

## Troubleshooting

### Ошибка подключения к API

```
Error: HTTP request failed: dial tcp 127.0.0.1:8086: connect: connection refused
```

**Решение:**
- Проверьте что admin-service запущен: `docker-compose ps admin-service`
- Проверьте правильность URL: `--server http://admin-service:8086`
- В Docker используйте имя сервиса вместо localhost

### Пустой вывод

```
No conferences found
```

**Причины:**
- Нет активных конференций в системе
- Проблемы с подключением к БД в admin-service
- Проверьте логи: `docker-compose logs admin-service`

## См. также

- [Admin Service README](../../services/admin/README.md) - Документация REST API
- [arch/CLI/README.md](../../arch/CLI/README.md) - Спецификация CLI архитектуры
