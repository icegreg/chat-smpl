---
modified: 2025-11-25T15:17:15+03:00
created: 2025-11-25T10:59:12+03:00
---
# Service - Управление сервисами

## Обзор

Модуль управления сервисами предоставляет возможности для мониторинга и управления RTUC сервисами.

## Типы сервисов

- **sip-service** - SIP сигнализация (установка голосовых/видео звонков)
- **smc-service** - Signaling & Media Controller (WebSocket сервер для сигнализации)
- **wbs-service** - Wasina Big Sctuka (медиа-ядро для обработки потоков)
- **aux-service** - Auxiliary Service (чат, presence и вспомогательные функции)
- **api-gw-service** - Public API Gateway (публичный API шлюз)

---

## Общие команды

### `rtuccli service list`

Список всех сервисов в системе (получение инварианта).

**Синтаксис:**
```bash
rtuccli service list [flags]
```

**Флаги:**
- `--type <type>` - фильтр по типу сервиса (sip|smc|wbs|aux|api-gw)
- `--host <host-id>` - фильтр по хосту

**Пример 1: Все сервисы**

```bash
$ rtuccli service list
```

**Вывод:**

```
SERVICES (8):

┌─────────────────┬─────────┬────────────┬───────┬─────────┬─────────────┬──────────┐
│ Service ID      │ Host    │ Type       │ CPU   │ Memory  │ Connections │ Status   │
├─────────────────┼─────────┼────────────┼───────┼─────────┼─────────────┼──────────┤
│ sip-01          │ node-1  │ sip        │ 18%   │ 2.4 GB  │ 456         │ ● running│
│ sip-02          │ node-2  │ sip        │ 15%   │ 2.1 GB  │ 389         │ ● running│
│ smc-01          │ node-1  │ smc        │ 35%   │ 4.2 GB  │ 3,421       │ ● running│
│ smc-02          │ node-2  │ smc        │ 32%   │ 3.8 GB  │ 2,987       │ ● running│
│ wbs-01          │ node-1  │ wbs        │ 45%   │ 8.2 GB  │ 1,247       │ ● running│
│ wbs-02          │ node-2  │ wbs        │ 38%   │ 7.1 GB  │ 1,102       │ ● running│
│ aux-01          │ node-3  │ aux        │ 12%   │ 1.2 GB  │ 12,450      │ ● running│
│ api-gw-01       │ node-4  │ api-gw     │ 22%   │ 2.8 GB  │ 8,234       │ ● running│
└─────────────────┴─────────┴────────────┴───────┴─────────┴─────────────┴──────────┘
```

**Пример 2: Фильтр по типу**

```bash
$ rtuccli service list --type smc
```

**Вывод:**

```
SMC SERVICES (2):

┌─────────────────┬─────────┬───────┬─────────┬─────────────┬──────────┐
│ Service ID      │ Host    │ CPU   │ Memory  │ Connections │ Status   │
├─────────────────┼─────────┼───────┼─────────┼─────────────┼──────────┤
│ smc-01          │ node-1  │ 35%   │ 4.2 GB  │ 3,421       │ ● running│
│ smc-02          │ node-2  │ 32%   │ 3.8 GB  │ 2,987       │ ● running│
└─────────────────┴─────────┴───────┴─────────┴─────────────┴──────────┘
```

**Пример 3: Фильтр по хосту**

```bash
$ rtuccli service list --host node-1
```

**Вывод:**

```
SERVICES on node-1 (3):

┌─────────────────┬────────────┬───────┬─────────┬─────────────┬──────────┐
│ Service ID      │ Type       │ CPU   │ Memory  │ Connections │ Status   │
├─────────────────┼────────────┼───────┼─────────┼─────────────┼──────────┤
│ sip-01          │ sip        │ 18%   │ 2.4 GB  │ 456         │ ● running│
│ smc-01          │ smc        │ 35%   │ 4.2 GB  │ 3,421       │ ● running│
│ wbs-01          │ wbs        │ 45%   │ 8.2 GB  │ 1,247       │ ● running│
└─────────────────┴────────────┴───────┴─────────┴─────────────┴──────────┘
```

---

### `rtuccli service <service-id> status`

Статус и health check сервиса.

**Синтаксис:**
```bash
rtuccli service <service-id> status [flags]
```

**Пример:**

```bash
$ rtuccli service smc-01 status
```

**Вывод:**

