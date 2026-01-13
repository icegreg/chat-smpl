---
modified: 2025-11-25T15:18:53+03:00
created: 2025-11-25T10:59:12+03:00
---
# Conference - Управление мероприятиями

## Обзор

Модуль управления мероприятиями (конференциями) предоставляет возможности для мониторинга активных видеоконференций, участников, качества соединений и чата.

## Режимы мероприятий

- **SFU** (Selective Forwarding Unit) - каждый участник получает индивидуальные потоки
- **MCU** (Multipoint Control Unit) - сервер микширует потоки и отправляет единый поток
- **Hybrid** - комбинированный режим (SFU + MCU)

---

## Команды

### `rtuccli conf list`

Список всех мероприятий (по умолчанию - активные мероприятия).

**Синтаксис:**
```bash
rtuccli conf list [flags]
```

**Флаги:**
- `--all` - все конференции
- `--date` - все конференции с датой старта {date}
- `--mode <mode>` - фильтр по режиму (sfu|mcu|hybrid)

**Пример 1: Все мероприятия**

```bash
$ rtuccli conf list
```

**Вывод:**

```
CONFERENCES (47):

┌───────────┬──────────────────────────┬────────┬─────────┬─────────┬──────────┐
│ Conf ID   │ Name                     │ Mode   │ Clients │ Quality │ Duration │
├───────────┼──────────────────────────┼────────┼─────────┼─────────┼──────────┤
│ conf-456  │ Q1 Planning Meeting      │ SFU    │ 24      │ ★★★☆    │ 2h 34m   │
│ conf-789  │ All-Hands Broadcast      │ MCU    │ 156     │ ★★★★    │ 1h 12m   │
│ conf-123  │ Engineering Sync         │ SFU    │ 8       │ ★★★★    │ 0h 45m   │
│ conf-234  │ Customer Demo            │ Hybrid │ 12      │ ★★★☆    │ 0h 23m   │
│ conf-567  │ 1:1 Alice/Bob            │ P2P    │ 2       │ ★★★★    │ 0h 15m   │
│ ...       │                          │        │         │         │          │
└───────────┴──────────────────────────┴────────┴─────────┴─────────┴──────────┘

Showing 5 of 47. Use --all to see all.
```

**Пример 2: Только активные SFU**

```bash
$ rtuccli conf list --active --mode sfu
```

**Вывод:**

```
CONFERENCES (active, SFU) (23):

┌───────────┬──────────────────────────┬─────────┬─────────┬──────────┐
│ Conf ID   │ Name                     │ Clients │ Quality │ Duration │
├───────────┼──────────────────────────┼─────────┼─────────┼──────────┤
│ conf-456  │ Q1 Planning Meeting      │ 24      │ ★★★☆    │ 2h 34m   │
│ conf-123  │ Engineering Sync         │ 8       │ ★★★★    │ 0h 45m   │
│ ...       │                          │         │         │          │
└───────────┴──────────────────────────┴─────────┴─────────┴──────────┘
```

---

### `rtuccli conf <conf-id>`

Детальная информация о мероприятии.

**Синтаксис:**
```bash
rtuccli conf <conf-id> [flags]
```

**Пример:**

```bash
$ rtuccli conf conf-456
```

**Вывод:**

```
CONFERENCE: conf-456
Name:         Q1 Planning Meeting
Mode:         SFU (with MCU fallback)
Host:         node-2 (smc-02, wbs-02)
Started:      2024-01-15 14:00:00 UTC (2h 34m ago)
Recording:    ● enabled (1.2 GB)

PARTICIPANTS:
  Total:      24
  Active:     22
  Presenting: 2

QUALITY DISTRIBUTION:
  ████████████████░░░░  Excellent: 16 (67%)
  ███░░░░░░░░░░░░░░░░░  Good:       4 (17%)
  ██░░░░░░░░░░░░░░░░░░  Normal:     3 (12%)
  ░░░░░░░░░░░░░░░░░░░░  Poor:       1 (4%)

MEDIA STREAMS:
  Video ↑:    24 streams (48.2 Mbps total)
  Video ↓:    312 streams (187.4 Mbps total)
  Audio ↑:    22 streams (1.4 Mbps total)
  Audio ↓:    484 streams (3.1 Mbps total)

WS:
  Messages:   847 total | 2.3 msg/s current

Use 'rtuccli conf conf-456 clients' to see participants
```

