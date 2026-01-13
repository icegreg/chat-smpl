---
modified: 2025-12-10T13:21:22+03:00
created: 2025-11-25T10:59:12+03:00
---
# Config - Управление конфигурациями

## Обзор

Модуль управления конфигурациями предоставляет возможности для хранения, версионирования и применения конфигураций VCS сервисов.

## Типы конфигураций

1. **Global Config** - глобальные настройки системы
2. **Host Config** - конфигурация хостов
3. **Service Config** - конфигурация отдельных сервисов (SIP, SMC, WBS, AUX, API Gateway)

---

## Команды

### `vcs config list`

Список всех конфигураций в системе.

**Синтаксис:**
```bash
vcs config list [flags]
```

**Флаги:**
- `--type <type>` - фильтр по типу (host|service|global)

**Пример 1: Все конфигурации**

```bash
$ vcs config list
```

**Вывод:**

```
CONFIGURATIONS (12):

┌────────────────┬──────────┬─────────┬─────────────────┬───────────────┐
│ Config ID      │ Type     │ Target  │ Modified        │ Status        │
├────────────────┼──────────┼─────────┼─────────────────┼───────────────┤
│ global-001     │ global   │ system  │ 2024-01-10      │ ● applied     │
│ host-node1-01  │ host     │ node-1  │ 2024-01-12      │ ● applied     │
│ host-node2-01  │ host     │ node-2  │ 2024-01-12      │ ● applied     │
│ host-node3-01  │ host     │ node-3  │ 2024-01-12      │ ● applied     │
│ sip01-config   │ service  │ sip-01  │ 2024-01-14      │ ● applied     │
│ smc01-config   │ service  │ smc-01  │ 2024-01-15      │ ● applied     │
│ wbs01-config   │ service  │ wbs-01  │ 2024-01-14      │ ● applied     │
│ aux01-config   │ service  │ aux-01  │ 2024-01-15      │ ● applied     │
│ api-gw01-config│ service  │ api-gw-01│ 2024-01-13     │ ● applied     │
│ smc01-draft    │ service  │ smc-01  │ 2024-01-15      │ ○ draft       │
│ ...            │          │         │                 │               │
└────────────────┴──────────┴─────────┴─────────────────┴───────────────┘
```

**Пример 2: Только конфигурации сервисов**

```bash
$ vcs config list --type service
```

**Вывод:**

```
SERVICE CONFIGURATIONS (6):

┌────────────────┬─────────────┬─────────────────┬───────────────┐
│ Config ID      │ Service     │ Modified        │ Status        │
├────────────────┼─────────────┼─────────────────┼───────────────┤
│ sip01-config   │ sip-01      │ 2024-01-14      │ ● applied     │
│ smc01-config   │ smc-01      │ 2024-01-15      │ ● applied     │
│ wbs01-config   │ wbs-01      │ 2024-01-14      │ ● applied     │
│ aux01-config   │ aux-01      │ 2024-01-15      │ ● applied     │
│ api-gw01-config│ api-gw-01   │ 2024-01-13      │ ● applied     │
│ smc01-draft    │ smc-01      │ 2024-01-15      │ ○ draft       │
└────────────────┴─────────────┴─────────────────┴───────────────┘
```

---

### `vcs config show <config-id>`

Просмотр конфигурации.

**Синтаксис:**
```bash
vcs config show <config-id> [flags]
```

**Параметры:**
- `<config-id>` - ID конфигурации

**Пример 1: SMC Service Config**

```bash
$ vcs config show smc01-config
```

**Вывод:**

```
CONFIG: smc01-config
Type:         service
Target:       smc-01 (node-1)
Status:       ● applied
Created:      2024-01-10 10:00:00 UTC
Modified:     2024-01-15 10:30:00 UTC
Applied:      2024-01-15 10:35:00 UTC

PARAMETERS:

websocket:
  port: 443
  tls: true
  ping_interval: 15s
  ping_timeout: 5s
  max_connections: 10000
  
signaling:
  protocol: json-rpc
  max_message_size: 256KB
  queue_size: 1000
  
sessions:
  timeout: 300s
  max_per_client: 5
  
performance:
  workers: 8
  max_queue_depth: 5000
  
security:
  jwt_enabled: true
  jwt_secret: encrypted:abc123...
  cors_origins:
    - https://app.company.com
    - https://web.company.com
```

**Пример 2: WBS Service Config**

```bash
$ vcs config show wbs01-config
```

**Вывод:**