```
SERVICE: smc-01

Status:       ● running
Health:       healthy
Last Check:   2s ago

CHECKS:
  ✓ Process running (PID 12345)
  ✓ Port 443 listening
  ✓ Memory under threshold
  ✓ CPU under threshold
  ✓ API responding (8ms)
  ✓ WebSocket accepting connections
```

---

## SIP Service

### `rtuccli service <sip-service-id>`

Детальная информация о SIP сервисе.

**Пример:**

```bash
$ rtuccli service sip-01
```

**Вывод:**

```
SERVICE: sip-01
Type:         sip (SIP Signaling)
Host:         node-1 (10.0.1.10)
Status:       ● running
PID:          12340
Uptime:       14d 6h 23m

RESOURCES:
  CPU:        18%
  Memory:     2.4 GB / 8 GB (30%)
  Threads:    48
  FDs:        892 / 65535

SIP CONNECTIONS:
  Total:      456
  Registered: 412
  In-Call:    298
  Idle:       114

CONFERENCES:
  Active:     23
  Total Legs: 298

PORTS:
  5060/UDP    (SIP)
  5061/TCP    (SIP-TLS)
  
CONFIGURATION:
  Config ID:  sip01-config
  Applied:    2024-01-14 15:35:00
```

---

## SMC Service (WebSocket Server)

### `rtuccli service <smc-service-id>`

Детальная информация о SMC сервисе.

**Пример:**

```bash
$ rtuccli service smc-01
```

**Вывод:**

```
SERVICE: smc-01
Type:         smc (Signaling & Media Controller)
Host:         node-1 (10.0.1.10)
Status:       ● running
PID:          12345
Uptime:       14d 6h 23m

RESOURCES:
  CPU:        35%
  Memory:     4.2 GB / 16 GB (26%)
  Threads:    124
  FDs:        4,521 / 65535

WEBSOCKET CONNECTIONS:
  Total:      3,421
  Active:     3,234
  Idle:       145
  Handshaking: 42

SIGNALING STATS:
  Messages/sec:   2,847
  Peak (24h):     12,450 msg/s
  Avg Latency:    8.2ms
  
CLIENT SESSIONS:
  Desktop:    1,245
  Mobile:     1,456
  Web:        720

CONFERENCES:
  Active:     23
  Participants: 412

PORTS:
  443/TCP     (WSS - signaling)
  8443/TCP    (WSS - backup)
  
CONFIGURATION:
  Config ID:  smc01-config
  Applied:    2024-01-15 10:30:00
```

---

### `rtuccli service <smc-service-id> status`

Детальная информация о WebSocket соединениях.

**Пример:**

```bash
$ rtuccli service smc-01 websockets
```

**Вывод:**

```
SERVICE: smc-01 | WebSocket Connections

SUMMARY:
  Total:          3,421
  Active:         3,234 (94%)
  Idle:           145 (4%)
  Handshaking:    42 (1%)

BY DEVICE TYPE:
  Desktop:        1,245 (36%)
    ├─ Windows    687
    ├─ macOS      423
    └─ Linux      135
  Mobile:         1,456 (43%)
    ├─ Android    892
    └─ iOS        564
  Web:            720 (21%)
    ├─ Chrome     456
    ├─ Firefox    189
    └─ Safari     75

```

---

## WBS Service (Media Core)

### `rtuccli service <wbs-service-id>`

Детальная информация о WBS сервисе.

**Пример:**

```bash
$ rtuccli service wbs-01
```

**Вывод:**

```
SERVICE: wbs-01
Type:         wbs (WebRTC Bridge Service / Media Core)
Host:         node-1 (10.0.1.10)
Status:       ● running
PID:          12350
Uptime:       14d 6h 23m

RESOURCES:
  CPU:        45%
  Memory:     8.2 GB / 32 GB (26%)
  Threads:    256
  FDs:        6,234 / 65535

MEDIA STREAMS:
  Total:      1,247 peers
  Video ↑:    1,102 streams (248.2 Mbps)
  Video ↓:    4,521 streams (892.4 Mbps)
  Audio ↑:    1,089 streams (52.3 Mbps)
  Audio ↓:    4,356 streams (139.4 Mbps)

MEDIA MODES:
  SFU:        18 conferences (312 participants)
  MCU:        3 conferences (78 participants)
  Hybrid:     2 conferences (22 participants)

CODECS IN USE:
  Video:      VP9 (65%), VP8 (20%), H264 (15%)
  Audio:      Opus (98%), G.722 (2%)

ICE SERVERS:
  STUN:       stun.example.com:3478
  TURN:       turn.example.com:3478 (892 relays active)

PORTS:
  10000-20000/UDP (RTP/RTCP media)
  3478/UDP        (STUN)
  5349/TCP        (TURN-TLS)
  
CONFIGURATION:
  Config ID:  wbs01-config
  Applied:    2024-01-14 15:35:00
```

