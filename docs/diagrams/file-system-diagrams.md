# Диаграммы файловой системы чата

## 1. Диаграмма последовательности: Загрузка файла и отправка сообщения

```mermaid
sequenceDiagram
    autonumber
    participant Клиент as Клиент (Browser)
    participant GW as API Gateway
    participant FS as Files Service
    participant CS as Chat Service
    participant DB_F as БД Files
    participant DB_C as БД Chat
    participant Storage as Файловое хранилище
    participant MQ as RabbitMQ
    participant Centrifugo as Centrifugo

    Note over Клиент,Centrifugo: Сценарий: Отправка сообщения с файлом

    %% Шаг 1: Загрузка файла
    rect rgb(230, 245, 255)
        Note right of Клиент: Этап 1: Загрузка файла
        Клиент->>+GW: POST /api/files/upload<br/>(multipart/form-data + JWT)
        GW->>GW: Проверка JWT токена
        GW->>+FS: POST /files/upload<br/>(X-User-ID: user_uuid)
        FS->>FS: Валидация размера файла
        FS->>+Storage: Сохранить файл на диск
        Storage-->>-FS: Путь к файлу
        FS->>+DB_F: INSERT files (id, filename, path, ...)
        DB_F-->>-FS: file_id
        FS->>+DB_F: INSERT file_links (id, file_id, uploaded_by)
        DB_F-->>-FS: link_id
        FS->>+DB_F: INSERT file_link_permissions<br/>(link_id, user_id=uploader, can_view=true)
        DB_F-->>-FS: OK
        FS-->>-GW: {file_id, link_id, filename, size}
        GW-->>-Клиент: 201 Created {link_id, ...}
    end

    %% Шаг 2: Отправка сообщения
    rect rgb(230, 255, 230)
        Note right of Клиент: Этап 2: Отправка сообщения с файлом
        Клиент->>+GW: POST /api/chats/{chatId}/messages<br/>{content, file_link_ids: [link_id]}
        GW->>GW: Проверка JWT токена
        GW->>+CS: gRPC: SendMessage<br/>(chat_id, sender_id, content, file_link_ids)
        CS->>+DB_C: Проверка участия в чате
        DB_C-->>-CS: participant (role)
        CS->>CS: Проверка роли (can_write?)
        CS->>+DB_C: INSERT messages<br/>(id, chat_id, sender_id, content, ...)
        DB_C-->>-CS: message_id, seq_num
        CS->>+DB_C: INSERT message_file_attachments<br/>(message_id, file_link_id)
        DB_C-->>-CS: OK
        CS->>+DB_C: SELECT participant_ids FROM chat_participants
        DB_C-->>-CS: [user1, user2, user3, ...]
        CS-->>-GW: Message (с seq_num)

        %% Выдача прав на файлы
        GW->>+FS: POST /files/grant-permissions<br/>{link_ids, user_ids: participants}
        FS->>+DB_F: INSERT file_link_permissions<br/>(для каждого участника)
        DB_F-->>-FS: OK
        FS-->>-GW: OK

        %% Обогащение метаданными файлов
        GW->>+FS: POST /files/batch<br/>{link_ids}
        FS->>+DB_F: SELECT files JOIN file_links
        DB_F-->>-FS: file metadata
        FS-->>-GW: [{link_id, filename, size, content_type}]

        GW-->>-Клиент: 201 Created {message + file_attachments}
    end

    %% Шаг 3: Real-time уведомление
    rect rgb(255, 245, 230)
        Note right of Клиент: Этап 3: Real-time уведомление
        CS->>+MQ: Publish: message.created<br/>(exchange: chat.events)
        MQ-->>-CS: ACK

        participant WS as WebSocket Service
        MQ->>+WS: Consume: message.created<br/>(queue: websocket.events)
        WS->>+Centrifugo: PublishToUser<br/>(для каждого участника)
        Centrifugo-->>-WS: OK
        Centrifugo-->>Клиент: WebSocket: new_message
    end
```