```
CONFIG: wbs01-config
Type:         service
Target:       wbs-01 (node-1)
Status:       ● applied
Modified:     2024-01-14 15:30:00 UTC
Applied:      2024-01-14 15:35:00 UTC

PARAMETERS:

media:
  mode: [sfu, mcu, hybrid]
  default_mode: sfu
  max_streams: 5000
  max_bitrate_per_stream: 10Mbps
  
codecs:
  video:
    preferred: [VP9, VP8, H264]
    vp9_profile: 0
    h264_profile: baseline
  audio:
    preferred: [opus, g722]
    opus_bitrate: 48kbps
    
ports:
  rtp_start: 10000
  rtp_end: 20000
  
ice:
  stun_servers:
    - stun:stun.example.com:3478
  turn_servers:
    - url: turn:turn.example.com:3478
      username: vcs-media
      credential: encrypted:def456...
      
quality:
  rtt_threshold: 200
  jitter_threshold: 80
  loss_threshold: 5.0
  
bandwidth:
  min: 256kbps
  max: 10Mbps
  start: 1Mbps
```

**Пример 3: AUX Service Config**

```bash
$ vcs config show aux01-config
```

**Вывод:**

```
CONFIG: aux01-config
Type:         service
Target:       aux-01 (node-3)
Status:       ● applied
Modified:     2024-01-15 10:30:00 UTC
Applied:      2024-01-15 10:35:00 UTC

PARAMETERS:

chat:
  enabled: true
  max_channels: 15000
  max_subscribers_per_channel: 1000
  p2p_channels_enabled: true
  
messages:
  max_size: 64KB
  retention_period: 7d
  rate_limit: 100/min
  
websocket:
  port: 8443
  tls: true
  timeout: 30s
  ping_interval: 15s
  reconnect_attempts: 3
  max_connections_per_client: 5
  
presence:
  enabled: true
  update_interval: 30s
  states: [online, away, busy, offline]
  
file_transfer:
  enabled: true
  max_size: 100MB
  allowed_types: [image/*, video/*, application/pdf]
  storage: s3
  
storage:
  backend: redis
  redis_host: 10.0.1.20
  redis_port: 6379
  redis_db: 0
```

---

### `vcs config get <config-id> <key>`

Получить значение конкретного параметра.

**Синтаксис:**
```bash
vcs config get <config-id> <key> [flags]
```

**Параметры:**
- `<config-id>` - ID конфигурации
- `<key>` - путь к параметру (через точку)

**Пример 1: Простое значение**

```bash
$ vcs config get smc01-config websocket.max_connections
```

**Вывод:**

```
smc01-config | websocket.max_connections
Value: 10000
Type:  integer
```

**Пример 2: Значение из AUX конфига**

```bash
$ vcs config get aux01-config chat.max_channels
```

**Вывод:**

```
aux01-config | chat.max_channels
Value: 15000
Type:  integer
```

---

### `vcs config set <config-id> <key> <value>`

Установить значение параметра (создает draft).

**Синтаксис:**
```bash
vcs config set <config-id> <key> <value> [flags]
```

**Параметры:**
- `<config-id>` - ID конфигурации
- `<key>` - путь к параметру
- `<value>` - новое значение

**Пример 1: Изменение max_connections**

```bash
$ vcs config set smc01-config websocket.max_connections 15000
```

**Вывод:**

```
CONFIG: smc01-config
Key:    websocket.max_connections
Change: 10000 → 15000
Status: ○ draft (not applied)

To apply changes:
  vcs config apply smc01-config

To validate before applying:
  vcs config validate smc01-config
```

**Пример 2: Изменение max_channels в чате**

```bash
$ vcs config set aux01-config chat.max_channels 20000
```

**Вывод:**

```
CONFIG: aux01-config
Key:    chat.max_channels
Change: 15000 → 20000
Status: ○ draft (not applied)

⚠ WARNING: This change may require service restart

To apply changes:
  vcs config apply aux01-config --restart
```

---

### `vcs config validate <config-id>`

Валидация конфигурации перед применением.

**Синтаксис:**
```bash
vcs config validate <config-id> [flags]
```

**Пример 1: Валидация SMC конфигурации**

```bash
$ vcs config validate smc01-config
```

**Вывод:**

```
VALIDATING: smc01-config

✓ Syntax valid
✓ Required fields present
✓ Data types correct
✓ Port ranges valid
✓ No conflicts with other configs
⚠ max_connections increased from 10000 to 15000
  Impact: May require more memory (~500MB)

WARNINGS (1):
  Resource impact detected
  
Validation: PASSED

Ready to apply. Use:
  vcs config apply smc01-config
```

