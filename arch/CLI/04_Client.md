---
modified: 2025-11-25T15:20:12+03:00
created: 2025-11-25T10:59:12+03:00
---
# Client - Управление клиентами

## Обзор

Модуль управления клиентами предоставляет возможности для глобального поиска пользователей, просмотра истории участия в мероприятиях и анализа статистики.

**Важно:** Клиенты всегда рассматриваются в контексте мероприятий. Для детального анализа клиента в конкретном мероприятии используйте команды из модуля [Conference](03_Conference.md).

---

## Команды

### `rtuccli client find <query>`

Глобальный поиск клиентов.

**Синтаксис:**
```bash
rtuccli client find <query> [flags]
```

**Параметры:**
- `<query>` - поисковый запрос (email, имя, часть имени)

**Флаги:**
- `--device-id <id>` - поиск по ID устройства

**Пример 1: Поиск по имени**

```bash
$ rtuccli client find "frank"
```

**Вывод:**

```
SEARCH: "frank"

┌────────────────────────┬───────────────────┬───────────────┬─────────────┐
│ Client ID              │ Name/Email        │ Last Seen     │ Status      │
├────────────────────────┼───────────────────┼───────────────┼─────────────┤
│ client-f1a2b3          │ frank@company.com │ now           │ ● in conf   │
│ client-f4d5e6          │ franklin@other.org│ 2d ago        │ ○ offline   │
│ client-f7g8h9          │ Frank Smith       │ 5h ago        │ ○ offline   │
└────────────────────────┴───────────────────┴───────────────┴─────────────┘

Use 'rtuccli client <client-id>' for details
```

**Пример 2: Поиск по email**

```bash
$ rtuccli client find "frank@company.com"
```

**Вывод:**

```
SEARCH: "frank@company.com"

┌────────────────────────┬───────────────────┬───────────────┬─────────────┐
│ Client ID              │ Name/Email        │ Last Seen     │ Status      │
├────────────────────────┼───────────────────┼───────────────┼─────────────┤
│ client-f1a2b3          │ frank@company.com │ now           │ ● in conf   │
└────────────────────────┴───────────────────┴───────────────┴─────────────┘
```

**Пример 3: Поиск по device ID**

```bash
$ rtuccli client find --device-id "abc123"
```

**Вывод:**

```
SEARCH: device-id "abc123"

┌────────────────────────┬───────────────────┬─────────────────┬─────────────┐
│ Client ID              │ Name/Email        │ Device          │ Status      │
├────────────────────────┼───────────────────┼─────────────────┼─────────────┤
│ client-f1a2b3          │ frank@company.com │ Desktop Linux   │ ● in conf   │
└────────────────────────┴───────────────────┴─────────────────┴─────────────┘
```

---

### `rtuccli client <client-id>`

Глобальная информация о клиенте (история, устройства, статистика).

**Синтаксис:**
```bash
rtuccli client <client-id> [flags]
```

**Параметры:**
- `<client-id>` - ID или email клиента

**Пример:**

```bash
$ rtuccli client frank@company.com
```

**Вывод:**

```
CLIENT: frank@company.com
Client ID:    client-f1a2b3
Registered:   2023-06-15
Last Active:  now (in conference)

DEVICES USED:
┌─────────────────────────┬────────────────┬─────────────┐
│ Device                  │ App Version    │ Last Used   │
├─────────────────────────┼────────────────┼─────────────┤
│ Desktop Linux (Firefox) │ Web            │ now         │
│ Android 14              │ App v2.4.1     │ 3d ago      │
│ Web Chrome              │ Web            │ 14d ago     │
└─────────────────────────┴────────────────┴─────────────┘

CURRENT SESSION:
  Conference:   conf-456 "Q1 Planning Meeting"
  Quality:      ★☆☆☆ Poor
  Duration:     5m
  
  Use 'rtuccli conf conf-456 client frank@company.com' for session details

STATISTICS (last 30 days):
  Conferences:  23
  Total Time:   47h 32m
  Avg Quality:  ★★★☆ Good
  Issues:       12 (mostly high jitter)

RECENT CONFERENCES:
┌───────────┬─────────────────────┬─────────────┬──────────┬─────────┐
│ Conf ID   │ Name                │ Date        │ Duration │ Quality │
├───────────┼─────────────────────┼─────────────┼──────────┼─────────┤
│ conf-456  │ Q1 Planning Meeting │ now         │ 5m       │ ★☆☆☆    │
│ conf-451  │ Daily Standup       │ today 09:00 │ 15m      │ ★★★★    │
│ conf-448  │ Design Review       │ yesterday   │ 1h 20m   │ ★★★☆    │
└───────────┴─────────────────────┴─────────────┴──────────┴─────────┘

Use 'rtuccli client frank@company.com conferences' for full history
```