## 2. Диаграмма последовательности: Пересылка сообщения с файлами

```mermaid
sequenceDiagram
    autonumber
    participant Клиент as Клиент (Browser)
    participant GW as API Gateway
    participant CS as Chat Service
    participant FS as Files Service (gRPC)
    participant DB_C as БД Chat
    participant DB_F as БД Files
    participant MQ as RabbitMQ

    Note over Клиент,MQ: Сценарий: Пересылка сообщения из Чата1 в Чат2

    rect rgb(255, 240, 245)
        Note right of Клиент: Пользователь пересылает сообщение
        Клиент->>+GW: POST /api/chats/messages/{messageId}/forward<br/>{target_chat_id}
        GW->>GW: Проверка JWT + роль != guest
        GW->>+CS: gRPC: ForwardMessage<br/>(message_id, target_chat_id, sender_id)

        %% Проверка прав
        CS->>+DB_C: GetParticipant(target_chat_id, sender_id)
        DB_C-->>-CS: participant (role)
        CS->>CS: Проверка role.CanWrite()

        %% Получение оригинального сообщения
        CS->>+DB_C: GetMessage(message_id)
        DB_C-->>-CS: original_message<br/>{chat_id, content, file_link_ids}

        %% Проверка доступа к исходному чату
        CS->>+DB_C: IsParticipant(original.chat_id, sender_id)
        DB_C-->>-CS: true
    end

    rect rgb(230, 255, 245)
        Note right of CS: Создание новых FileLink для файлов
        loop Для каждого file_link_id
            CS->>+FS: gRPC: GetFileIDByLinkID(link_id)
            FS->>+DB_F: SELECT file_id FROM file_links
            DB_F-->>-FS: file_id
            FS-->>-CS: {file_id}

            CS->>+FS: gRPC: CreateFileLink<br/>(file_id, created_by=sender)
            FS->>+DB_F: INSERT file_links<br/>(new_id, file_id, uploaded_by)
            DB_F-->>-FS: new_link_id
            FS->>+DB_F: INSERT file_link_permissions<br/>(new_link_id, sender_id)
            DB_F-->>-FS: OK
            FS-->>-CS: {new_link_id}
        end
    end

    rect rgb(245, 245, 230)
        Note right of CS: Создание пересланного сообщения
        CS->>+DB_C: INSERT messages<br/>(target_chat_id, content,<br/>forwarded_from_message_id,<br/>forwarded_from_chat_id,<br/>new_file_link_ids)
        DB_C-->>-CS: new_message_id, seq_num

        CS->>+DB_C: INSERT message_file_attachments
        DB_C-->>-CS: OK

        %% Выдача прав участникам целевого чата
        CS->>+DB_C: GetParticipantIDs(target_chat_id)
        DB_C-->>-CS: [user1, user2, user3]

        CS->>+FS: gRPC: GrantPermissions<br/>(new_link_ids, participant_ids)
        FS->>+DB_F: INSERT file_link_permissions<br/>(для каждого участника)
        DB_F-->>-FS: OK
        FS-->>-CS: OK
    end

    rect rgb(255, 245, 230)
        Note right of CS: Уведомление и ответ
        CS->>+MQ: Publish: message.created<br/>(chat_id: target_chat_id)
        MQ-->>-CS: ACK

        CS-->>-GW: ForwardedMessage<br/>{id, forwarded_from_*, new_file_link_ids}

        %% Обогащение файловыми метаданными
        GW->>+FS: POST /files/batch {new_link_ids}
        FS-->>-GW: file_attachments[]

        GW-->>-Клиент: 201 Created<br/>{message + file_attachments}
    end

    Note over Клиент,MQ: Результат: Новые link_id доступны только участникам Чата2<br/>Оригинальные link_id остаются доступны только участникам Чата1
```

## 3. Диаграмма последовательности: Скачивание файла с проверкой прав

