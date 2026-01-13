---
modified: 2025-11-25T15:09:17+03:00
created: 2025-11-25T10:59:12+03:00
---
# Host - Управление хостами

## Обзор

Модуль управления хостами предоставляет возможности для мониторинга физических и виртуальных серверов RTUC системы.

## Команды

### `rtuccli host list`

Список всех хостов в системе.

**Синтаксис:**
```bash
rtuccli host list [flags]
```

**Пример:**

```bash
$ rtuccli host list
```

**Вывод:**

```
HOSTS (4):

┌─────────────┬─────────────────┬────────┬─────┬─────────┬──────────┬──────────┐
│ Host ID     │ Address         │ Role   │ LA  │ Memory  │ Disk     │ Status   │
├─────────────┼─────────────────┼────────┼─────┼─────────┼──────────┼──────────┤
│ node-1      │ 10.0.1.10       │ media  │ 2.4 │ 64%     │ 45%      │ ● online │
│ node-2      │ 10.0.1.11       │ media  │ 1.8 │ 52%     │ 38%      │ ● online │
│ node-3      │ 10.0.1.12       │ chat   │ 0.9 │ 31%     │ 22%      │ ● online │
│ node-4      │ 10.0.1.13       │ edge   │ 3.1 │ 78%     │ 61%      │ ● online │
└─────────────┴─────────────────┴────────┴─────┴─────────┴──────────┴──────────┘
```

---

### `rtuccli host <host-id>`

Детальная информация о хосте.

**Синтаксис:**
```bash
rtuccli host <host-id> [flags]
```

**Параметры:**
- `<host-id>` - ID хоста (например, node-1)

**Пример:**

```bash
$ rtuccli host node-1
```

**Вывод:**

```
HOST: node-1
Address:      10.0.1.10
Role:         media (SFU/MCU)
Status:       ● online
Uptime:       14d 6h 23m

RESOURCES:
  Load Avg:   2.4 / 2.1 / 1.8 (1m / 5m / 15m)
  Memory:     24.6 GB / 38.4 GB (64%)
  Swap:       0 GB / 8 GB (0%)
  Disk /:     180 GB / 400 GB (45%)

SERVICES (3):
  sip-01        ● running    CPU: 18%   MEM: 2.4 GB
  smc-01        ● running    CPU: 35%   MEM: 4.2 GB
  wbs-01        ● running    CPU: 45%   MEM: 8.2 GB

NETWORK:
  eth0          10.0.1.10       ↑ 245 Mbps   ↓ 892 Mbps
  eth1          192.168.1.10    ↑ 12 Mbps    ↓ 8 Mbps
```

---

### `rtuccli host <host-id> net`

Информация о сетевых интерфейсах хоста.

**Синтаксис:**
```bash
rtuccli host <host-id> net [flags]
```

**Флаги:**
- `-i, --interface <name>` - показать только конкретный интерфейс
- `--bandwidth` - только информация о bandwidth
- `--io` - только IO статистика

**Пример 1: Все интерфейсы**

```bash
$ rtuccli host node-1 net
```

**Вывод:**

```
HOST: node-1 | Network Interfaces

┌───────────┬─────────────────┬───────────┬───────────┬───────────┬───────────┐
│ Interface │ Address         │ TX Rate   │ RX Rate   │ TX Total  │ RX Total  │
├───────────┼─────────────────┼───────────┼───────────┼───────────┼───────────┤
│ eth0      │ 10.0.1.10       │ 245 Mbps  │ 892 Mbps  │ 2.4 TB    │ 8.9 TB    │
│ eth1      │ 192.168.1.10    │ 12 Mbps   │ 8 Mbps    │ 124 GB    │ 89 GB     │
│ lo        │ 127.0.0.1       │ 1.2 Mbps  │ 1.2 Mbps  │ 45 GB     │ 45 GB     │
└───────────┴─────────────────┴───────────┴───────────┴───────────┴───────────┘

IO STATS (eth0):
  Packets:    TX 1.2M/s    RX 3.4M/s
  Errors:     TX 0         RX 0
  Dropped:    TX 0         RX 12
```

**Пример 2: Конкретный интерфейс**

```bash
$ rtuccli host node-1 net -i eth0
```

**Вывод:**

```
HOST: node-1 | Interface: eth0

ADDRESS:      10.0.1.10/24
MAC:          00:1a:2b:3c:4d:5e
MTU:          9000
State:        UP

BANDWIDTH:
  TX Rate:    245 Mbps (peak 1.2 Gbps)
  RX Rate:    892 Mbps (peak 2.8 Gbps)
  TX Total:   2.4 TB
  RX Total:   8.9 TB

IO:
  TX Packets: 1,245,892/s
  RX Packets: 3,421,004/s
  TX Errors:  0
  RX Errors:  0
  TX Dropped: 0
  RX Dropped: 12

QUEUES:
  TX Queue:   1000 (len) / 0 (dropped)
  RX Queue:   1000 (len) / 12 (dropped)
```

---

### `rtuccli host <host-id> ports`

Информация об открытых портах и сервисах.

**Синтаксис:**
```bash
rtuccli host <host-id> ports [flags]
```

**Флаги:**
- `--all-ports` - показать все порты хоста, не только порты rtuccli сервисов

**Пример 1: Все порты**

```bash
$ rtuccli host node-1 ports --all-ports
```

**Вывод:**