---

### `rtuccli service <wbs-service-id> status`

Детальная медиа статистика.

**Пример:**

```bash
$ rtuccli service wbs-01 status
```

**Вывод:**

```
SERVICE: wbs-01 | Media Statistics

CONFERENCES BY MODE:
┌─────────┬───────────────┬──────────────┬────────────┬─────────────┐
│ Mode    │ Conferences   │ Participants │ Bandwidth  │ Avg Quality │
├─────────┼───────────────┼──────────────┼────────────┼─────────────┤
│ SFU     │ 18            │ 312          │ 892 Mbps   │ ★★★★        │
│ MCU     │ 3             │ 78           │ 156 Mbps   │ ★★★☆        │
│ Hybrid  │ 2             │ 22           │ 92 Mbps    │ ★★★☆        │
└─────────┴───────────────┴──────────────┴────────────┴─────────────┘

VIDEO STREAMS:
  Send:     1,102 streams
    ├─ VP9:     717 (65%)
    ├─ VP8:     221 (20%)
    └─ H264:    164 (15%)
  Receive:  4,521 streams
  Bitrate ↑: 248.2 Mbps
  Bitrate ↓: 892.4 Mbps

AUDIO STREAMS:
  Send:     1,089 streams
    ├─ Opus:    1,067 (98%)
    └─ G.722:   22 (2%)
  Receive:  4,356 streams
  Bitrate ↑: 52.3 Mbps
  Bitrate ↓: 139.4 Mbps

QUALITY METRICS:
  Avg RTT:      67ms
  Avg Jitter:   23ms
  Avg Loss:     1.2%
  
ICE STATISTICS:
  Total Candidates:     8,234
  STUN Success:         7,892 (96%)
  TURN Relay Active:    892 (11%)
  Failed:               342 (4%)
```

---

## AUX Service (Chat & Auxiliary)

### `rtuccli service <aux-service-id>`

Детальная информация об AUX сервисе.

**Пример:**

```bash
$ rtuccli service aux-01
```

**Вывод:**

```
SERVICE: aux-01
Type:         aux (Auxiliary Service - Chat, Presence)
Host:         node-3 (10.0.1.12)
Status:       ● running
PID:          12360
Uptime:       14d 6h 23m

RESOURCES:
  CPU:        12%
  Memory:     1.2 GB / 4 GB (30%)
  Threads:    64
  FDs:        12,892 / 65535

CHAT:
  Active Channels:     847
  Subscribers:         12,450 (avg 14.7 per channel)
  P2P Channels:        89
  Messages/sec:        1,247
  Peak (24h):          8,920 msg/s
  Avg Latency:         4.2ms

WEBSOCKET CONNECTIONS:
  Active:              12,450
  Idle:                342
  Reconnecting:        12

PRESENCE:
  Online Users:        12,450
  Away:                2,341
  Busy (in-call):      3,421

FEATURES:
  Chat:                ✓ enabled
  File Transfer:       ✓ enabled (max 100MB)

PORTS:
  8080/TCP    (WS - chat)
  8443/TCP    (WSS - chat secure)
  
CONFIGURATION:
  Config ID:  aux01-config
  Applied:    2024-01-15 10:35:00
```

---

### `rtuccli service <aux-service-id> stats`

Детальная статистика чата.

**Пример:**

```bash
$ rtuccli service aux-01 stats
```

**Вывод:**