**Пример 2: Валидация AUX конфигурации**

```bash
$ vcs config validate aux01-config
```

**Вывод:**

```
VALIDATING: aux01-config

✓ Syntax valid
✓ Required fields present
✓ Data types correct
✓ Redis connection valid
⚠ max_channels increased from 15000 to 20000
  Impact: Requires service restart
⚠ Memory usage will increase (~800MB)

WARNINGS (2):
  Service restart required
  Resource impact detected
  
Validation: PASSED with warnings

Ready to apply. Use:
  vcs config apply aux01-config --restart
```

---

### `vcs config apply <config-id>`

Применить конфигурацию к сервису.

**Синтаксис:**
```bash
vcs config apply <config-id> [flags]
```

**Флаги:**
- `--restart` - перезапустить сервис после применения

**Пример 1: Применение без рестарта**

```bash
$ vcs config apply smc01-config
```

**Вывод:**

```
APPLYING: smc01-config to smc-01

Steps:
  1. Validate configuration           ✓ done
  2. Backup current configuration     ✓ done (backup-smc01-20240115-1530)
  3. Push configuration to service    ✓ done
  4. Reload service configuration     ✓ done
  5. Verify service health            ✓ done

✓ Configuration applied successfully

Changes:
  websocket.max_connections: 10000 → 15000

Service: smc-01 is ● running
Uptime: 14d 6h 28m (no restart required)
```

**Пример 2: Применение с рестартом**

```bash
$ vcs config apply aux01-config --restart
```

**Вывод:**

```
APPLYING: aux01-config to aux-01

Steps:
  1. Validate configuration           ✓ done
  2. Backup current configuration     ✓ done (backup-aux01-20240115-1035)
  3. Push configuration to service    ✓ done
  4. Graceful shutdown                ⏳ in progress...
     - Draining connections (12,450 active)
     - Waiting for message queue flush
     ✓ done (15s)
  5. Service restart                  ✓ done
  6. Verify service health            ✓ done

✓ Configuration applied successfully

Changes:
  chat.max_channels: 15000 → 20000

Service: aux-01 is ● running
Uptime: 5s
Active connections: 12,234 (216 reconnected)
```

---

### `vcs config diff <config-id-1> <config-id-2>`

Сравнение двух конфигураций.

**Синтаксис:**
```bash
vcs config diff <config-id-1> <config-id-2> [flags]
```

**Пример:**

```bash
$ vcs config diff smc01-config smc02-config
```

**Вывод:**

```
DIFF: smc01-config ↔ smc02-config

TARGET:
  smc01-config → smc-01 (node-1)
  smc02-config → smc-02 (node-2)

DIFFERENCES:

  websocket.max_connections:
    smc01-config: 15000
    smc02-config: 10000

  websocket.port:
    smc01-config: 443
    smc02-config: 8443

  performance.workers:
    smc01-config: 8
    smc02-config: 6

COMMON (7 parameters):
  websocket.tls: true
  signaling.protocol: json-rpc
  sessions.timeout: 300s
  security.jwt_enabled: true
  ...
```

---

### `vcs config export <config-id>`

Экспорт конфигурации в файл.

**Синтаксис:**
```bash
vcs config export <config-id> [flags]
```

**Флаги:**
- `--format <format>` - формат вывода (yaml|json), по умолчанию yaml

**Пример 1: YAML формат**

```bash
$ vcs config export smc01-config
```

**Вывод:**

```yaml
# Exported configuration: smc01-config
# Type: service
# Target: smc-01
# Exported: 2024-01-15 16:40:00 UTC

config_id: smc01-config
type: service
target: smc-01
version: 1.2

parameters:
  websocket:
    port: 443
    tls: true
    ping_interval: 15s
    ping_timeout: 5s
    max_connections: 15000
    
  signaling:
    protocol: json-rpc
    max_message_size: 256KB
    queue_size: 1000
    
  sessions:
    timeout: 300s
    max_per_client: 5
    
  performance:
    workers: 8
    max_queue_depth: 5000
    
  security:
    jwt_enabled: true
    jwt_secret: encrypted:abc123...
    cors_origins:
      - https://app.company.com
      - https://web.company.com

Saved to: smc01-config.yaml
```

**Пример 2: JSON формат**

```bash
$ vcs config export aux01-config --format json
```

**Вывод:**