```
HOST: node-1 | Open Ports

┌─────────┬──────────┬─────────────┬─────────────────┬─────────────┐
│ Port    │ Protocol │ Service     │ PID/Process     │ Connections │
├─────────┼──────────┼─────────────┼─────────────────┼─────────────┤
│ 443     │ TCP      │ smc-01      │ 12345/node      │ 3,421       │
│ 5060    │ UDP      │ sip-01      │ 12340/kamailio  │ 456         │
│ 5061    │ TCP      │ sip-01      │ 12340/kamailio  │ 124         │
│ 10000-  │ UDP      │ wbs-01      │ 12350/mediasoup │ 4,521       │
│  20000  │          │             │                 │             │
└─────────┴──────────┴─────────────┴─────────────────┴─────────────┘
```

**Пример 2: Только rtuccli порты**

```bash
$ rtuccli host node-1 ports
```

**Вывод:**

```
HOST: node-1 | rtuccli Ports

┌─────────────────┬──────────┬─────────────┬─────────────┐
│ Port Range      │ Protocol │ Service     │ Connections │
├─────────────────┼──────────┼─────────────┼─────────────┤
│ 443             │ TCP      │ smc-01      │ 3,421       │
│ 5060            │ UDP      │ sip-01      │ 456         │
│ 5061            │ TCP      │ sip-01      │ 124         │
│ 10000-20000     │ UDP      │ wbs-01      │ 4,521       │
└─────────────────┴──────────┴─────────────┴─────────────┘
```

---

### `rtuccli host <host-id> sockets`

Информация об активных сокетах.

**Синтаксис:**
```bash
rtuccli host <host-id> sockets [flags]
```

**Флаги:**
- `--all` - показать все сокеты (по умолчанию первые 20)

**Пример:**

```bash
$ rtuccli host node-1 sockets
```

**Вывод:**

```
HOST: node-1 | Sockets

SUMMARY:
  TCP Established:  2,456
  TCP Listen:       12
  TCP Time-Wait:    342
  UDP Active:       7,517

rtuccli SOCKETS:
┌─────────────┬──────────┬───────────────────────┬───────────────────────┬─────────┐
│ Service     │ Protocol │ Local                 │ Remote                │ State   │
├─────────────┼──────────┼───────────────────────┼───────────────────────┼─────────┤
│ smc-01      │ TCP      │ 10.0.1.10:443         │ 203.0.113.45:54321    │ ESTAB   │
│ smc-01      │ TCP      │ 10.0.1.10:443         │ 198.51.100.22:49821   │ ESTAB   │
│ wbs-01      │ UDP      │ 10.0.1.10:10245       │ 203.0.113.45:54322    │ —       │
│ sip-01      │ UDP      │ 10.0.1.10:5060        │ 192.0.2.100:32145     │ —       │
│ ...         │          │                       │                       │         │
└─────────────┴──────────┴───────────────────────┴───────────────────────┴─────────┘

Showing 5 of 9,973 sockets. Use --all to see all.
```

---

### `rtuccli host <host-id> stats`

Системная статистика хоста.

**Синтаксис:**
```bash
rtuccli host <host-id> stats [flags]
```

**Флаги:**
- `--live` - режим live обновления # после реализации не live 

**Пример 1: Статический вывод**

```bash
$ rtuccli host node-1 stats
```

**Вывод:**

```
HOST: node-1 | System Stats

LOAD AVERAGE:
  1 min:      2.4
  5 min:      2.1
  15 min:     1.8

MEMORY:
  Total:      38.4 GB
  Used:       24.6 GB (64%)
  Free:       8.2 GB
  Cached:     5.6 GB
  Swap:       0 / 8 GB (0%)

DISK:
  /           180 GB / 400 GB (45%)    ext4
  /var/log    12 GB / 50 GB (24%)      ext4
  /data       890 GB / 2 TB (44%)      xfs

CPU:
  Cores:      16
  Usage:      42%
  User:       38%
  System:     4%
  IOWait:     0.2%
```

**Пример 2: Live режим (задача на второй этап - со зведочкой)**

```bash
$ rtuccli host node-1 stats --live
```

**Вывод:**

```
HOST: node-1 | Live Stats (refresh: 1s)                    Ctrl+C to exit

LA: 2.4 / 2.1 / 1.8    MEM: 64% [████████████░░░░░░░░]    DISK: 45%

CPU [████████░░░░░░░░░░░░] 42%    NET ↑ 245Mbps ↓ 892Mbps

SERVICES:
  sip-01   CPU [████░░░░░░░░░░░░░░░░] 18%  MEM [████░░░░░░░░░░░░░░░░] 15%
  smc-01   CPU [███████░░░░░░░░░░░░░] 35%  MEM [██████░░░░░░░░░░░░░░] 26%
  wbs-01   CPU [█████████░░░░░░░░░░░] 45%  MEM [██████░░░░░░░░░░░░░░] 26%
```

---

## Практические примеры

### Мониторинг перегруженного хоста

```bash
# Проверить общее состояние
rtuccli host node-1

# Детальная статистика в реальном времени
rtuccli host node-1 stats --live

# Проверить сетевую нагрузку
rtuccli host node-1 net
```

### Диагностика сетевых проблем

```bash
# Проверить все интерфейсы
rtuccli host node-1 net

# Детали конкретного интерфейса
rtuccli host node-1 net -i eth0

# Проверить открытые порты и соединения
rtuccli host node-1 ports
rtuccli host node-1 sockets
```

### Поиск проблем с сервисами

```bash
# Посмотреть какие сервисы запущены
rtuccli host node-1

# Проверить использование портов сервисами
rtuccli host node-1 ports --rtuccli-only

# Проверить активные соединения
rtuccli host node-1 sockets
```

## См. также

- [Service - Управление сервисами](02_Service.md)
- [Config - Управление конфигурациями](05_Config.md)