---

### `rtuccli conf <conf-id> clients`

Список клиентов (участников) мероприятия.

**Синтаксис:**
```bash
rtuccli conf <conf-id> clients [flags]
```

**Флаги:**
- `--quality <level>` - фильтр по качеству (poor|normal|good|excellent)
- `--device <type>` - фильтр по типу устройства

**Пример 1: Все клиенты**

```bash
$ rtuccli conf conf-456 clients
```

**Вывод:**

```
CONFERENCE: conf-456 | Clients (24)

┌──────────────────────┬─────────────────┬───────┬─────────┬──────────┬─────────────┐
│ Client               │ Device          │ Peers │ Quality │ Duration │ Status      │
├──────────────────────┼─────────────────┼───────┼─────────┼──────────┼─────────────┤
│ alice@company.com    │ Desktop Windows │ 2     │ ★★★★    │ 2h 30m   │ ● active    │
│ bob@company.com      │ Android 14      │ 1     │ ★★★☆    │ 1h 45m   │ ● active    │
│ carol@company.com    │ iOS 17          │ 1     │ ★★★★    │ 2h 10m   │ ● active    │
│ dave@external.org    │ Web Chrome      │ 1     │ ★★☆☆    │ 0h 15m   │ ● active    │
│ eve@company.com      │ Desktop macOS   │ 3     │ ★★★★    │ 2h 34m   │ ◐ presenting│
│ frank@company.com    │ Desktop Linux   │ 1     │ ★☆☆☆    │ 0h 05m   │ ● active    │
│ ...                  │                 │       │         │          │             │
└──────────────────────┴─────────────────┴───────┴─────────┴──────────┴─────────────┘

Quality: ★★★★ Excellent | ★★★☆ Good | ★★☆☆ Normal | ★☆☆☆ Poor

Use 'rtuccli conf conf-456 client <client-id>' for details
```

**Пример 2: Клиенты с плохим качеством**

```bash
$ rtuccli conf conf-456 clients --quality poor
```

**Вывод:**

```
CONFERENCE: conf-456 | Clients with Poor Quality (1)

┌──────────────────────┬─────────────────┬───────┬─────────┬──────────┬────────────┐
│ Client               │ Device          │ Peers │ Quality │ Duration │ Issues     │
├──────────────────────┼─────────────────┼───────┼─────────┼──────────┼────────────┤
│ frank@company.com    │ Desktop Linux   │ 1     │ ★☆☆☆    │ 0h 05m   │ jitter,loss│
└──────────────────────┴─────────────────┴───────┴─────────┴──────────┴────────────┘
```

---

### `rtuccli conf <conf-id> client <client-id>`

Детальная информация о клиенте в мероприятии.

**Синтаксис:**
```bash
rtuccli conf <conf-id> client <client-id> [flags]
```

**Пример:**

```bash
$ rtuccli conf conf-456 client frank@company.com
```

**Вывод:**

```
CLIENT: frank@company.com
Conference:   conf-456 "Q1 Planning Meeting"
Device:       Desktop Linux (Firefox 121)
User Agent:   Mozilla/5.0 (X11; Linux x86_64; rv:121.0)
IP:           203.0.113.45
Joined:       2024-01-15 16:29:00 UTC (5m ago)
Duration:     5m 23s
Quality:      ★☆☆☆ Poor

PEERS:
┌──────────┬────────────────┬─────────────────────────────┬─────────┐
│ Peer ID  │ Role           │ Streams                     │ Quality │
├──────────┼────────────────┼─────────────────────────────┼─────────┤
│ peer-f1  │ main           │ video↑↓×6, audio↔           │ ★☆☆☆    │
└──────────┴────────────────┴─────────────────────────────┴─────────┘

ISSUES:
  ⚠ High packet loss on video↓ (8.2%)
  ⚠ Jitter above threshold (85ms)
  ⚠ ICE reconnects: 3 in last 5m

CHAT:
  Channel:    main
  Messages:   12 sent | 847 received
  WS Latency: 4.2ms

RECENT EVENTS:
  16:33:05  BITRATE_ADJUSTED video_recv 1.1Mbps→0.6Mbps
  16:33:00  PACKET_LOSS threshold_exceeded
  16:32:11  ICE_CONNECTED candidate_pair=relay/relay
  16:32:10  ICE_RESTART trigger=connectivity_check_failed

Use 'rtuccli conf conf-456 peer peer-f1' for full peer details
```