```
SERVICE: aux-01 | Chat Statistics

CHANNELS:
  Total Active:     847
  Public:           758
  Private:          89
  P2P:              89
  Subscribers:      12,450 (avg 14.7 per channel)

THROUGHPUT:
  Messages/sec:     1,247
  Peak (24h):       8,920 msg/s
  Peak (1h):        4,234 msg/s
  Avg Latency:      4.2ms

MESSAGE TYPES:
  Text:             78%
  File Share:       12%
  Reactions:        8%
  System:           2%

WEBSOCKET CONNECTIONS:
  Active:           12,450
  Idle:             342
  Reconnecting:     12
  Avg Latency:      32ms

TOP WS CHANNELS (by activity):
┌───────────────────┬─────────────┬──────────┬───────────┬──────────┐
│ Channel           │ Conference  │ Subs     │ Messages  │ Msg/s    │
├───────────────────┼─────────────┼──────────┼───────────┼──────────┤
│ conf-456:main     │ conf-456    │ 24       │ 756       │ 2.3      │
│ conf-789:main     │ conf-789    │ 156      │ 4,521     │ 45.2     │
│ conf-123:q&a      │ conf-123    │ 89       │ 1,234     │ 12.1     │
│ conf-234:private  │ conf-234    │ 3        │ 45        │ 0.5      │
└───────────────────┴─────────────┴──────────┴───────────┴──────────┘

STORAGE:
  Backend:          Redis
  Total Messages:   1,247,892 (7d retention)
  Storage Size:     8.4 GB
```

---

## API Gateway Service

### `rtuccli service <api-gw-service-id>`

Детальная информация об API Gateway сервисе.

**Пример:**

```bash
$ rtuccli service api-gw-01
```

**Вывод:**

```
SERVICE: api-gw-01
Type:         api-gw (Public API Gateway)
Host:         node-4 (10.0.1.13)
Status:       ● running
PID:          12370
Uptime:       14d 6h 23m

RESOURCES:
  CPU:        22%
  Memory:     2.8 GB / 8 GB (35%)
  Threads:    96
  FDs:        8,456 / 65535

API TRAFFIC:
  Requests/sec:       2,456
  Peak (24h):         15,234 req/s
  Avg Response Time:  45ms
  Error Rate:         0.02%

ENDPOINTS:
  /api/v1/conferences     1,245 req/s
  /api/v1/auth            456 req/s
  /api/v1/users           342 req/s
  /api/v1/recordings      123 req/s
  /api/v1/stats           89 req/s

CONNECTIONS:
  Active:     8,234
  TLS:        8,234 (100%)
  HTTP/2:     7,892 (96%)
  HTTP/1.1:   342 (4%)

AUTHENTICATION:
  JWT:        7,234 sessions
  OAuth2:     892 sessions
  API Keys:   108 sessions

RATE LIMITING:
  Enforced:   ✓ enabled
  Limits:     1000 req/min per IP
  Blocked:    23 IPs (last hour)

PORTS:
  443/TCP     (HTTPS)
  8443/TCP    (Management API)
  
CONFIGURATION:
  Config ID:  api-gw01-config
  Applied:    2024-01-13 14:20:00
```

---

### `rtuccli service <api-gw-service-id> stats`

Детальная статистика API Gateway.

**Пример:**

```bash
$ rtuccli service api-gw-01 stats
```

**Вывод:**

```
SERVICE: api-gw-01 | API Statistics

REQUEST METRICS:
  Current:          2,456 req/s
  Peak (24h):       15,234 req/s
  Peak (1h):        8,920 req/s
  Avg Response:     45ms
  95th %ile:        120ms
  99th %ile:        340ms

STATUS CODES (last hour):
  2xx Success:      98.42%
  3xx Redirect:     0.89%
  4xx Client Err:   0.67%
  5xx Server Err:   0.02%

TOP ENDPOINTS (by req/s):
┌──────────────────────────┬─────────┬─────────────┬────────────┐
│ Endpoint                 │ Req/s   │ Avg Latency │ Error Rate │
├──────────────────────────┼─────────┼─────────────┼────────────┤
│ /api/v1/conferences      │ 1,245   │ 34ms        │ 0.01%      │
│ /api/v1/auth             │ 456     │ 89ms        │ 0.05%      │
│ /api/v1/users            │ 342     │ 23ms        │ 0.02%      │
│ /api/v1/recordings       │ 123     │ 156ms       │ 0.03%      │
│ /api/v1/stats            │ 89      │ 12ms        │ 0.00%      │
└──────────────────────────┴─────────┴─────────────┴────────────┘

AUTHENTICATION:
  JWT Sessions:     7,234
  OAuth2 Sessions:  892
  API Key Sessions: 108
  Failed Auth:      23 (last hour)

RATE LIMITING:
  Blocked IPs:      23
  Throttled:        145
  Total Blocks:     892 (24h)
```

---

## См. также

- [Host - Управление хостами](01_Host.md)
- [Conference - Управление мероприятиями](03_Conference.md)
- [Config - Управление конфигурациями](05_Config.md)