```mermaid
sequenceDiagram
    autonumber
    participant Клиент as Клиент (Browser)
    participant GW as API Gateway
    participant FS as Files Service
    participant DB_F as БД Files
    participant Storage as Файловое хранилище

    Note over Клиент,Storage: Сценарий: Скачивание файла по link_id

    Клиент->>+GW: GET /api/files/{link_id}<br/>(Authorization: Bearer JWT)
    GW->>GW: Проверка JWT токена
    GW->>+FS: GET /files/{link_id}<br/>(X-User-ID: user_uuid)

    FS->>+DB_F: SELECT fl.*, f.*<br/>FROM file_links fl<br/>JOIN files f ON fl.file_id = f.id<br/>WHERE fl.id = link_id
    DB_F-->>-FS: file_link + file metadata

    alt Файл не найден
        FS-->>GW: 404 Not Found
        GW-->>Клиент: 404 Not Found
    else Файл найден
        FS->>+DB_F: SELECT * FROM file_link_permissions<br/>WHERE file_link_id = link_id<br/>AND user_id = user_uuid<br/>AND can_download = true
        DB_F-->>-FS: permission record

        alt Нет прав доступа
            FS-->>GW: 403 Forbidden<br/>"access denied"
            GW-->>Клиент: 403 Forbidden
        else Есть права
            FS->>+Storage: Открыть файл<br/>(file_path)
            Storage-->>-FS: FileReader
            FS-->>-GW: Stream + Headers<br/>(Content-Type, Content-Disposition)
            GW-->>-Клиент: 200 OK + бинарные данные
        end
    end
```

## 4. Диаграмма модели данных Files Service

```mermaid
erDiagram
    FILES ||--o{ FILE_LINKS : "имеет"
    FILE_LINKS ||--o{ FILE_LINK_PERMISSIONS : "имеет"
    FILE_LINKS ||--o{ MESSAGE_FILE_ATTACHMENTS : "используется в"
    FILES ||--o{ FILE_SHARE_LINKS : "имеет"
    MESSAGES ||--o{ MESSAGE_FILE_ATTACHMENTS : "содержит"

    FILES {
        uuid id PK "Уникальный ID файла"
        string filename "Имя файла в хранилище"
        string original_filename "Оригинальное имя файла"
        string content_type "MIME тип"
        bigint size "Размер в байтах"
        string file_path "Путь к файлу на диске"
        uuid uploaded_by FK "ID загрузившего пользователя"
        timestamp uploaded_at "Дата загрузки"
        enum status "active | deleted"
        jsonb metadata "Дополнительные метаданные"
    }

    FILE_LINKS {
        uuid id PK "ID ссылки (используется в API)"
        uuid file_id FK "Ссылка на файл"
        uuid uploaded_by FK "Создатель ссылки"
        timestamp uploaded_at "Дата создания"
        boolean is_deleted "Флаг удаления"
    }

    FILE_LINK_PERMISSIONS {
        uuid id PK "ID записи"
        uuid file_link_id FK "ID ссылки на файл"
        uuid user_id FK "ID пользователя"
        boolean can_view "Право просмотра"
        boolean can_download "Право скачивания"
        boolean can_delete "Право удаления"
    }

    FILE_SHARE_LINKS {
        uuid id PK "ID публичной ссылки"
        uuid file_id FK "Ссылка на файл"
        string token UK "Уникальный токен"
        string password "Хеш пароля (опционально)"
        int max_downloads "Лимит скачиваний"
        int download_count "Счётчик скачиваний"
        uuid created_by FK "Создатель ссылки"
        timestamp created_at "Дата создания"
        timestamp expires_at "Срок действия"
        boolean is_active "Активна ли ссылка"
    }

    MESSAGE_FILE_ATTACHMENTS {
        uuid id PK "ID записи"
        uuid message_id FK "ID сообщения (в chat-service)"
        uuid file_link_id FK "ID ссылки на файл"
        int sort_order "Порядок сортировки"
    }

    MESSAGES {
        uuid id PK "ID сообщения"
        uuid chat_id FK "ID чата"
        uuid sender_id FK "ID отправителя"
        text content "Текст сообщения"
        uuid forwarded_from_message_id FK "ID оригинала (при пересылке)"
        uuid forwarded_from_chat_id "ID исходного чата"
    }
```

