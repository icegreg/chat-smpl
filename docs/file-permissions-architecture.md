# Архитектура файловых прав доступа

## Обзор

Документ описывает систему групповых прав доступа к файлам в чатах.

**Ключевые концепции:**
- Каждый чат имеет две группы: `moderate_all` (полный доступ) и `view_all` (просмотр/скачивание)
- Модераторы видят ВСЕ файлы чата
- Обычные участники видят только файлы, загруженные ПОСЛЕ их вступления в чат
- Права назначаются автоматически через триггеры PostgreSQL

## Модель данных (реальная схема БД)

### Полная ER-диаграмма

```mermaid
erDiagram
    %% ===== USERS DOMAIN =====
    USERS ||--o{ REFRESH_TOKENS : "имеет токены"
    USERS ||--o{ GROUP_MEMBERS : "состоит в группах"
    GROUPS ||--o{ GROUP_MEMBERS : "содержит"

    USERS {
        uuid id PK
        varchar username UK
        varchar email UK
        varchar password_hash
        varchar role "owner/moderator/user/guest"
        timestamp created_at
        timestamp updated_at
    }

    GROUPS {
        uuid id PK
        varchar name
        text description
        timestamp created_at
        timestamp updated_at
    }

    GROUP_MEMBERS {
        uuid id PK
        uuid group_id FK
        uuid user_id FK
        timestamp joined_at
    }

    %% ===== FILES DOMAIN (независимый) =====
    FILES ||--o{ FILE_LINKS : "имеет ссылки"
    FILES ||--o{ FILE_SHARE_LINKS : "публичные ссылки"
    FILE_LINKS ||--o{ FILE_LINK_PERMISSIONS : "индивид. права"
    FILE_LINKS ||--o{ MESSAGE_FILE_ATTACHMENTS : "вложения"

    FILES {
        uuid id PK
        varchar filename
        varchar original_filename
        varchar content_type
        bigint size
        varchar file_path
        uuid uploaded_by
        timestamp uploaded_at
        varchar status "active/deleted"
        jsonb metadata
    }

    FILE_LINKS {
        uuid id PK
        uuid file_id FK
        uuid chat_id FK "NULL = standalone"
        uuid uploaded_by
        timestamp uploaded_at
        boolean is_deleted
    }

    FILE_LINK_PERMISSIONS {
        uuid id PK
        uuid file_link_id FK
        uuid user_id
        boolean can_view
        boolean can_download
        boolean can_delete
    }

    FILE_SHARE_LINKS {
        uuid id PK
        uuid file_id FK
        varchar token UK
        varchar password "nullable"
        int max_downloads "nullable"
        int download_count
        uuid created_by
        timestamp expires_at "nullable"
        boolean is_active
    }

    MESSAGE_FILE_ATTACHMENTS {
        uuid id PK
        uuid message_id
        uuid file_link_id FK
        int sort_order
    }

    %% ===== CHAT DOMAIN =====
    CHATS ||--o{ CHAT_PARTICIPANTS : "участники"
    CHATS ||--o{ MESSAGES : "сообщения"
    MESSAGES ||--o{ MESSAGE_REACTIONS : "реакции"
    MESSAGES ||--o{ MESSAGE_READERS : "прочитано"

    CHATS {
        uuid id PK
        varchar name
        varchar chat_type "private/group/channel"
        uuid created_by
        timestamp created_at
        timestamp updated_at
    }

    CHAT_PARTICIPANTS {
        uuid id PK
        uuid chat_id FK
        uuid user_id
        varchar role "admin/member/readonly"
        timestamp joined_at
    }

    MESSAGES {
        uuid id PK
        uuid chat_id FK
        uuid parent_id FK "nullable"
        uuid sender_id
        text content
        timestamp sent_at
        boolean is_deleted
    }

    %% ===== CHAT-FILES BRIDGE (групповые права) =====
    CHATS ||--o| CHAT_FILE_ACCESS_GROUPS : "группы доступа к файлам"
    CHAT_FILE_ACCESS_GROUPS ||--|| GROUPS : moderate_group
    CHAT_FILE_ACCESS_GROUPS ||--|| GROUPS : view_group
    FILE_LINKS ||--o{ FILE_LINK_GROUP_PERMISSIONS : "групповые права"
    FILE_LINK_GROUP_PERMISSIONS }o--|| GROUPS : "для группы"

    CHAT_FILE_ACCESS_GROUPS {
        uuid id PK
        uuid chat_id FK UK
        uuid moderate_group_id FK
        uuid view_group_id FK
    }

    FILE_LINK_GROUP_PERMISSIONS {
        uuid id PK
        uuid file_link_id FK
        uuid group_id FK
        boolean can_view
        boolean can_download
        boolean can_delete
        timestamp valid_from "nullable"
    }
```