```json
{
  "config_id": "aux01-config",
  "type": "service",
  "target": "aux-01",
  "version": "1.0",
  "parameters": {
    "chat": {
      "enabled": true,
      "max_channels": 20000,
      "max_subscribers_per_channel": 1000,
      "p2p_channels_enabled": true
    },
    "messages": {
      "max_size": "64KB",
      "retention_period": "7d",
      "rate_limit": "100/min"
    },
    "websocket": {
      "port": 8443,
      "tls": true,
      "timeout": "30s",
      "ping_interval": "15s",
      "reconnect_attempts": 3,
      "max_connections_per_client": 5
    },
    "presence": {
      "enabled": true,
      "update_interval": "30s",
      "states": ["online", "away", "busy", "offline"]
    },
    "file_transfer": {
      "enabled": true,
      "max_size": "100MB",
      "allowed_types": ["image/*", "video/*", "application/pdf"],
      "storage": "s3"
    },
    "storage": {
      "backend": "redis",
      "redis_host": "10.0.1.20",
      "redis_port": 6379,
      "redis_db": 0
    }
  }
}

Saved to: aux01-config.json
```

---

### `vcs config import <file>`

Импорт конфигурации из файла.

**Синтаксис:**
```bash
vcs config import <file> [flags]
```

**Параметры:**
- `<file>` - путь к файлу конфигурации (YAML или JSON)

**Пример:**

```bash
$ vcs config import smc03-config.yaml
```

**Вывод:**

```
IMPORTING: smc03-config.yaml

Parsing file...                  ✓ valid YAML
Validating structure...          ✓ valid schema
Checking conflicts...            ✓ no conflicts

CONFIG PREVIEW:
  config_id: smc03-config
  type:      service
  target:    smc-03
  
  Key parameters:
    websocket.max_connections: 10000
    websocket.port: 443
    signaling.protocol: json-rpc

Import this configuration? [y/N]: y

✓ Configuration imported successfully

Config ID: smc03-config
Status:    ○ draft

To apply:
  vcs config apply smc03-config
```

---

### `vcs config history <config-id>`

История изменений конфигурации.

**Синтаксис:**
```bash
vcs config history <config-id> [flags]
```

**Флаги:**
- `--last <n>` - показать последние N изменений

**Пример 1: Полная история**

```bash
$ vcs config history smc01-config
```

**Вывод:**

```
CONFIG HISTORY: smc01-config

┌─────────────────────┬──────────┬──────────────────────────────────────────────┐
│ Timestamp           │ Action   │ Changes                                      │
├─────────────────────┼──────────┼──────────────────────────────────────────────┤
│ 2024-01-15 15:35:00 │ APPLIED  │ websocket.max_connections: 10000 → 15000     │
│ 2024-01-15 15:30:00 │ MODIFIED │ websocket.max_connections: 10000 → 15000     │
│ 2024-01-14 10:20:00 │ APPLIED  │ signaling.max_message_size: 128KB → 256KB    │
│ 2024-01-14 10:15:00 │ MODIFIED │ signaling.max_message_size: 128KB → 256KB    │
│ 2024-01-12 14:00:00 │ APPLIED  │ security.cors_origins: added 2 origins       │
│ 2024-01-12 13:55:00 │ MODIFIED │ security.cors_origins: added 2 origins       │
│ 2024-01-10 10:05:00 │ APPLIED  │ Initial configuration                        │
│ 2024-01-10 10:00:00 │ CREATED  │ Configuration created                        │
└─────────────────────┴──────────┴──────────────────────────────────────────────┘

Total changes: 8
Applied: 4
Drafts: 0

Use 'vcs config show smc01-config@<timestamp>' to see specific version
```

**Пример 2: Последние 5 изменений**

```bash
$ vcs config history aux01-config --last 5
```

**Вывод:**

```
CONFIG HISTORY: aux01-config (last 5)

┌─────────────────────┬──────────┬──────────────────────────────────────────┐
│ Timestamp           │ Action   │ Changes                                  │
├─────────────────────┼──────────┼──────────────────────────────────────────┤
│ 2024-01-15 10:35:00 │ APPLIED  │ chat.max_channels: 15000 → 20000         │
│ 2024-01-15 10:30:00 │ MODIFIED │ chat.max_channels: 15000 → 20000         │
│ 2024-01-14 16:20:00 │ APPLIED  │ websocket.timeout: 60s → 30s             │
│ 2024-01-14 16:15:00 │ MODIFIED │ websocket.timeout: 60s → 30s             │
│ 2024-01-13 11:00:00 │ APPLIED  │ messages.retention_period: 3d → 7d       │
└─────────────────────┴──────────┴──────────────────────────────────────────┘
```