## 5. Соответствие gRPC и REST API

```mermaid
flowchart TB
    subgraph REST_API["REST API (через API Gateway)"]
        R1["POST /api/files/upload<br/>Загрузка файла"]
        R2["GET /api/files/{linkId}<br/>Скачивание файла"]
        R3["GET /api/files/{linkId}/info<br/>Метаданные файла"]
        R4["DELETE /api/files/{linkId}<br/>Удаление файла"]
        R5["POST /api/files/batch<br/>Пакетное получение метаданных"]
        R6["POST /api/files/grant-permissions<br/>Выдача прав"]
        R7["POST /api/files/{fileId}/share<br/>Создание публичной ссылки"]
        R8["GET /api/files/share/{token}<br/>Скачивание по токену"]
    end

    subgraph GRPC_API["gRPC API (межсервисное)"]
        G1["AddLocalFile<br/>Добавление локального файла"]
        G2["CreateFileLink<br/>Создание новой ссылки"]
        G3["GrantPermissions<br/>Выдача прав пользователям"]
        G4["RevokePermissions<br/>Отзыв прав"]
        G5["GetFileIDByLinkID<br/>Получение file_id по link_id"]
        G6["GetFilesByLinkIDs<br/>Пакетные метаданные"]
    end

    subgraph SERVICE_LAYER["Service Layer"]
        S1["Upload()"]
        S2["Download()"]
        S3["GetFileInfo()"]
        S4["Delete()"]
        S5["GetFilesByLinkIDs()"]
        S6["GrantPermissions()"]
        S7["CreateShareLink()"]
        S8["DownloadByShareToken()"]
        S9["AddLocalFile()"]
        S10["CreateFileLink()"]
        S11["RevokePermissions()"]
        S12["GetFileLinkByID()"]
    end

    %% REST -> Service
    R1 --> S1
    R2 --> S2
    R3 --> S3
    R4 --> S4
    R5 --> S5
    R6 --> S6
    R7 --> S7
    R8 --> S8

    %% gRPC -> Service
    G1 --> S9
    G2 --> S10
    G3 --> S6
    G4 --> S11
    G5 --> S12
    G6 --> S5

    subgraph CONSUMERS["Потребители API"]
        C1["Browser/Клиент<br/>(через API Gateway)"]
        C2["Chat Service<br/>(gRPC клиент)"]
        C3["Другие сервисы<br/>(gRPC клиент)"]
    end

    C1 -.-> REST_API
    C2 -.-> GRPC_API
    C3 -.-> GRPC_API

    style REST_API fill:#e3f2fd
    style GRPC_API fill:#e8f5e9
    style SERVICE_LAYER fill:#fff3e0
    style CONSUMERS fill:#fce4ec
```

## 6. Поток данных при добавлении/удалении участника чата