---

### `rtuccli conf <conf-id> peer <peer-id>`

Детальная информация о пире (peer connection).

**Синтаксис:**
```bash
rtuccli conf <conf-id> peer <peer-id> [flags]
```

**Пример:**

```bash
$ rtuccli conf conf-456 peer peer-f1
```

**Вывод:**

```
PEER: peer-f1
Conference:   conf-456 "Q1 Planning Meeting"
Client:       frank@company.com
Device:       Desktop Linux (Firefox 121)
Role:         main (camera + microphone)
Created:      2024-01-15 16:29:02 UTC (5m ago)
Quality:      ★☆☆☆ Poor

ICE:
  Local:      srflx 203.0.113.45:54321
  Remote:     relay 10.0.1.11:16789
  RTT:        89ms

DTLS:
  State:      connected
  Cipher:     TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256

CHANNELS:
┌─────────┬───────────┬────────┬─────────┬────────┬─────────┬────────────┐
│ Type    │ Direction │ Codec  │ Bitrate │ Jitter │ RTT     │ Loss       │
├─────────┼───────────┼────────┼─────────┼────────┼─────────┼────────────┤
│ WebRTC  │ ↑ send    │ VP9    │ 1.2Mbps │ 12ms   │ 89ms    │ 0.4%       │
│ WebRTC  │ ↓ recv    │ VP9    │ 0.8Mbps │ 85ms   │ 89ms    │ 8.2% ⚠     │
│ WebRTC  │ ↓ recv    │ VP9    │ 0.9Mbps │ 78ms   │ 89ms    │ 7.1% ⚠     │
│ WebRTC  │ ↓ recv    │ VP9    │ 1.1Mbps │ 45ms   │ 89ms    │ 2.3%       │
│ WebRTC  │ ↓ recv    │ VP9    │ 0.7Mbps │ 92ms   │ 89ms    │ 9.0% ⚠     │
│ WebRTC  │ ↓ recv    │ VP9    │ 1.0Mbps │ 38ms   │ 89ms    │ 1.8%       │
│ WebRTC  │ ↓ recv    │ VP9    │ 0.6Mbps │ 67ms   │ 89ms    │ 4.5%       │
│ RTP     │ ↑ send    │ Opus   │ 48kbps  │ 8ms    │ 85ms    │ 0.1%       │
│ RTP     │ ↓ recv    │ Opus   │ 32kbps  │ 15ms   │ 85ms    │ 0.8%       │
│ WS      │ ↔ bidi    │ —      │ 0.8kbps │ —      │ 42ms    │ —          │
└─────────┴───────────┴────────┴─────────┴────────┴─────────┴────────────┘

Direction: ↑ send | ↓ recv | ↔ bidi
```

---

### `rtuccli conf <conf-id> peer <peer-id> channels`

Детальная информация о каналах данных пира.

**Синтаксис:**
```bash
rtuccli conf <conf-id> peer <peer-id> channels [flags]
```

**Пример:**

```bash
$ rtuccli conf conf-456 peer peer-f1 channels
```

**Вывод:**

