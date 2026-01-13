# Руководство по работе с rtuccli

**rtuccli** - утилита командной строки для управления и мониторинга chat-smpl системы.

## Содержание

1. [Быстрый старт](#быстрый-старт)
2. [Установка и настройка](#установка-и-настройка)
3. [Основные команды](#основные-команды)
4. [Примеры использования](#примеры-использования)
5. [Интеграция с Docker](#интеграция-с-docker)
6. [Troubleshooting](#troubleshooting)

---

## Быстрый старт

### Запуск admin-service

Сначала необходимо запустить admin-service:

```bash
# Сборка и запуск admin-service
docker-compose up -d admin-service

# Проверка что сервис запущен
docker-compose ps admin-service
docker-compose logs admin-service

# Проверка health endpoint
curl http://localhost:8086/health
```

### Первая команда CLI

```bash
# Список всех сервисов
docker-compose exec admin-service sh -c 'echo "rtuccli service list"'

# Или если CLI собран локально
cd cmd/rtuccli
go build -o rtuccli
./rtuccli --server http://localhost:8086 service list
```

---

## Установка и настройка

### Вариант 1: Использование через Docker (рекомендуется)

Admin-service уже содержит готовую команду CLI. Используйте алиас для удобства:

#### Windows PowerShell (рекомендуется для Windows)

**Автоматическая настройка:**
```powershell
# Запустить скрипт автоматической настройки
powershell -ExecutionPolicy Bypass -File scripts\setup-rtuccli-alias.ps1

# Или если вы уже в PowerShell
.\scripts\setup-rtuccli-alias.ps1

# Перезагрузить профиль
. $PROFILE
```

**Ручная настройка:**
```powershell
# Открыть PowerShell профиль в редакторе
notepad $PROFILE

# Добавить функцию в конец файла:
function rtuccli {
    param(
        [Parameter(ValueFromRemainingArguments=$true)]
        [string[]]$Arguments
    )
    $cmd = $Arguments -join ' '
    docker-compose exec -T admin-service sh -c $cmd
}

# Сохранить и перезагрузить профиль
. $PROFILE

# Использование
rtuccli service list
rtuccli conf list --status active
```

#### Windows CMD

**Автоматическая настройка:**
```cmd
REM Запустить скрипт (работает только в текущей сессии)
scripts\setup-rtuccli-alias.bat

REM Использование
rtuccli service list
```

**ВАЖНО:** CMD doskey макросы работают только в текущей сессии. Для постоянного алиаса используйте PowerShell.

**Ручная настройка:**
```cmd
REM Создать алиас для текущей сессии CMD
doskey rtuccli=docker-compose exec -T admin-service sh -c $*

REM Использование
rtuccli service list
```

#### Linux (bash/zsh)

**Автоматическая настройка:**
```bash
# Запустить скрипт автоматической настройки
bash scripts/setup-rtuccli-alias.sh

# Перезагрузить shell конфигурацию
source ~/.bashrc  # для bash
source ~/.zshrc   # для zsh
```

**Ручная настройка:**
```bash
# Для bash - добавить в ~/.bashrc
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.bashrc
source ~/.bashrc

# Для zsh - добавить в ~/.zshrc
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.zshrc
source ~/.zshrc

# Использование
rtuccli 'service list'
rtuccli 'conf list --status active'
```

#### macOS (zsh/bash)

**Автоматическая настройка:**
```bash
# macOS Catalina+ использует zsh по умолчанию
bash scripts/setup-rtuccli-alias.sh

# Перезагрузить конфигурацию
source ~/.zshrc   # для zsh (по умолчанию)
source ~/.bashrc  # для bash
```

**Ручная настройка:**
```bash
# Для zsh (по умолчанию в macOS Catalina+)
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.zshrc
source ~/.zshrc

# Для bash (старые версии macOS)
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.bashrc
source ~/.bashrc

# Использование
rtuccli 'service list'
rtuccli 'conf list --status active'
```

### Вариант 2: Локальная сборка (для разработки)

```bash
cd cmd/rtuccli
go build -o rtuccli

# Для Windows
go build -o rtuccli.exe

# Использование
./rtuccli --server http://localhost:8086 service list
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `ADMIN_SERVICE_URL` | URL admin-service API | `http://localhost:8086` |

Установите переменную окружения:
```bash
export ADMIN_SERVICE_URL=http://admin-service:8086
```

---

## Основные команды

### Conference Management (`conf`)

#### Список конференций

```bash
# Все конференции
rtuccli conf list

# Только активные
rtuccli conf list --status active

# Только завершённые
rtuccli conf list --status ended

# Только запланированные
rtuccli conf list --status scheduled
```

**Вывод:**
```
CONFERENCES (5):

┌──────────┬────────────────┬──────────┬────────┬──────────────┬──────────┬─────────────────┐
│Conf ID   │Name            │Type      │Status  │Participants  │Duration  │Started          │
├──────────┼────────────────┼──────────┼────────┼──────────────┼──────────┼─────────────────┤
│123e4567  │Team Meeting    │adhoc     │active  │5             │30m 15s   │2024-01-20 10:00 │
│234f5678  │Daily Standup   │scheduled │ended   │8             │15m 30s   │2024-01-20 09:00 │
└──────────┴────────────────┴──────────┴────────┴──────────────┴──────────┴─────────────────┘
```

#### Детали конференции

```bash
rtuccli conf get <conference-id>
```

**Пример:**
```bash
rtuccli conf get 123e4567-e89b-12d3-a456-426614174000
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

Use 'rtuccli conf clients 123e4567-...' to see participants
```

#### Список участников

```bash
rtuccli conf clients <conference-id>
```

**Вывод:**
```
CONFERENCE PARTICIPANTS (5):

┌──────────┬──────────┬───────────┬──────────┬──────────┐
│Username  │Extension │Status     │Joined    │Duration  │
├──────────┼──────────┼───────────┼──────────┼──────────┤
│user1     │1001      │connected  │10:00:15  │30m 0s    │
│user2     │1002      │connected  │10:01:22  │29m 0s    │
│user3     │1003      │connected  │10:05:10  │25m 5s    │
└──────────┴──────────┴───────────┴──────────┴──────────┘
```

### Service Monitoring (`service`)

#### Список сервисов

```bash
rtuccli service list
```

**Вывод:**
```
SERVICES (5):

┌──────────────┬─────────────────┬─────────┬───────────┬─────────┬────────────┐
│Service ID    │Name             │Type     │Status     │Health   │Last Check  │
├──────────────┼─────────────────┼─────────┼───────────┼─────────┼────────────┤
│voice-service │Voice Service    │voice    │● running  │healthy  │5s ago      │
│api-gateway   │API Gateway      │gateway  │● running  │healthy  │5s ago      │
│users-service │Users Service    │users    │● running  │healthy  │5s ago      │
│files-service │Files Service    │files    │● running  │healthy  │5s ago      │
│admin-service │Admin Service    │admin    │● running  │healthy  │5s ago      │
└──────────────┴─────────────────┴─────────┴───────────┴─────────┴────────────┘
```

#### Детали сервиса

```bash
rtuccli service get <service-id>
```

**Пример:**
```bash
rtuccli service get voice-service
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

---

## Примеры использования

### Мониторинг активных конференций

```bash
# Проверить все активные конференции
rtuccli conf list --status active

# В режиме watch (обновление каждые 5 секунд)
watch -n 5 'rtuccli conf list --status active'

# Посчитать количество активных конференций
rtuccli conf list --status active | grep -c "active"
```

### Проверка участников конференции

```bash
# 1. Получить ID активной конференции
CONF_ID=$(curl -s http://localhost:8086/api/conferences?status=active | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

# 2. Показать участников
rtuccli conf clients $CONF_ID
```

### Проверка статуса всех сервисов

```bash
# Показать статус всех сервисов
rtuccli service list

# Проверить конкретный сервис
rtuccli service get voice-service

# Проверить только упавшие сервисы
rtuccli service list | grep "error"
```

### Скрипт автоматической проверки

Создайте файл `check-system.sh`:

```bash
#!/bin/bash
# check-system.sh - Проверка состояния системы

echo "=== Проверка сервисов ==="
rtuccli service list

echo ""
echo "=== Активные конференции ==="
rtuccli conf list --status active

echo ""
echo "=== Статистика ==="
ACTIVE_CONFS=$(rtuccli conf list --status active | grep -c "active" || echo "0")
echo "Активных конференций: $ACTIVE_CONFS"

# Алерт если много конференций
if [ $ACTIVE_CONFS -gt 10 ]; then
  echo "⚠️  ВНИМАНИЕ: Много активных конференций ($ACTIVE_CONFS)"
fi
```

Запуск:
```bash
chmod +x check-system.sh
./check-system.sh
```

### Мониторинг с автоматическим алертом

```bash
#!/bin/bash
# monitor-alerts.sh - Мониторинг с отправкой алертов

while true; do
  # Проверка упавших сервисов
  DOWN_SERVICES=$(rtuccli service list | grep "error" | wc -l)

  if [ $DOWN_SERVICES -gt 0 ]; then
    echo "⚠️  ALERT: $DOWN_SERVICES сервисов не работают!"
    rtuccli service list | grep "error"
    # Здесь можно добавить отправку в Slack/Telegram
  fi

  # Проверка зависших конференций
  LONG_CONFS=$(curl -s http://localhost:8086/api/conferences?status=active | jq '.conferences[] | select(.duration_seconds > 7200) | .id' | wc -l)

  if [ $LONG_CONFS -gt 0 ]; then
    echo "⚠️  ALERT: $LONG_CONFS конференций идут более 2 часов"
  fi

  sleep 60
done
```

---

## Интеграция с Docker

### Запуск CLI внутри контейнера

```bash
# Выполнить одну команду
docker-compose exec admin-service rtuccli conf list

# Интерактивная сессия
docker-compose exec admin-service sh
# Теперь внутри контейнера:
rtuccli service list
rtuccli conf list
```

### Использование с docker run

```bash
# Собрать отдельный образ для CLI
docker build -t rtuccli -f cmd/rtuccli/Dockerfile .

# Запустить команду
docker run --rm --network chatapp-network \
  -e ADMIN_SERVICE_URL=http://admin-service:8086 \
  rtuccli conf list
```

### Создание алиаса для удобства

Используйте автоматические скрипты настройки (см. раздел [Установка и настройка](#установка-и-настройка)):

**Windows PowerShell:**
```powershell
.\scripts\setup-rtuccli-alias.ps1
```

**Windows CMD:**
```cmd
scripts\setup-rtuccli-alias.bat
```

**Linux/macOS:**
```bash
bash scripts/setup-rtuccli-alias.sh
```

После настройки можно использовать:
```bash
rtuccli service list
rtuccli conf list --status active
```

---

## Troubleshooting

### Ошибка: "Error: HTTP request failed: connection refused"

**Проблема:** admin-service не запущен или недоступен.

**Решение:**
```bash
# Проверить что admin-service запущен
docker-compose ps admin-service

# Если не запущен - запустить
docker-compose up -d admin-service

# Проверить логи
docker-compose logs admin-service

# Проверить health
curl http://localhost:8086/health
```

### Ошибка: "relation voice.conferences does not exist"

**Проблема:** Не запущены миграции БД.

**Решение:**
```bash
# Запустите миграции для voice service
docker-compose up -d voice-service

# Проверьте логи voice-service
docker-compose logs voice-service | grep "migration"
```

### CLI показывает "No conferences found"

**Возможные причины:**
1. Нет активных конференций в системе
2. Проблемы с подключением к БД в admin-service

**Диагностика:**
```bash
# Проверить API напрямую
curl http://localhost:8086/api/conferences

# Проверить логи admin-service
docker-compose logs admin-service | tail -20

# Проверить подключение к БД
docker-compose exec admin-service sh -c 'wget -q -O- http://localhost:8086/health'
```

### В Docker на Windows пути не работают

**Проблема:** Git Bash конфликтует с Docker путями.

**Решение:** Используйте PowerShell или CMD вместо Git Bash:
```powershell
# PowerShell
docker-compose exec admin-service rtuccli service list
```

### CLI работает медленно

**Причина:** Каждый запуск CLI создает новое HTTP соединение.

**Решение:** Используйте прямые API вызовы для скриптов:
```bash
# Вместо CLI
curl -s http://localhost:8086/api/services | jq

# Или кешируйте результаты
rtuccli service list > /tmp/services.txt
cat /tmp/services.txt
```

---

## API Endpoints (для продвинутых сценариев)

Если CLI не подходит, используйте прямые API вызовы:

```bash
# Список конференций
curl http://localhost:8086/api/conferences

# Фильтр по статусу
curl http://localhost:8086/api/conferences?status=active

# Детали конференции
curl http://localhost:8086/api/conferences/{id}

# Участники
curl http://localhost:8086/api/conferences/{id}/participants

# Список сервисов
curl http://localhost:8086/api/services

# Детали сервиса
curl http://localhost:8086/api/services/{id}

# Prometheus метрики
curl http://localhost:8086/metrics
```

---

## Дополнительные ресурсы

- [Admin Service README](services/admin/README.md) - Документация REST API
- [CLI README](cmd/rtuccli/README.md) - Подробная документация CLI
- [arch/CLI/README.md](arch/CLI/README.md) - Спецификация архитектуры
- [Grafana Dashboard](http://localhost:3000) - Визуальный мониторинг

---

## Поддержка

Если у вас возникли проблемы:

1. Проверьте логи: `docker-compose logs admin-service`
2. Проверьте health: `curl http://localhost:8086/health`
3. Проверьте API: `curl http://localhost:8086/api/services`
4. Создайте issue в репозитории проекта

**Версия:** 1.0.0
**Дата:** 2026-01-05