```mermaid
sequenceDiagram
    autonumber
    participant Клиент as Клиент
    participant GW as API Gateway
    participant CS as Chat Service
    participant FS as Files Service (gRPC)
    participant DB_C as БД Chat
    participant DB_F as БД Files

    Note over Клиент,DB_F: Сценарий A: Добавление участника в чат

    rect rgb(230, 255, 230)
        Клиент->>+GW: POST /api/chats/{chatId}/participants<br/>{user_id}
        GW->>+CS: gRPC: AddParticipant

        CS->>+DB_C: Проверка прав добавляющего
        DB_C-->>-CS: OK (admin/moderator)

        CS->>+DB_C: INSERT chat_participants
        DB_C-->>-CS: participant_id

        %% Выдача прав на все файлы чата
        CS->>+DB_C: GetAllFileLinkIDsForChat(chat_id)
        Note right of DB_C: SELECT DISTINCT file_link_id<br/>FROM message_file_attachments mfa<br/>JOIN messages m ON m.id = mfa.message_id<br/>WHERE m.chat_id = chat_id
        DB_C-->>-CS: [link_id1, link_id2, ...]

        alt Есть файлы в чате
            CS->>+FS: gRPC: GrantPermissions<br/>(link_ids, [new_user_id])
            FS->>+DB_F: INSERT file_link_permissions<br/>(для каждого link_id)
            DB_F-->>-FS: OK
            FS-->>-CS: OK
        end

        CS-->>-GW: Participant
        GW-->>-Клиент: 201 Created
    end

    Note over Клиент,DB_F: Сценарий B: Удаление участника из чата

    rect rgb(255, 230, 230)
        Клиент->>+GW: DELETE /api/chats/{chatId}/participants/{userId}
        GW->>+CS: gRPC: RemoveParticipant

        CS->>+DB_C: Проверка прав удаляющего
        DB_C-->>-CS: OK

        %% Отзыв прав на файлы ПЕРЕД удалением
        CS->>+DB_C: GetAllFileLinkIDsForChat(chat_id)
        DB_C-->>-CS: [link_id1, link_id2, ...]

        alt Есть файлы в чате
            CS->>+FS: gRPC: RevokePermissions<br/>(link_ids, user_id)
            FS->>+DB_F: DELETE FROM file_link_permissions<br/>WHERE file_link_id IN (...)<br/>AND user_id = user_id
            DB_F-->>-FS: OK
            FS-->>-CS: OK
        end

        CS->>+DB_C: DELETE chat_participants<br/>WHERE chat_id AND user_id
        DB_C-->>-CS: OK

        CS-->>-GW: OK
        GW-->>-Клиент: 204 No Content
    end

    Note over Клиент,DB_F: После удаления пользователь теряет доступ<br/>ко всем файлам этого чата
```

## 7. Архитектура компонентов Files Service

```mermaid
flowchart TB
    subgraph External["Внешние клиенты"]
        Browser["Browser"]
        ChatSvc["Chat Service"]
    end

    subgraph FilesService["Files Service"]
        subgraph Handlers["HTTP Handlers"]
            H1["Upload"]
            H2["Download"]
            H3["GetFileInfo"]
            H4["Delete"]
            H5["GetFilesByLinkIDs"]
            H6["GrantPermissions"]
            H7["CreateShareLink"]
            H8["DownloadByShareToken"]
        end

        subgraph GRPCServer["gRPC Server"]
            G1["AddLocalFile"]
            G2["CreateFileLink"]
            G3["GrantPermissions"]
            G4["RevokePermissions"]
            G5["GetFileIDByLinkID"]
            G6["GetFilesByLinkIDs"]
        end

        subgraph ServiceLayer["Service Layer"]
            SVC["FileService Interface"]
        end

        subgraph Repository["Repository Layer"]
            REPO["FileRepository"]
        end

        subgraph Storage["Storage"]
            DISK["Local Disk Storage"]
        end
    end

    subgraph Database["PostgreSQL"]
        T1[("files")]
        T2[("file_links")]
        T3[("file_link_permissions")]
        T4[("file_share_links")]
    end

    Browser --> |"REST API"| Handlers
    ChatSvc --> |"gRPC"| GRPCServer

    Handlers --> SVC
    GRPCServer --> SVC
    SVC --> REPO
    SVC --> DISK

    REPO --> T1
    REPO --> T2
    REPO --> T3
    REPO --> T4

    style FilesService fill:#f5f5f5
    style Handlers fill:#e3f2fd
    style GRPCServer fill:#e8f5e9
    style ServiceLayer fill:#fff3e0
    style Repository fill:#fce4ec
```

## Легенда

| Цвет | Значение |
|------|----------|
| 🔵 Синий | REST API / HTTP взаимодействие |
| 🟢 Зелёный | gRPC взаимодействие |
| 🟡 Жёлтый | Service Layer / Бизнес-логика |
| 🔴 Красный | Repository / Данные |
| 🟣 Фиолетовый | Real-time / События |