### Три сценария использования файлов

| Сценарий | chat_id | Как работают права |
|----------|---------|-------------------|
| **Standalone файл** | `NULL` | Только `file_link_permissions` (индивидуальные) |
| **Публичная ссылка** | `NULL` | Через `file_share_links` (по токену) |
| **Файл в чате** | `UUID` | `file_link_permissions` + `file_link_group_permissions` |

### Ключевые связи

```
FILES (физический файл)
  └── FILE_LINKS (контекстные ссылки, может быть много на один файл)
        ├── chat_id = NULL → standalone файл
        │     └── FILE_LINK_PERMISSIONS (кому дали доступ вручную)
        │
        └── chat_id = UUID → файл в чате
              ├── FILE_LINK_PERMISSIONS (индивидуальные права)
              └── FILE_LINK_GROUP_PERMISSIONS → GROUPS
                    ├── moderate_all → полный доступ
                    └── view_all → скачивание (с учётом joined_at)
```

## Иерархия прав доступа

```mermaid
flowchart TD
    subgraph AccessLevels["Уровни доступа (от высшего к низшему)"]
        DELETE["delete - Полный доступ (просмотр, скачивание, удаление)"]
        DOWNLOAD["download - Просмотр и скачивание"]
        VIEW["view - Только просмотр метаданных"]
        NONE["none - Нет доступа"]
    end

    DELETE --> DOWNLOAD
    DOWNLOAD --> VIEW
    VIEW --> NONE

    subgraph Roles["Кто какой доступ получает"]
        UPLOADER["Загрузивший файл"] --> DELETE
        MODERATOR["Модератор чата<br/>(группа moderate_all)"] --> DELETE
        VIEWER["Участник чата<br/>(группа view_all)"] --> DOWNLOAD
        OUTSIDER["Не участник"] --> NONE
    end
```

## Создание чата

При создании чата триггер БД автоматически создаёт группы доступа к файлам.

```mermaid
sequenceDiagram
    participant User as Пользователь
    participant ChatService as Сервис чатов
    participant PostgreSQL as PostgreSQL
    participant Trigger as Триггер БД

    User->>ChatService: Создать чат
    ChatService->>PostgreSQL: INSERT INTO chats
    PostgreSQL->>Trigger: Срабатывает AFTER INSERT триггер

    Note over Trigger: create_chat_file_access_groups()

    Trigger->>PostgreSQL: Создать группу<br/>chat_{id}_moderate_all
    Trigger->>PostgreSQL: Создать группу<br/>chat_{id}_view_all
    Trigger->>PostgreSQL: Связать группы с чатом<br/>(chat_file_access_groups)
    Trigger->>PostgreSQL: Добавить создателя<br/>в moderate_all

    PostgreSQL-->>ChatService: Чат создан
    ChatService-->>User: Успех
```

## Загрузка файла в чат

```mermaid
sequenceDiagram
    participant User as Пользователь
    participant APIGateway as API Gateway
    participant FilesService as Сервис файлов
    participant Storage as Хранилище
    participant PostgreSQL as PostgreSQL
    participant Trigger as Триггер БД

    User->>APIGateway: Загрузить файл в чат
    APIGateway->>FilesService: UploadWithChat(file, chatID)
    FilesService->>Storage: Сохранить файл
    Storage-->>FilesService: Путь к файлу

    FilesService->>PostgreSQL: INSERT INTO files
    FilesService->>PostgreSQL: INSERT INTO file_links<br/>(с указанием chat_id)

    PostgreSQL->>Trigger: Срабатывает AFTER INSERT триггер
    Note over Trigger: add_file_link_group_permissions()

    Trigger->>PostgreSQL: Получить группы доступа чата
    Trigger->>PostgreSQL: Добавить права для moderate_all<br/>(полный доступ)
    Trigger->>PostgreSQL: Добавить права для view_all<br/>(просмотр/скачивание)

    FilesService->>PostgreSQL: Добавить индивидуальные права<br/>загрузившему (полный доступ)

    PostgreSQL-->>FilesService: Успех
    FilesService-->>APIGateway: Ответ с данными файла
    APIGateway-->>User: Файл загружен
```

## Скачивание файла (проверка прав)