```
PEER: peer-f1 | Channels

VIDEO SEND (1):
┌────────────┬────────┬─────────────┬─────────┬────────┬────────┬───────┐
│ Stream ID  │ Codec  │ Resolution  │ Bitrate │ FPS    │ Jitter │ Loss  │
├────────────┼────────┼─────────────┼─────────┼────────┼────────┼───────┤
│ v-send-01  │ VP9    │ 1280×720    │ 1.2Mbps │ 30     │ 12ms   │ 0.4%  │
└────────────┴────────┴─────────────┴─────────┴────────┴────────┴───────┘

VIDEO RECV (6):
┌────────────┬────────┬─────────────┬─────────┬────────┬────────┬───────┬──────────────┐
│ Stream ID  │ Codec  │ Resolution  │ Bitrate │ FPS    │ Jitter │ Loss  │ Source       │
├────────────┼────────┼─────────────┼─────────┼────────┼────────┼───────┼──────────────┤
│ v-recv-01  │ VP9    │ 1280×720    │ 0.8Mbps │ 28     │ 85ms   │ 8.2%⚠ │ alice        │
│ v-recv-02  │ VP9    │ 1280×720    │ 0.9Mbps │ 29     │ 78ms   │ 7.1%⚠ │ bob          │
│ v-recv-03  │ VP9    │ 1920×1080   │ 1.1Mbps │ 30     │ 45ms   │ 2.3%  │ eve (screen) │
│ v-recv-04  │ VP9    │ 640×480     │ 0.7Mbps │ 25     │ 92ms   │ 9.0%⚠ │ eve (camera) │
│ v-recv-05  │ VP9    │ 1280×720    │ 1.0Mbps │ 30     │ 38ms   │ 1.8%  │ carol        │
│ v-recv-06  │ VP9    │ 1280×720    │ 0.6Mbps │ 24     │ 67ms   │ 4.5%  │ dave         │
└────────────┴────────┴─────────────┴─────────┴────────┴────────┴───────┴──────────────┘

AUDIO SEND (1):
┌────────────┬────────┬─────────┬────────┬────────┐
│ Stream ID  │ Codec  │ Bitrate │ Jitter │ Loss   │
├────────────┼────────┼─────────┼────────┼────────┤
│ a-send-01  │ Opus   │ 48kbps  │ 8ms    │ 0.1%   │
└────────────┴────────┴─────────┴────────┴────────┘

AUDIO RECV (1 mixed):
┌────────────┬────────┬─────────┬────────┬────────┬───────────┐
│ Stream ID  │ Codec  │ Bitrate │ Jitter │ Loss   │ Sources   │
├────────────┼────────┼─────────┼────────┼────────┼───────────┤
│ a-recv-01  │ Opus   │ 32kbps  │ 15ms   │ 0.8%   │ 5 mixed   │
└────────────┴────────┴─────────┴────────┴────────┴───────────┘

DATA CHANNELS:
┌────────────┬──────────┬───────────┬─────────┬─────────┐
│ Channel    │ Protocol │ Direction │ Bitrate │ Latency │
├────────────┼──────────┼───────────┼─────────┼─────────┤
│ chat       │ WS       │ ↔ bidi    │ 0.8kbps │ 42ms    │
└────────────┴──────────┴───────────┴─────────┴─────────┘
```

---

### `rtuccli conf <conf-id> peer <peer-id> log`

Лог событий пира.

**Синтаксис:**
```bash
rtuccli conf <conf-id> peer <peer-id> log [flags]
```

**Флаги:**
- `--follow` - следить за новыми событиями в реальном времени
- `--all` - показать все события (без ограничений)

**Пример:**

```bash
$ rtuccli conf conf-456 peer peer-f1 log
```

**Вывод:**

