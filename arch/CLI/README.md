---
modified: 2025-11-25T11:15:10+03:00
created: 2025-11-25T10:59:12+03:00
---
# RTUC CLI - Документация

## Описание

CLI инструмент для управления и мониторинга RTUC (модуль видеоконференцсвязь РТУ) сервером. Предоставляет возможности для:

- Мониторинга хостов и системных ресурсов
- Управления сервисами (SIP, SMC, WBS, AUX, API Gateway и пр.)
- Просмотра активных мероприятий и участников
- Анализа качества соединений
- Управления конфигурациями

## Архитектура системы

```
rtuccli Server
├── Hosts (физические/виртуальные серверы)
│   ├── Network Interfaces
│   ├── Services (микросервисы rtuccli)
│   └── System Resources
├── Services
│   ├── sip-service       # SIP signaling (voice/video calls setup)
│   ├── smc-service       # Signaling & Media Controller (WebSocket server)
│   ├── wbs-service       # WebRTC Bridge Service (media-core)
│   ├── aux-service       # Auxiliary service (chat, presence, etc)
│   └── api-gw-service    # Public API Gateway
├── Conferences (мероприятия)
│   ├── Media Mode (MCU/SFU/Hybrid)
│   ├── Chat Channels
│   └── Clients (в контексте мероприятия)
│       └── Peers
│           ├── WebRTC/RTP connections
│           ├── WebSocket connections
│           └── Quality Score
└── Configurations
    ├── Global Config
    ├── Host Config
    └── Service Config
```

## Структура команд

```
rtuccli
├── host          # Управление хостами
├── service       # Управление сервисами
├── conf          # Управление мероприятиями
├── client        # Управление клиентами
└── config        # Управление конфигурациями
```

## Документация по разделам

1. **[01_Host.md](01_Host.md)** - Управление хостами
   - Просмотр списка хостов
   - Мониторинг сетевых интерфейсов
   - Анализ портов и сокетов
   - Системная статистика

2. **[02_Service.md](02_Service.md)** - Управление сервисами
   - SIP Service (сигнализация)
   - SMC Service (WebSocket сервер)
   - WBS Service (медиа-ядро)
   - AUX Service (чат и вспомогательные функции)
   - API Gateway Service

3. **[03_Conference.md](03_Conference.md)** - Управление мероприятиями
   - Список мероприятий
   - Информация о клиентах
   - Анализ пиров и соединений
   - Чат мероприятий

4. **[04_Client.md](04_Client.md)** - Управление клиентами
   - Поиск клиентов
   - История участия в мероприятиях
   - Статистика клиентов

5. **[05_Config.md](05_Config.md)** - Управление конфигурациями
   - Просмотр конфигураций
   - Изменение параметров
   - Валидация и применение
   - История изменений

## Глобальные флаги

```bash
--output, -o    json|yaml|table|wide     # формат вывода
--server, -s    <server-url>             # адрес rtuccli API
--config, -c    <path>                   # путь к конфигу
--verbose, -v                            # подробный вывод
--watch, -w                              # live обновление
--no-color                               # без цветов
```

## Quality Score Reference

Оценка качества соединений:

```
★★★★ Excellent:  RTT <50ms,  Jitter <20ms,  Loss <1%
★★★☆ Good:       RTT <100ms, Jitter <40ms,  Loss <3%
★★☆☆ Normal:     RTT <200ms, Jitter <80ms,  Loss <5%
★☆☆☆ Poor:       Above thresholds
```

## Быстрый старт

```bash
# Просмотр всех хостов
rtuccli host list

# Информация о конкретном хосте
rtuccli host node-1

# Список всех сервисов
rtuccli service list

# Активные мероприятия
rtuccli conf list --active

# Поиск клиента
rtuccli client find "user@example.com"

# Просмотр конфигурации
rtuccli config show smc01-config
```

## Примеры использования

### Мониторинг системы

```bash
# Live мониторинг хоста
rtuccli host node-1 stats --watch

# Список сервисов определенного типа
rtuccli service list --type wbs

# Статистика WebSocket соединений
rtuccli service smc-01 websockets
```

### Анализ мероприятий

```bash
# Клиенты с плохим качеством
rtuccli conf conf-456 clients --quality poor

# Детали конкретного клиента в мероприятии
rtuccli conf conf-456 client frank@company.com

# Каналы данных пира
rtuccli conf conf-456 peer peer-f1 channels
```

### Управление конфигурациями

```bash
# Изменение параметра
rtuccli config set smc01-config websocket.max_connections 15000

# Валидация перед применением
rtuccli config validate smc01-config

# Применение конфигурации
rtuccli config apply smc01-config
```

## Интеграция с другими инструментами

- необходимо продумать сервис через который можно получать такую же или расширенную информацию.
- сервис должен отдавать логи, как полные, так и по сервисам 
## Дополнительная информация

- **Версия CLI**: 1.0.0-rc-0
- **Совместимость**: -
- **Протокол API**: REST + WebSocket
- **Аутентификация**: JWT / API Key