```mermaid
sequenceDiagram
    participant User as Пользователь
    participant APIGateway as API Gateway
    participant FilesService as Сервис файлов
    participant PostgreSQL as PostgreSQL
    participant Storage as Хранилище

    User->>APIGateway: Скачать файл (linkID)
    APIGateway->>FilesService: Download(linkID, userID)
    FilesService->>PostgreSQL: Получить FileLink
    PostgreSQL-->>FilesService: FileLink (с chat_id)

    FilesService->>PostgreSQL: check_file_access(linkID, userID)

    Note over PostgreSQL: Логика проверки прав

    alt Пользователь загрузил этот файл
        PostgreSQL-->>FilesService: "delete"
    else Пользователь в группе moderate_all
        PostgreSQL-->>FilesService: "delete"
    else Пользователь в view_all И вступил ДО загрузки файла
        PostgreSQL-->>FilesService: "download"
    else Есть индивидуальные права
        PostgreSQL-->>FilesService: согласно правам
    else Нет доступа
        PostgreSQL-->>FilesService: "none"
    end

    alt Уровень доступа >= download
        FilesService->>PostgreSQL: Получить данные файла
        FilesService->>Storage: Получить содержимое
        Storage-->>FilesService: Поток данных
        FilesService-->>APIGateway: Файл + метаданные
        APIGateway-->>User: Скачивание файла
    else Доступ запрещён
        FilesService-->>APIGateway: ErrAccessDenied
        APIGateway-->>User: 403 Forbidden
    end
```

## Повышение до модератора

```mermaid
sequenceDiagram
    participant Admin as Администратор
    participant ChatService as Сервис чатов
    participant FilesService as Сервис файлов
    participant PostgreSQL as PostgreSQL

    Admin->>ChatService: Повысить пользователя до модератора
    ChatService->>ChatService: Обновить role в chat_participants
    ChatService->>FilesService: PromoteToModerator(chatID, userID)

    FilesService->>PostgreSQL: Удалить из группы view_all<br/>(если был там)
    FilesService->>PostgreSQL: Добавить в группу moderate_all

    PostgreSQL-->>FilesService: Успех
    FilesService-->>ChatService: Успех
    ChatService-->>Admin: Пользователь повышен

    Note over PostgreSQL: Теперь у пользователя доступ DELETE<br/>ко ВСЕМ файлам чата
```

## Понижение модератора

```mermaid
sequenceDiagram
    participant Admin as Администратор
    participant ChatService as Сервис чатов
    participant FilesService as Сервис файлов
    participant PostgreSQL as PostgreSQL

    Admin->>ChatService: Понизить модератора
    ChatService->>ChatService: Обновить role в chat_participants
    ChatService->>FilesService: DemoteFromModerator(chatID, userID)

    FilesService->>PostgreSQL: Удалить из группы moderate_all
    FilesService->>PostgreSQL: Добавить в группу view_all

    PostgreSQL-->>FilesService: Успех
    FilesService-->>ChatService: Успех
    ChatService-->>Admin: Пользователь понижен

    Note over PostgreSQL: Теперь доступ DOWNLOAD только к файлам,<br/>загруженным ПОСЛЕ вступления в чат
```

## Новый участник вступает в чат

```mermaid
sequenceDiagram
    participant NewUser as Новый участник
    participant ChatService as Сервис чатов
    participant FilesService as Сервис файлов
    participant PostgreSQL as PostgreSQL

    NewUser->>ChatService: Вступить в чат / Добавлен в чат
    ChatService->>PostgreSQL: INSERT INTO chat_participants<br/>(с меткой времени joined_at)
    ChatService->>FilesService: AddParticipantAccess(chatID, userID)

    FilesService->>PostgreSQL: Добавить в группу view_all

    PostgreSQL-->>FilesService: Успех
    FilesService-->>ChatService: Успех
    ChatService-->>NewUser: Вступление подтверждено

    Note over PostgreSQL: Пользователь может скачивать файлы,<br/>загруженные ПОСЛЕ его вступления (joined_at)
```

## Участник покидает чат

```mermaid
sequenceDiagram
    participant User as Пользователь
    participant ChatService as Сервис чатов
    participant FilesService as Сервис файлов
    participant PostgreSQL as PostgreSQL

    User->>ChatService: Покинуть чат / Удалён из чата
    ChatService->>PostgreSQL: DELETE FROM chat_participants
    ChatService->>FilesService: RemoveParticipantAccess(chatID, userID)

    FilesService->>PostgreSQL: Удалить из группы moderate_all
    FilesService->>PostgreSQL: Удалить из группы view_all

    PostgreSQL-->>FilesService: Успех
    FilesService-->>ChatService: Успех
    ChatService-->>User: Вышел из чата

    Note over PostgreSQL: Пользователь теряет весь групповой доступ.<br/>Индивидуальные права сохраняются (если были)
```

## Алгоритм проверки прав (check_file_access)