```
PEER LOG: peer-f1
Client:     frank@company.com
Conference: conf-456

┌─────────────────────┬───────────────────────────────────────────────────┐
│ Timestamp           │ Event                                             │
├─────────────────────┼───────────────────────────────────────────────────┤
│ 16:29:02.100        │ PEER_CREATED role=main                            │
│ 16:29:02.150        │ ICE_GATHERING state=new                           │
│ 16:29:02.200        │ ICE_CANDIDATE type=host addr=192.168.1.100:54320  │
│ 16:29:02.890        │ ICE_CANDIDATE type=srflx addr=203.0.113.45:54321  │
│ 16:29:03.100        │ ICE_CANDIDATE type=relay addr=10.0.1.11:16789     │
│ 16:29:03.200        │ ICE_CONNECTED candidate_pair=srflx/relay          │
│ 16:29:03.350        │ DTLS_CONNECTING                                   │
│ 16:29:03.450        │ DTLS_CONNECTED                                    │
│ 16:29:03.500        │ TRACK_ADDED kind=video codec=VP9 dir=send         │
│ 16:29:03.520        │ TRACK_ADDED kind=audio codec=Opus dir=send        │
│ 16:29:04.100        │ TRACK_ADDED kind=video codec=VP9 dir=recv src=eve │
│ 16:29:04.150        │ TRACK_ADDED kind=video codec=VP9 dir=recv src=alice│
│ 16:29:04.200        │ TRACK_ADDED kind=video codec=VP9 dir=recv src=bob │
│ ...                                                                     │
│ 16:31:45.000        │ QUALITY_DEGRADED reason=high_jitter value=92ms    │
│ 16:32:10.000        │ ICE_RESTART trigger=connectivity_check_failed     │
│ 16:32:10.500        │ ICE_GATHERING state=gathering                     │
│ 16:32:11.200        │ ICE_CONNECTED candidate_pair=relay/relay          │
│ 16:33:00.000        │ PACKET_LOSS threshold_exceeded stream=v-recv-01   │
│ 16:33:00.100        │ PACKET_LOSS threshold_exceeded stream=v-recv-02   │
│ 16:33:00.200        │ PACKET_LOSS threshold_exceeded stream=v-recv-04   │
│ 16:33:05.000        │ BITRATE_ADJUSTED v-recv-01 1.1Mbps→0.8Mbps        │
│ 16:33:05.100        │ BITRATE_ADJUSTED v-recv-04 1.0Mbps→0.7Mbps        │
│ 16:33:10.000        │ QUALITY_SCORE_CHANGED ★★☆☆→★☆☆☆                   │
└─────────────────────┴───────────────────────────────────────────────────┘

Showing last 50 events. Use --all for complete log.
Use --follow to stream new events.
```

---

### `rtuccli conf <conf-id> connections`

Все соединения мероприятия.

**Синтаксис:**
```bash
rtuccli conf <conf-id> connections [flags]
```

**Флаги:**
- `--type <type>` - фильтр по типу (webrtc|rtp|ws)

**Пример 1: Все соединения**

```bash
$ rtuccli conf conf-456 connections
```

**Вывод:**

```
CONFERENCE: conf-456 | Connections

SUMMARY:
  WebRTC:     312 streams (24 clients)
  RTP:        48 streams
  WebSocket:  24 connections

WEBRTC STREAMS:
┌──────────────────┬───────────┬────────┬─────────┬────────┐
│ Client           │ Direction │ Type   │ Bitrate │ Quality│
├──────────────────┼───────────┼────────┼─────────┼────────┤
│ alice            │ ↑ send    │ video  │ 2.4Mbps │ ★★★★   │
│ alice            │ ↑ send    │ audio  │ 48kbps  │ ★★★★   │
│ alice            │ ↓ recv    │ video  │ 8.2Mbps │ ★★★★   │
│ bob              │ ↑ send    │ video  │ 1.8Mbps │ ★★★☆   │
│ ...              │           │        │         │        │
└──────────────────┴───────────┴────────┴─────────┴────────┘

Showing 5 of 312. Use --all to see all.
```

**Пример 2: Только WebSocket**

```bash
$ rtuccli conf conf-456 connections --type ws
```

**Вывод:**

```
CONFERENCE: conf-456 | WebSocket Connections (24)

┌──────────────────────┬─────────────────┬─────────┬──────────┬──────────┐
│ Client               │ Device          │ Latency │ Messages │ Status   │
├──────────────────────┼─────────────────┼─────────┼──────────┼──────────┤
│ alice@company.com    │ Desktop Windows │ 12ms    │ 1,245    │ ● active │
│ bob@company.com      │ Android 14      │ 45ms    │ 892      │ ● active │
│ carol@company.com    │ iOS 17          │ 38ms    │ 756      │ ● active │
│ dave@external.org    │ Web Chrome      │ 89ms    │ 124      │ ● active │
│ eve@company.com      │ Desktop macOS   │ 8ms     │ 2,341    │ ● active │
│ frank@company.com    │ Desktop Linux   │ 42ms    │ 12       │ ● active │
│ ...                  │                 │         │          │          │
└──────────────────────┴─────────────────┴─────────┴──────────┴──────────┘
```