---

### `rtuccli client <client-id> conferences`

История участия клиента в мероприятиях.

**Синтаксис:**
```bash
rtuccli client <client-id> conferences [flags]
```

**Флаги:**
- `--active` - только текущие активные мероприятия
- `--last <duration>` - за последний период (например, 7d, 30d, 3m)

**Пример 1: Вся история (последние 30 дней)**

```bash
$ rtuccli client frank@company.com conferences
```

**Вывод:**

```
CLIENT: frank@company.com | Conferences (last 30 days)

┌───────────┬──────────────────────────┬─────────────┬──────────┬─────────┐
│ Conf ID   │ Name                     │ Date        │ Duration │ Quality │
├───────────┼──────────────────────────┼─────────────┼──────────┼─────────┤
│ conf-456  │ Q1 Planning Meeting      │ today       │ 5m (now) │ ★☆☆☆    │
│ conf-451  │ Daily Standup            │ today 09:00 │ 15m      │ ★★★★    │
│ conf-448  │ Design Review            │ yesterday   │ 1h 20m   │ ★★★☆    │
│ conf-445  │ Daily Standup            │ 2d ago      │ 12m      │ ★★★★    │
│ conf-440  │ Sprint Retrospective     │ 3d ago      │ 45m      │ ★★★☆    │
│ conf-432  │ 1:1 with Manager         │ 4d ago      │ 28m      │ ★★★★    │
│ conf-428  │ Daily Standup            │ 5d ago      │ 14m      │ ★★★★    │
│ conf-419  │ Team Sync                │ 1w ago      │ 52m      │ ★★★☆    │
│ ...       │                          │             │          │         │
└───────────┴──────────────────────────┴─────────────┴──────────┴─────────┘

SUMMARY:
  Total:        23 conferences
  Total Time:   47h 32m
  Avg Quality:  ★★★☆ Good
  
Quality Distribution:
  ★★★★ Excellent:  14 (61%)
  ★★★☆ Good:        7 (30%)
  ★★☆☆ Normal:      1 (4%)
  ★☆☆☆ Poor:        1 (4%)

Use 'rtuccli conf <conf-id> client frank@company.com' to see session details
```

**Пример 2: Только активные мероприятия**

```bash
$ rtuccli client frank@company.com conferences --active
```

**Вывод:**

```
CLIENT: frank@company.com | Active Conferences (1)

┌───────────┬──────────────────────────┬──────────┬─────────┬─────────────┐
│ Conf ID   │ Name                     │ Duration │ Quality │ Status      │
├───────────┼──────────────────────────┼──────────┼─────────┼─────────────┤
│ conf-456  │ Q1 Planning Meeting      │ 5m       │ ★☆☆☆    │ ● connected │
└───────────┴──────────────────────────┴──────────┴─────────┴─────────────┘

Use 'rtuccli conf conf-456 client frank@company.com' for session details
```

**Пример 3: За последние 7 дней**

```bash
$ rtuccli client frank@company.com conferences --last 7d
```

**Вывод:**

```
CLIENT: frank@company.com | Conferences (last 7 days)

┌───────────┬──────────────────────────┬─────────────┬──────────┬─────────┐
│ Conf ID   │ Name                     │ Date        │ Duration │ Quality │
├───────────┼──────────────────────────┼─────────────┼──────────┼─────────┤
│ conf-456  │ Q1 Planning Meeting      │ today       │ 5m (now) │ ★☆☆☆    │
│ conf-451  │ Daily Standup            │ today 09:00 │ 15m      │ ★★★★    │
│ conf-448  │ Design Review            │ yesterday   │ 1h 20m   │ ★★★☆    │
│ conf-445  │ Daily Standup            │ 2d ago      │ 12m      │ ★★★★    │
│ conf-440  │ Sprint Retrospective     │ 3d ago      │ 45m      │ ★★★☆    │
│ conf-432  │ 1:1 with Manager         │ 4d ago      │ 28m      │ ★★★★    │
│ conf-428  │ Daily Standup            │ 5d ago      │ 14m      │ ★★★★    │
└───────────┴──────────────────────────┴─────────────┴──────────┴─────────┘

SUMMARY (7 days):
  Total:        7 conferences
  Total Time:   3h 19m
  Avg Quality:  ★★★☆ Good
```

---

## См. также

- [Conference - Управление мероприятиями](03_Conference.md) - детальный анализ клиентов в контексте мероприятий
- [Service - Управление сервисами](02_Service.md) - анализ сервисов, через которые подключены клиенты
- [README - Главная страница](README.md)