---

## Практические примеры

### Изменение конфигурации сервиса

```bash
# 1. Просмотреть текущую конфигурацию
vcs config show smc01-config

# 2. Изменить параметр
vcs config set smc01-config websocket.max_connections 15000

# 3. Валидировать изменения
vcs config validate smc01-config

# 4. Применить конфигурацию
vcs config apply smc01-config
```

### Копирование конфигурации между сервисами

```bash
# 1. Экспортировать конфигурацию
vcs config export smc01-config -o smc-template.yaml

# 2. Отредактировать файл (изменить target и config_id)
vim smc-template.yaml

# 3. Импортировать как новую конфигурацию
vcs config import smc-template.yaml

# 4. Применить к другому сервису
vcs config apply smc03-config
```

### Сравнение конфигураций

```bash
# Сравнить конфигурации двух SMC сервисов
vcs config diff smc01-config smc02-config

# Сравнить текущую с backup версией
vcs config diff smc01-config smc01-config@2024-01-14

# Найти различия в WBS конфигурациях
vcs config diff wbs01-config wbs02-config
```

### Откат конфигурации

```bash
# 1. Посмотреть историю
vcs config history smc01-config

# 2. Экспортировать старую версию
vcs config export smc01-config@2024-01-14 -o smc01-rollback.yaml

# 3. Импортировать как новую конфигурацию
vcs config import smc01-rollback.yaml

# 4. Применить
vcs config apply smc01-rollback --restart
```

---

## JSON вывод для автоматизации

```bash
# Экспорт списка конфигураций
vcs config list -o json

# Экспорт конфигурации в JSON
vcs config export smc01-config --format json

# История в JSON
vcs config history smc01-config -o json

# Обработка через jq
vcs config list -o json | jq '.[] | select(.status=="draft")'
```

---

## Автоматизация

### Массовое обновление конфигураций

```bash
#!/bin/bash
# update_all_smc_configs.sh

PARAM="websocket.max_connections"
VALUE="15000"

# Получить все SMC конфигурации
vcs config list --type service -o json | \
  jq -r '.[] | select(.target | startswith("smc-")) | .config_id' | \
  while read config; do
    echo "Updating $config..."
    
    # Обновить параметр
    vcs config set $config $PARAM $VALUE
    
    # Валидировать
    vcs config validate $config
    
    # Применить
    vcs config apply $config
  done
```

### Backup всех конфигураций

```bash
#!/bin/bash
# backup_all_configs.sh

BACKUP_DIR="config_backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

vcs config list -o json | jq -r '.[].config_id' | while read config; do
  echo "Backing up $config..."
  vcs config export $config --format yaml > "$BACKUP_DIR/$config.yaml"
done

echo "Backup completed: $BACKUP_DIR"
```

### Проверка дрейфа конфигураций

```bash
#!/bin/bash
# check_config_drift.sh

# Сравнить конфигурации одинаковых сервисов
for service in smc wbs; do
  echo "=== Checking $service configurations ==="
  
  CONFIGS=$(vcs config list --type service -o json | \
    jq -r '.[] | select(.target | startswith("'$service'-")) | .config_id')
  
  # Сравнить первую со всеми остальными
  FIRST=$(echo "$CONFIGS" | head -1)
  echo "$CONFIGS" | tail -n +2 | while read config; do
    echo "Comparing $FIRST with $config:"
    vcs config diff $FIRST $config
  done
done
```

---

## Интеграция с сервисами

### Просмотр активной конфигурации сервиса

```bash
# Информация о сервисе с конфигурацией
vcs service smc-01 | grep "Config ID"

# Полная конфигурация
vcs service smc-01
vcs config show smc01-config
```

### Применение конфигурации и мониторинг

```bash
#!/bin/bash
# apply_and_monitor.sh

CONFIG_ID="$1"
SERVICE_ID=$(vcs config show $CONFIG_ID -o json | jq -r '.target')

echo "Applying configuration $CONFIG_ID to $SERVICE_ID..."

# Применить конфигурацию
vcs config apply $CONFIG_ID

# Подождать 5 секунд
sleep 5

# Проверить статус сервиса
vcs service $SERVICE_ID status

# Проверить метрики
vcs service $SERVICE_ID
```

---

## См. также

- [Service - Управление сервисами](02_Service.md) - применение конфигураций к сервисам
- [Host - Управление хостами](01_Host.md) - конфигурации хостов
- [README - Главная страница](README.md)