---

### `rtuccli conf <conf-id> chat`

Статистика чата мероприятия.

**Синтаксис:**
```bash
rtuccli conf <conf-id> chat [flags]
```

**Пример:**

```bash
$ rtuccli conf conf-456 chat
```

**Вывод:**

```
CONFERENCE: conf-456 | Chat

CHANNELS:
┌─────────────────┬─────────────┬──────────┬───────────┬──────────┐
│ Channel         │ Type        │ Members  │ Messages  │ Msg/s    │
├─────────────────┼─────────────┼──────────┼───────────┼──────────┤
│ main            │ public      │ 24       │ 756       │ 2.1      │
│ q&a             │ public      │ 24       │ 89        │ 0.2      │
│ private-hosts   │ private     │ 3        │ 12        │ 0.0      │
└─────────────────┴─────────────┴──────────┴───────────┴──────────┘

TOTALS:
  Messages:       847
  Rate:           2.3 msg/s
  Peak (session): 12.4 msg/s

WEBSOCKET STATS:
  Connections:    24
  Avg Latency:    32ms
  Reconnects:     2
```

---

## Практические примеры

### Мониторинг активных мероприятий

```bash
# Все активные мероприятия
rtuccli conf list --active

# Мероприятия с проблемами качества
rtuccli conf list --active | grep "★☆☆☆\|★★☆☆"

# Live мониторинг мероприятия
rtuccli conf conf-456 --watch
```

### Поиск проблемных клиентов

```bash
# Клиенты с плохим качеством
rtuccli conf conf-456 clients --quality poor

# Все клиенты с детальной информацией
rtuccli conf conf-456 clients -o wide

# Экспорт для анализа
rtuccli conf conf-456 clients -o json > clients.json
```

### Диагностика конкретного клиента

```bash
# Общая информация
rtuccli conf conf-456 client frank@company.com

# Детали пира
rtuccli conf conf-456 peer peer-f1

# Детальная информация о каналах
rtuccli conf conf-456 peer peer-f1 channels

# Лог событий
rtuccli conf conf-456 peer peer-f1 log
```

### Анализ соединений

```bash
# Все соединения мероприятия
rtuccli conf conf-456 connections

# Только WebRTC потоки
rtuccli conf conf-456 connections --type webrtc

# Только WebSocket соединения
rtuccli conf conf-456 connections --type ws
```

### Мониторинг чата

```bash
# Статистика чата
rtuccli conf conf-456 chat

# Экспорт в JSON
rtuccli conf conf-456 chat -o json
```

## JSON вывод для автоматизации

```bash
# Экспорт списка мероприятий
rtuccli conf list --active -o json

# Поиск мероприятий с плохим качеством
rtuccli conf list -o json | jq '.[] | select(.quality=="poor")'

# Клиенты с проблемами
rtuccli conf conf-456 clients -o json | jq '.[] | select(.quality_score < 2)'

# Детали пира в JSON
rtuccli conf conf-456 peer peer-f1 -o json
```

## Автоматизация мониторинга

### Скрипт проверки качества

```bash
#!/bin/bash
# check_conference_quality.sh

for conf in $(rtuccli conf list --active -o json | jq -r '.[].conf_id'); do
  poor=$(rtuccli conf $conf clients --quality poor -o json | jq 'length')
  
  if [ $poor -gt 0 ]; then
    echo "Alert: Conference $conf has $poor clients with poor quality"
    rtuccli conf $conf clients --quality poor
  fi
done
```

### Экспорт метрик

```bash
#!/bin/bash
# export_metrics.sh

rtuccli conf list --active -o json | \
  jq -r '.[] | "\(.conf_id),\(.clients),\(.quality),\(.duration)"' | \
  while IFS=, read conf clients quality duration; do
    echo "conference_clients{conf=\"$conf\"} $clients"
    echo "conference_duration{conf=\"$conf\"} $duration"
  done
```

## См. также

- [Client - Управление клиентами](04_Client.md)
- [Service - Управление сервисами](02_Service.md)
- [README - Главная страница](README.md)
