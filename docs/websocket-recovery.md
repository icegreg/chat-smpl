# WebSocket Recovery - Восстановление сообщений после потери связи

Документация описывает механизм автоматического восстановления пропущенных сообщений при временной потере интернет-соединения.

## Содержание

- [Архитектура](#архитектура)
- [Фаза 1: Нормальная работа](#фаза-1-нормальная-работа)
- [Фаза 2: Потеря соединения](#фаза-2-потеря-соединения)
- [Фаза 3: Centrifugo Recovery](#фаза-3-centrifugo-recovery)
- [Фаза 4: API Fallback](#фаза-4-api-fallback)
- [Состояния соединения](#состояния-соединения)
- [Хранение seq_num](#хранение-seq_num)
- [Конфигурация](#конфигурация)
- [Временная диаграмма](#временная-диаграмма)
- [Multi-Device сценарий](#multi-device-сценарий)
- [Тестирование](#тестирование)

---

## Архитектура

```mermaid
flowchart TB
    subgraph Client["Клиент (Vue)"]
        CF[Centrifuge Client]
        CS[chat.ts Store]
        NS[network.ts Store]
        LS[(localStorage<br/>seq_nums)]

        CF <--> CS
        CS <--> NS
        CS <--> LS
    end

    subgraph Infrastructure["Инфраструктура"]
        NG[NGINX]

        subgraph Centrifugo["Centrifugo Cluster"]
            C1[Centrifugo 1]
            C2[Centrifugo 2]
            C3[Centrifugo 3]
            RD[(Redis<br/>History)]

            C1 <--> RD
            C2 <--> RD
            C3 <--> RD
        end

        subgraph Backend["Backend Services"]
            AG[API Gateway]
            PG[(PostgreSQL<br/>messages)]

            AG <--> PG
        end
    end

    CF -->|WebSocket| NG
    CS -->|REST API| NG
    NG -->|/connection/websocket| Centrifugo
    NG -->|/api/chats/.../sync| AG
    AG -->|Publish| Centrifugo
```

### Ключевые компоненты

| Компонент | Роль |
|-----------|------|
| **Centrifuge Client** | WebSocket клиент с автоматическим переподключением |
| **chat.ts Store** | Pinia store для управления сообщениями и синхронизацией |
| **network.ts Store** | Отслеживание состояния сети |
| **localStorage** | Персистентное хранение seq_num между сессиями |
| **Centrifugo** | WebSocket сервер с поддержкой history и recovery |
| **Redis** | Хранилище истории сообщений для recovery |
| **PostgreSQL** | Основное хранилище сообщений (источник истины) |

---

## Фаза 1: Нормальная работа

При нормальной работе сообщения доставляются через WebSocket в реальном времени.

```mermaid
sequenceDiagram
    autonumber
    participant UA as User A (Browser)
    participant API as API Gateway
    participant PG as PostgreSQL
    participant CF as Centrifugo
    participant RD as Redis
    participant UB as User B (Browser)

    UA->>API: POST /api/chats/{id}/messages<br/>{content: "Hello"}
    API->>PG: INSERT message<br/>RETURNING seq_num
    PG-->>API: seq_num = 42
    API->>CF: Publish to user:{userB_id}
    CF->>RD: Store in history<br/>(user:* history_size=50)
    CF->>UB: WebSocket push<br/>{type: "message.created", seq_num: 42}
    API-->>UA: 200 OK {id, seq_num: 42}

    Note over UB: handleCentrifugoEvent()
    UB->>UB: messages.push(msg)
    UB->>UB: updateLastSeqNum(chatId, 42)
    UB->>UB: localStorage.setItem('chat_seq_nums', {...})
```

### Что происходит

1. **User A** отправляет сообщение через REST API
2. **API Gateway** сохраняет в PostgreSQL и получает `seq_num`
3. **API Gateway** публикует событие в Centrifugo
4. **Centrifugo** сохраняет в Redis history и отправляет через WebSocket
5. **User B** получает сообщение, обновляет store и localStorage

---

## Фаза 2: Потеря соединения

При потере интернета клиент сохраняет последний известный `seq_num`, а Centrifugo продолжает накапливать сообщения в Redis.

```mermaid
sequenceDiagram
    autonumber
    participant UB as User B (Browser)
    participant CF as Centrifugo
    participant RD as Redis
    participant UA as User A (Browser)
    participant API as API Gateway

    Note over UB: Интернет пропал

    UB->>UB: centrifuge.on('disconnected')
    UB->>UB: networkStore.setWebSocketConnected(false)

    Note over UB: localStorage сохранён:<br/>{chatId: 42}

    rect rgb(255, 230, 230)
        Note over UA,API: Пока User B offline, User A отправляет сообщения
        UA->>API: POST message (seq=43)
        API->>CF: Publish
        CF->>RD: Store in history
        Note over CF: User B offline -<br/>сообщение в Redis

        UA->>API: POST message (seq=44)
        API->>CF: Publish
        CF->>RD: Store in history

        UA->>API: POST message (seq=45)
        API->>CF: Publish
        CF->>RD: Store in history
    end

    Note over RD: Redis хранит:<br/>seq 43, 44, 45<br/>для user:{userB_id}
```

### Что сохраняется

- **localStorage**: последний `seq_num` для каждого чата
- **Redis**: пропущенные сообщения (до 50 шт, до 1 часа для `user:*`)
- **PostgreSQL**: все сообщения (источник истины)

---

## Фаза 3: Centrifugo Recovery

При восстановлении соединения Centrifugo автоматически отправляет пропущенные сообщения из Redis history.

```mermaid
sequenceDiagram
    autonumber
    participant UB as User B (Browser)
    participant CF as Centrifugo
    participant RD as Redis

    Note over UB: Интернет восстановлен

    UB->>UB: centrifuge автоматически<br/>переподключается

    Note over UB: minReconnectDelay: 500ms<br/>maxReconnectDelay: 20s

    UB->>CF: WebSocket Connect
    CF-->>UB: Connected

    UB->>UB: centrifuge.on('connected')
    UB->>UB: subscribeToUserChannel()<br/>with recoverable: true

    UB->>CF: Subscribe user:{id}<br/>with recovery

    CF->>RD: Get missed messages<br/>from history
    RD-->>CF: [seq=43, seq=44, seq=45]

    CF-->>UB: subscribed event<br/>{wasRecovering: true, recovered: true}

    Note over UB: subscription.on('subscribed')

    loop Для каждого пропущенного сообщения
        CF->>UB: publication event<br/>{seq_num: 43, content: "..."}
        UB->>UB: handleCentrifugoEvent()
        UB->>UB: messages.push(msg)
        UB->>UB: updateLastSeqNum()
    end

    Note over UB: UI обновляется<br/>автоматически (Vue reactivity)
```

### Ключевые параметры

```typescript
// Подписка с включенным recovery
subscription = centrifuge.newSubscription(`user:${userId}`, {
  recoverable: true,  // Включает автоматическое восстановление
  getToken: async () => { ... }
})

// Обработка результата recovery
subscription.on('subscribed', (ctx) => {
  if (ctx.wasRecovering && ctx.recovered) {
    // Успех - сообщения придут через publication events
  } else if (ctx.wasRecovering && !ctx.recovered) {
    // Неудача - используем API fallback
    syncAllChatsAfterReconnect()
  }
})
```

---

## Фаза 4: API Fallback

Если Centrifugo Recovery не сработал (offline > 1 часа или пропущено > 50 сообщений), используется синхронизация через REST API.

```mermaid
sequenceDiagram
    autonumber
    participant UB as User B (Browser)
    participant CF as Centrifugo
    participant API as API Gateway
    participant PG as PostgreSQL

    Note over UB: Интернет восстановлен

    UB->>CF: Subscribe with recovery
    CF-->>UB: subscribed event<br/>{wasRecovering: true, recovered: false}

    Note over UB,CF: Recovery failed:<br/>- offline > 1 hour<br/>- missed > 50 messages

    UB->>UB: syncAllChatsAfterReconnect()

    UB->>API: GET /api/chats
    API-->>UB: {chats: [...]}

    Note over UB: Читаем из localStorage:<br/>lastSeqNum = 42

    UB->>API: GET /api/chats/{id}/messages/sync<br/>?after_seq=42&limit=100

    API->>PG: SELECT * FROM messages<br/>WHERE seq_num > 42<br/>ORDER BY seq_num
    PG-->>API: [seq=43, 44, 45, ...]
    API-->>UB: {messages: [...], has_more: false}

    loop Для каждого сообщения
        UB->>UB: Проверка дубликатов
        UB->>UB: messages.push(msg)
        UB->>UB: updateLastSeqNum()
    end

    UB->>UB: messages.sort(by seq_num)
    UB->>UB: localStorage.setItem()
```

### API Endpoint

```
GET /api/chats/{chatId}/messages/sync?after_seq=42&limit=100

Response:
{
  "messages": [
    { "id": "...", "seq_num": 43, "content": "...", ... },
    { "id": "...", "seq_num": 44, "content": "...", ... }
  ],
  "has_more": false
}
```

---

## Состояния соединения

```mermaid
stateDiagram-v2
    [*] --> Disconnected

    Disconnected --> Connecting
    Connecting --> Connected
    Connecting --> Disconnected

    Connected --> Subscribing

    Subscribing --> Subscribed
    Subscribing --> Recovering

    Recovering --> Subscribed
    Recovering --> APIFallback

    APIFallback --> Subscribed

    Subscribed --> Disconnected

    Disconnected: Нет соединения
    Connecting: Подключение...
    Connected: WebSocket открыт
    Subscribing: Подписка на канал
    Subscribed: Подписка активна
    Recovering: Восстановление сообщений
    APIFallback: Синхронизация через API
```

**Переходы между состояниями:**

| Из | В | Триггер |
|----|---|---------|
| Start | Disconnected | Старт приложения |
| Disconnected | Connecting | `centrifuge.connect()` |
| Connecting | Connected | Успешное подключение |
| Connecting | Disconnected | Таймаут / Ошибка |
| Connected | Subscribing | Подписка на `user:*` канал |
| Subscribing | Subscribed | Подписка успешна |
| Subscribing | Recovering | `wasRecovering=true` |
| Recovering | Subscribed | `recovered=true` - сообщения доставлены |
| Recovering | APIFallback | `recovered=false` - нужен fallback |
| APIFallback | Subscribed | Синхронизация завершена |
| Subscribed | Disconnected | Потеря соединения |

---

## Хранение seq_num

```mermaid
flowchart LR
    subgraph Sources["Источники обновления seq_num"]
        WS[WebSocket<br/>publication event]
        REST[REST API<br/>response]
        SEND[Отправка<br/>сообщения]
    end

    subgraph Storage["Хранилище"]
        STORE[chat.ts Store<br/>lastSeqNums Map]
        LS[(localStorage<br/>chat_seq_nums)]
    end

    subgraph Usage["Использование"]
        SYNC[syncMessagesAfterReconnect<br/>after_seq=X]
        RELOAD[После перезагрузки<br/>страницы]
    end

    WS --> STORE
    REST --> STORE
    SEND --> STORE

    STORE <--> LS

    LS --> SYNC
    LS --> RELOAD
```

### Формат хранения

```javascript
// localStorage key: "chat_seq_nums"
{
  "chat-uuid-1": 156,   // последний известный seq_num
  "chat-uuid-2": 89,
  "chat-uuid-3": 234
}
```

### Когда обновляется

| Событие | Действие |
|---------|----------|
| Получение сообщения через WebSocket | `updateLastSeqNum(chatId, msg.seq_num)` |
| Загрузка сообщений через REST | `updateLastSeqNum(chatId, maxSeqNum)` |
| Отправка собственного сообщения | `updateLastSeqNum(chatId, response.seq_num)` |

---

## Конфигурация

```mermaid
flowchart TB
    subgraph Centrifugo["Centrifugo Config"]
        direction TB
        UC["user:* namespace"]
        CC["chat:* namespace"]

        UC --- UCH["history_size: 50"]
        UC --- UCT["history_ttl: 1h"]
        UC --- UCR["force_recovery: true"]

        CC --- CCH["history_size: 100"]
        CC --- CCT["history_ttl: 24h"]
        CC --- CCR["force_recovery: true"]
    end

    subgraph VueClient["Vue Centrifuge Client"]
        direction TB
        RC["Reconnect Config"]
        SC["Subscription Config"]

        RC --- RCM["minReconnectDelay: 500ms"]
        RC --- RCX["maxReconnectDelay: 20s"]
        RC --- RCT["timeout: 10s"]

        SC --- SCR["recoverable: true"]
    end

    subgraph API["API Sync Endpoint"]
        direction TB
        EP["GET /messages/sync"]
        EP --- EPA["after_seq: number"]
        EP --- EPL["limit: number"]
        EP --- EPR["returns: messages[], has_more"]
    end
```

### Файлы конфигурации

| Файл | Параметры |
|------|-----------|
| `deployments/centrifugo/config.json` | history_size, history_ttl, force_recovery |
| `services/api-gateway/web/src/stores/chat.ts` | reconnect delays, recoverable |

### Centrifugo Config

```json
{
  "namespaces": [
    {
      "name": "user",
      "history_size": 50,
      "history_ttl": "1h",
      "force_recovery": true
    },
    {
      "name": "chat",
      "history_size": 100,
      "history_ttl": "24h",
      "force_recovery": true
    }
  ]
}
```

### Vue Client Config

```typescript
centrifuge = new Centrifuge(wsUrl, {
  token,
  minReconnectDelay: 500,      // 500ms
  maxReconnectDelay: 20000,    // 20 seconds
  timeout: 10000,              // 10 seconds
  maxServerPingDelay: 15000,   // 15 seconds
})

subscription = centrifuge.newSubscription(channel, {
  recoverable: true,
  getToken: async () => { ... }
})
```

---

## Временная диаграмма

```mermaid
gantt
    title Сценарий потери и восстановления связи
    dateFormat X
    axisFormat %s

    section Device A (online)
    Отправка msg1 (seq=1)    :a1, 0, 1
    Отправка msg2 (seq=2)    :a2, 3, 1
    Отправка msg3 (seq=3)    :a3, 5, 1
    Отправка msg4 (seq=4)    :a4, 7, 1
    Отправка msg5 (seq=5)    :a5, 9, 1

    section Device B
    Получение msg1           :b1, 0, 1
    OFFLINE                  :crit, offline, 2, 8
    Reconnect + Recovery     :b2, 10, 2
    Получение msg2-5         :b3, 12, 1

    section Centrifugo Redis
    Store msg2               :r2, 3, 7
    Store msg3               :r3, 5, 7
    Store msg4               :r4, 7, 5
    Store msg5               :r5, 9, 3

    section localStorage (Device B)
    seq_num = 1              :l1, 1, 11
    seq_num = 5              :l2, 13, 2
```

---

## Multi-Device сценарий

Когда у одного пользователя несколько устройств и одно из них теряет соединение.

```mermaid
sequenceDiagram
    autonumber
    participant D1 as Device 1<br/>(Laptop)
    participant D2 as Device 2<br/>(Phone - OFFLINE)
    participant D3 as Device 3<br/>(Tablet)
    participant CF as Centrifugo
    participant RD as Redis
    participant API as API Gateway
    participant OT as Other User

    Note over D1,D3: Все устройства одного пользователя<br/>подписаны на user:{userId}

    rect rgb(200, 255, 200)
        Note over D1,D3: Фаза 1: Все online
        OT->>API: POST message
        API->>CF: Publish to user:{userId}
        par Доставка на все устройства
            CF->>D1: WebSocket push (seq=10)
            CF->>D2: WebSocket push (seq=10)
            CF->>D3: WebSocket push (seq=10)
        end
    end

    rect rgb(255, 230, 230)
        Note over D2: Фаза 2: Device 2 теряет интернет
        D2--xCF: Connection lost
        D2->>D2: localStorage: seq_num=10

        Note over D1,D3: Device 1 и Device 3 работают нормально

        OT->>API: POST message (seq=11)
        API->>CF: Publish
        CF->>RD: Store in history
        par Доставка на online устройства
            CF->>D1: WebSocket push (seq=11)
            CF->>D3: WebSocket push (seq=11)
        end
        Note over D2: Не получено

        OT->>API: POST message (seq=12)
        API->>CF: Publish
        CF->>RD: Store in history
        par
            CF->>D1: WebSocket push (seq=12)
            CF->>D3: WebSocket push (seq=12)
        end
        Note over D2: Не получено

        OT->>API: POST message (seq=13)
        API->>CF: Publish
        CF->>RD: Store in history
        par
            CF->>D1: WebSocket push (seq=13)
            CF->>D3: WebSocket push (seq=13)
        end
        Note over D2: Не получено
    end

    rect rgb(230, 230, 255)
        Note over D2: Фаза 3: Device 2 восстанавливает соединение
        D2->>CF: Reconnect + Subscribe<br/>with recoverable: true

        CF->>RD: Get missed messages<br/>since last offset
        RD-->>CF: [seq=11, 12, 13]

        CF-->>D2: subscribed<br/>{wasRecovering: true, recovered: true}

        loop Доставка пропущенных
            CF->>D2: publication (seq=11)
            CF->>D2: publication (seq=12)
            CF->>D2: publication (seq=13)
        end

        D2->>D2: localStorage: seq_num=13
        Note over D2: UI обновлён<br/>Все сообщения получены
    end

    rect rgb(200, 255, 200)
        Note over D1,D3: Фаза 4: Все устройства синхронизированы
        Note over D1: seq_num=13
        Note over D2: seq_num=13
        Note over D3: seq_num=13
    end
```

### Ключевые особенности Multi-Device

| Аспект | Поведение |
|--------|-----------|
| **Канал подписки** | Все устройства подписаны на `user:{userId}` |
| **Независимость** | Каждое устройство имеет свой WebSocket и свой `seq_num` |
| **localStorage** | Каждое устройство хранит свой `seq_num` локально |
| **Recovery** | Centrifugo восстанавливает сообщения для каждого устройства отдельно |
| **Дубликаты** | Клиент фильтрует дубликаты по `message.id` |

### Почему это работает

1. **Centrifugo tracking** - Centrifugo отслеживает позицию каждого клиента отдельно
2. **Client offset** - При reconnect клиент сообщает свой последний offset
3. **Redis history** - Пропущенные сообщения хранятся в Redis до 1 часа (user:*) или 24 часов (chat:*)
4. **Automatic recovery** - `recoverable: true` включает автоматическую доставку пропущенных сообщений

---

## Тестирование

### E2E тесты

Расположение: `services/api-gateway/web/e2e-selenium/tests/websocket-recovery.spec.ts`

| Тест | Описание |
|------|----------|
| `Centrifugo Recovery` | Автоматическое восстановление через WebSocket |
| `API Fallback Sync` | Синхронизация через REST при отказе recovery |
| `Multi-Device Scenario` | Несколько устройств, одно offline |
| `seq_num Tracking` | Персистентность seq_num в localStorage |
| `Reconnect with Pending` | Отправка сообщений во время offline |

### Запуск тестов

```powershell
# Все тесты
.\services\api-gateway\web\run-recovery-test.ps1

# Конкретный тест
.\run-recovery-test.ps1 -Test automatic
.\run-recovery-test.ps1 -Test fallback
.\run-recovery-test.ps1 -Test multidevice

# Headless режим
.\run-recovery-test.ps1 -Headless
```

### Ручное тестирование

1. Откройте чат в двух браузерах (User A и User B)
2. Откройте DevTools → Network в браузере User B
3. Включите "Offline" режим
4. Отправьте несколько сообщений от User A
5. Отключите "Offline" режим у User B
6. Убедитесь, что сообщения появились автоматически (без refresh)

---

## Ограничения

| Ограничение | Значение | Fallback |
|-------------|----------|----------|
| Max offline time (user:*) | 1 час | API sync |
| Max missed messages (user:*) | 50 | API sync |
| Max offline time (chat:*) | 24 часа | API sync |
| Max missed messages (chat:*) | 100 | API sync |

---

## Связанные файлы

- `deployments/centrifugo/config.json` - конфигурация Centrifugo
- `services/api-gateway/web/src/stores/chat.ts` - Vue store с логикой recovery
- `services/api-gateway/web/src/stores/network.ts` - отслеживание состояния сети
- `services/api-gateway/internal/handler/chat.go` - API endpoint `/messages/sync`
- `services/api-gateway/web/e2e-selenium/tests/websocket-recovery.spec.ts` - E2E тесты