```mermaid
flowchart TD
    START([Вызов check_file_access]) --> GET_LINK[Получить информацию о file_link]
    GET_LINK --> CHECK_DELETED{Файл удалён?}

    CHECK_DELETED -->|Да| NONE[Вернуть 'none']
    CHECK_DELETED -->|Нет| CHECK_UPLOADER{Пользователь<br/>загрузил файл?}

    CHECK_UPLOADER -->|Да| DELETE[Вернуть 'delete']
    CHECK_UPLOADER -->|Нет| CHECK_INDIVIDUAL[Проверить индивидуальные права]

    CHECK_INDIVIDUAL --> HAS_DELETE{Есть право<br/>can_delete?}
    HAS_DELETE -->|Да| DELETE
    HAS_DELETE -->|Нет| CHECK_CHAT{Файл привязан<br/>к чату?}

    CHECK_CHAT -->|Нет| CHECK_INDIVIDUAL_DOWNLOAD{Есть право<br/>can_download?}
    CHECK_INDIVIDUAL_DOWNLOAD -->|Да| DOWNLOAD[Вернуть 'download']
    CHECK_INDIVIDUAL_DOWNLOAD -->|Нет| CHECK_INDIVIDUAL_VIEW{Есть право<br/>can_view?}
    CHECK_INDIVIDUAL_VIEW -->|Да| VIEW[Вернуть 'view']
    CHECK_INDIVIDUAL_VIEW -->|Нет| NONE

    CHECK_CHAT -->|Да| CHECK_MODERATE{Пользователь в<br/>группе moderate_all?}
    CHECK_MODERATE -->|Да| DELETE
    CHECK_MODERATE -->|Нет| CHECK_VIEW{Пользователь в<br/>группе view_all?}

    CHECK_VIEW -->|Нет| CHECK_INDIVIDUAL_DOWNLOAD
    CHECK_VIEW -->|Да| GET_JOINED[Получить дату вступления<br/>пользователя в чат]

    GET_JOINED --> CHECK_TIME{Файл загружен<br/>ПОСЛЕ вступления?}
    CHECK_TIME -->|Да| DOWNLOAD
    CHECK_TIME -->|Нет| CHECK_INDIVIDUAL_DOWNLOAD
```

## Архитектура сервисов

```mermaid
flowchart TB
    subgraph Frontend["Фронтенд"]
        VUE[Vue.js SPA]
    end

    subgraph APIGateway["API Gateway :8080"]
        HANDLER[File Handler]
    end

    subgraph FilesService["Сервис файлов :8082"]
        SVC[FileService]
        REPO[FileRepository]
    end

    subgraph ChatService["Сервис чатов :50051"]
        CHAT_SVC[ChatService]
    end

    subgraph Infrastructure["Инфраструктура"]
        PG[(PostgreSQL)]
        STORAGE[(Файловое хранилище)]
    end

    VUE -->|REST API| HANDLER
    HANDLER -->|gRPC| SVC
    SVC --> REPO
    REPO -->|SQL| PG
    SVC -->|Файловый I/O| STORAGE

    CHAT_SVC -->|gRPC| SVC

    subgraph PostgreSQL_Functions["Функции и триггеры PostgreSQL"]
        PG --> TRIGGER1[trg_create_chat_file_access_groups]
        PG --> TRIGGER2[trg_add_file_link_group_permissions]
        PG --> FUNC[check_file_access]
    end
```

## Маппинг ролей: Чат → Файловые права

| Роль в чате | Группа файловых прав | Уровень доступа |
|-------------|---------------------|-----------------|
| owner | moderate_all | DELETE (все файлы) |
| admin | moderate_all | DELETE (все файлы) |
| moderator | moderate_all | DELETE (все файлы) |
| member | view_all | DOWNLOAD (файлы после joined_at) |
| guest | view_all | DOWNLOAD (файлы после joined_at) |

## Временной доступ для группы view_all

```mermaid
gantt
    title Временная шкала доступа к файлам
    dateFormat  YYYY-MM-DD

    section Пользователь
    Вступил в чат           :milestone, m1, 2024-01-15, 0d

    section Файлы
    Файл A (до вступления)      :done, fileA, 2024-01-10, 2024-01-20
    Файл B (после вступления)   :active, fileB, 2024-01-16, 2024-01-25
    Файл C (после вступления)   :active, fileC, 2024-01-20, 2024-01-30

    section Доступ
    Нет доступа к Файлу A       :crit, noA, 2024-01-15, 2024-01-25
    Может скачать Файл B        :fileB, 2024-01-16, 2024-01-25
    Может скачать Файл C        :fileC, 2024-01-20, 2024-01-30
```

**Важно**: Модераторы (в группе moderate_all) имеют доступ ко ВСЕМ файлам, независимо от даты вступления.

---

## Связанные файлы

| Файл | Описание |
|------|----------|
| `migrations/files/002_chat_file_access_groups.sql` | Миграция БД с таблицами и триггерами |
| `services/files/internal/model/file.go` | Модели данных Go |
| `services/files/internal/repository/file.go` | Репозиторий с методами доступа к БД |
| `services/files/internal/service/file.go` | Бизнес-логика сервиса файлов |
| `services/files/internal/service/file_test.go` | Unit-тесты |
