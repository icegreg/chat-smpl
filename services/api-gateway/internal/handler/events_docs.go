package handler

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// EventDefinition describes a WebSocket event
type EventDefinition struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Channel     string                 `json:"channel"`
	Payload     map[string]FieldSchema `json:"payload"`
}

// FieldSchema describes a field in the event payload
type FieldSchema struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Example     any    `json:"example,omitempty"`
}

// EventsDocumentation contains all event definitions
type EventsDocumentation struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Channel     ChannelInfo       `json:"channel"`
	BaseEvent   BaseEventSchema   `json:"base_event"`
	Events      []EventDefinition `json:"events"`
}

// ChannelInfo describes the WebSocket channel
type ChannelInfo struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// BaseEventSchema describes the common event wrapper
type BaseEventSchema struct {
	Description string                 `json:"description"`
	Fields      map[string]FieldSchema `json:"fields"`
}

// EventRegistry contains all WebSocket events
var EventRegistry = EventsDocumentation{
	Title:       "Chat WebSocket Events API",
	Description: "Документация по WebSocket событиям, отправляемым через Centrifugo",
	Version:     "1.0.0",
	Channel: ChannelInfo{
		Pattern:     "user:{userId}",
		Description: "Персональный канал пользователя. Все события доставляются в этот канал.",
		Example:     "user:550e8400-e29b-41d4-a716-446655440000",
	},
	BaseEvent: BaseEventSchema{
		Description: "Все события имеют общую структуру-обёртку",
		Fields: map[string]FieldSchema{
			"type": {
				Type:        "string",
				Description: "Тип события",
				Required:    true,
				Example:     "message.created",
			},
			"timestamp": {
				Type:        "string (ISO 8601)",
				Description: "Время создания события",
				Required:    true,
				Example:     "2024-01-15T10:30:00Z",
			},
			"actor_id": {
				Type:        "string (UUID)",
				Description: "ID пользователя, инициировавшего событие",
				Required:    true,
				Example:     "550e8400-e29b-41d4-a716-446655440000",
			},
			"chat_id": {
				Type:        "string (UUID)",
				Description: "ID чата, к которому относится событие",
				Required:    true,
				Example:     "660e8400-e29b-41d4-a716-446655440001",
			},
			"data": {
				Type:        "object",
				Description: "Полезная нагрузка события (зависит от типа)",
				Required:    true,
			},
		},
	},
	Events: []EventDefinition{
		// Chat events
		{
			Type:        "chat.created",
			Description: "Создан новый чат",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":         {Type: "string (UUID)", Description: "ID чата", Required: true},
				"name":       {Type: "string", Description: "Название чата", Required: true},
				"chat_type":  {Type: "string", Description: "Тип чата: direct, group, channel", Required: true},
				"created_by": {Type: "string (UUID)", Description: "ID создателя", Required: true},
			},
		},
		{
			Type:        "chat.updated",
			Description: "Чат обновлён (изменено название или настройки)",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":         {Type: "string (UUID)", Description: "ID чата", Required: true},
				"name":       {Type: "string", Description: "Новое название чата", Required: true},
				"chat_type":  {Type: "string", Description: "Тип чата", Required: true},
				"created_by": {Type: "string (UUID)", Description: "ID создателя", Required: true},
			},
		},
		{
			Type:        "chat.deleted",
			Description: "Чат удалён",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"chat_id": {Type: "string (UUID)", Description: "ID удалённого чата", Required: true},
			},
		},

		// Message events
		{
			Type:        "message.created",
			Description: "Новое сообщение в чате",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":                  {Type: "string (UUID)", Description: "ID сообщения", Required: true},
				"chat_id":             {Type: "string (UUID)", Description: "ID чата", Required: true},
				"sender_id":           {Type: "string (UUID)", Description: "ID отправителя", Required: true},
				"content":             {Type: "string", Description: "Текст сообщения", Required: true},
				"sent_at":             {Type: "string (ISO 8601)", Description: "Время отправки", Required: true},
				"parent_id":           {Type: "string (UUID)", Description: "ID родительского сообщения (для тредов)", Required: false},
				"sender_username":     {Type: "string", Description: "Username отправителя", Required: false},
				"sender_display_name": {Type: "string", Description: "Отображаемое имя отправителя", Required: false},
				"sender_avatar_url":   {Type: "string", Description: "URL аватара отправителя", Required: false},
				"file_link_ids":       {Type: "array[string]", Description: "ID прикреплённых файлов", Required: false},
			},
		},
		{
			Type:        "message.updated",
			Description: "Сообщение отредактировано",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":         {Type: "string (UUID)", Description: "ID сообщения", Required: true},
				"chat_id":    {Type: "string (UUID)", Description: "ID чата", Required: true},
				"sender_id":  {Type: "string (UUID)", Description: "ID отправителя", Required: true},
				"content":    {Type: "string", Description: "Новый текст сообщения", Required: true},
				"sent_at":    {Type: "string (ISO 8601)", Description: "Время отправки", Required: true},
				"updated_at": {Type: "string (ISO 8601)", Description: "Время редактирования", Required: true},
			},
		},
		{
			Type:        "message.deleted",
			Description: "Сообщение удалено (soft delete)",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"message_id":            {Type: "string (UUID)", Description: "ID удалённого сообщения", Required: true},
				"chat_id":               {Type: "string (UUID)", Description: "ID чата", Required: true},
				"is_moderated_deletion": {Type: "boolean", Description: "true если удалено модератором", Required: true},
			},
		},
		{
			Type:        "message.restored",
			Description: "Сообщение восстановлено после удаления",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"message_id": {Type: "string (UUID)", Description: "ID восстановленного сообщения", Required: true},
				"chat_id":    {Type: "string (UUID)", Description: "ID чата", Required: true},
				"sender_id":  {Type: "string (UUID)", Description: "ID отправителя", Required: true},
				"content":    {Type: "string", Description: "Текст сообщения", Required: true},
			},
		},

		// Typing indicator
		{
			Type:        "typing",
			Description: "Индикатор набора текста",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"user_id":   {Type: "string (UUID)", Description: "ID пользователя, который печатает", Required: true},
				"is_typing": {Type: "boolean", Description: "true - начал печатать, false - закончил", Required: true},
			},
		},

		// Reaction events
		{
			Type:        "reaction.added",
			Description: "Добавлена реакция на сообщение",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"message_id": {Type: "string (UUID)", Description: "ID сообщения", Required: true},
				"emoji":      {Type: "string", Description: "Эмодзи реакции", Required: true, Example: "👍"},
				"user_id":    {Type: "string (UUID)", Description: "ID пользователя", Required: true},
			},
		},
		{
			Type:        "reaction.removed",
			Description: "Удалена реакция с сообщения",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"message_id": {Type: "string (UUID)", Description: "ID сообщения", Required: true},
				"emoji":      {Type: "string", Description: "Эмодзи реакции", Required: true},
				"user_id":    {Type: "string (UUID)", Description: "ID пользователя", Required: true},
			},
		},

		// Thread events
		{
			Type:        "thread.created",
			Description: "Создан новый тред",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":                     {Type: "string (UUID)", Description: "ID треда", Required: true},
				"chat_id":                {Type: "string (UUID)", Description: "ID чата", Required: true},
				"parent_message_id":      {Type: "string (UUID)", Description: "ID родительского сообщения", Required: false},
				"thread_type":            {Type: "string", Description: "Тип треда: reply, topic", Required: true},
				"title":                  {Type: "string", Description: "Заголовок треда", Required: false},
				"message_count":          {Type: "integer", Description: "Количество сообщений", Required: true},
				"created_by":             {Type: "string (UUID)", Description: "ID создателя", Required: false},
				"is_archived":            {Type: "boolean", Description: "Архивирован ли тред", Required: true},
				"restricted_participants": {Type: "boolean", Description: "Ограничен ли список участников", Required: true},
			},
		},
		{
			Type:        "thread.archived",
			Description: "Тред архивирован",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"id":                     {Type: "string (UUID)", Description: "ID треда", Required: true},
				"chat_id":                {Type: "string (UUID)", Description: "ID чата", Required: true},
				"parent_message_id":      {Type: "string (UUID)", Description: "ID родительского сообщения", Required: false},
				"thread_type":            {Type: "string", Description: "Тип треда", Required: true},
				"title":                  {Type: "string", Description: "Заголовок треда", Required: false},
				"message_count":          {Type: "integer", Description: "Количество сообщений", Required: true},
				"created_by":             {Type: "string (UUID)", Description: "ID создателя", Required: false},
				"is_archived":            {Type: "boolean", Description: "Всегда true", Required: true},
				"restricted_participants": {Type: "boolean", Description: "Ограничен ли список участников", Required: true},
			},
		},

		// Voice events
		{
			Type:        "conference.created",
			Description: "Создана голосовая конференция",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"conference_id": {Type: "string (UUID)", Description: "ID конференции", Required: true},
				"chat_id":       {Type: "string (UUID)", Description: "ID чата", Required: true},
				"name":          {Type: "string", Description: "Название конференции", Required: true},
				"created_by":    {Type: "string (UUID)", Description: "ID создателя", Required: true},
			},
		},
		{
			Type:        "conference.ended",
			Description: "Голосовая конференция завершена",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"conference_id": {Type: "string (UUID)", Description: "ID конференции", Required: true},
				"chat_id":       {Type: "string (UUID)", Description: "ID чата", Required: true},
			},
		},
		{
			Type:        "participant.joined",
			Description: "Участник присоединился к конференции",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"conference_id": {Type: "string (UUID)", Description: "ID конференции", Required: true},
				"user_id":       {Type: "string (UUID)", Description: "ID пользователя", Required: true},
				"username":      {Type: "string", Description: "Username пользователя", Required: false},
			},
		},
		{
			Type:        "participant.left",
			Description: "Участник покинул конференцию",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"conference_id": {Type: "string (UUID)", Description: "ID конференции", Required: true},
				"user_id":       {Type: "string (UUID)", Description: "ID пользователя", Required: true},
			},
		},
		{
			Type:        "call.initiated",
			Description: "Инициирован звонок",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"call_id":   {Type: "string (UUID)", Description: "ID звонка", Required: true},
				"caller_id": {Type: "string (UUID)", Description: "ID звонящего", Required: true},
				"callee_id": {Type: "string (UUID)", Description: "ID вызываемого", Required: true},
			},
		},
		{
			Type:        "call.answered",
			Description: "Звонок принят",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"call_id": {Type: "string (UUID)", Description: "ID звонка", Required: true},
			},
		},
		{
			Type:        "call.ended",
			Description: "Звонок завершён",
			Channel:     "user:{userId}",
			Payload: map[string]FieldSchema{
				"call_id": {Type: "string (UUID)", Description: "ID звонка", Required: true},
				"reason":  {Type: "string", Description: "Причина завершения", Required: false},
			},
		},
	},
}

// EventsDocsHandler handles event documentation requests
type EventsDocsHandler struct{}

// NewEventsDocsHandler creates a new handler
func NewEventsDocsHandler() *EventsDocsHandler {
	return &EventsDocsHandler{}
}

// GetEventsJSON returns events documentation as JSON
// @Summary Get WebSocket events documentation
// @Description Returns JSON schema of all WebSocket events
// @Tags documentation
// @Produce json
// @Success 200 {object} EventsDocumentation
// @Router /docs/events [get]
func (h *EventsDocsHandler) GetEventsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EventRegistry)
}

// GetEventsHTML returns events documentation as HTML page
// @Summary Get WebSocket events documentation (HTML)
// @Description Returns HTML page with WebSocket events documentation
// @Tags documentation
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /docs/events.html [get]
func (h *EventsDocsHandler) GetEventsHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(eventsHTMLTemplate))
}

// Helper to get type name from reflect
func getTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		return "array[" + getTypeName(t.Elem()) + "]"
	case reflect.Ptr:
		return getTypeName(t.Elem()) + " (optional)"
	default:
		return "object"
	}
}

const eventsHTMLTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebSocket Events API</title>
    <style>
        :root {
            --bg-primary: #1a1a2e;
            --bg-secondary: #16213e;
            --bg-card: #0f3460;
            --text-primary: #eaeaea;
            --text-secondary: #a0a0a0;
            --accent: #e94560;
            --accent-hover: #ff6b6b;
            --border: #2a2a4a;
            --code-bg: #0d1117;
            --success: #4ade80;
            --warning: #fbbf24;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }

        header {
            text-align: center;
            margin-bottom: 3rem;
            padding-bottom: 2rem;
            border-bottom: 1px solid var(--border);
        }

        h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
            background: linear-gradient(135deg, var(--accent), var(--accent-hover));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .version {
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .section {
            margin-bottom: 2rem;
        }

        .section-title {
            font-size: 1.5rem;
            margin-bottom: 1rem;
            color: var(--accent);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .card {
            background: var(--bg-secondary);
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 1rem;
            border: 1px solid var(--border);
        }

        .channel-info {
            display: grid;
            grid-template-columns: 1fr 2fr;
            gap: 1rem;
        }

        .channel-info dt {
            color: var(--text-secondary);
        }

        .channel-info dd {
            font-family: 'Monaco', 'Menlo', monospace;
        }

        .base-event code {
            background: var(--code-bg);
            padding: 1rem;
            border-radius: 8px;
            display: block;
            overflow-x: auto;
            font-size: 0.85rem;
        }

        .events-grid {
            display: grid;
            gap: 1rem;
        }

        .event-card {
            background: var(--bg-secondary);
            border-radius: 12px;
            border: 1px solid var(--border);
            overflow: hidden;
            transition: transform 0.2s, box-shadow 0.2s;
        }

        .event-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 20px rgba(233, 69, 96, 0.2);
        }

        .event-header {
            background: var(--bg-card);
            padding: 1rem 1.5rem;
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .event-type {
            font-family: 'Monaco', 'Menlo', monospace;
            font-weight: 600;
            color: var(--accent);
        }

        .event-description {
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .event-body {
            padding: 1.5rem;
            display: none;
        }

        .event-card.open .event-body {
            display: block;
        }

        .event-card.open .toggle-icon {
            transform: rotate(180deg);
        }

        .toggle-icon {
            transition: transform 0.2s;
            color: var(--text-secondary);
        }

        .payload-table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.9rem;
        }

        .payload-table th,
        .payload-table td {
            padding: 0.75rem;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }

        .payload-table th {
            color: var(--text-secondary);
            font-weight: 500;
        }

        .field-name {
            font-family: 'Monaco', 'Menlo', monospace;
            color: var(--success);
        }

        .field-type {
            color: var(--warning);
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.8rem;
        }

        .required {
            color: var(--accent);
            font-size: 0.75rem;
            margin-left: 0.25rem;
        }

        .category {
            margin-top: 2rem;
        }

        .category-title {
            font-size: 1.2rem;
            color: var(--text-secondary);
            margin-bottom: 1rem;
            padding-left: 0.5rem;
            border-left: 3px solid var(--accent);
        }

        .json-link {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            color: var(--accent);
            text-decoration: none;
            padding: 0.5rem 1rem;
            border: 1px solid var(--accent);
            border-radius: 6px;
            transition: all 0.2s;
        }

        .json-link:hover {
            background: var(--accent);
            color: white;
        }

        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }

            .channel-info {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>WebSocket Events API</h1>
            <p class="version">Version 1.0.0</p>
            <p style="margin-top: 1rem; color: var(--text-secondary);">
                Документация по WebSocket событиям, отправляемым через Centrifugo
            </p>
            <a href="/api/docs/events" class="json-link" style="margin-top: 1rem; display: inline-flex;">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M5 3h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2zm0 2v14h14V5H5zm2 2h10v2H7V7zm0 4h10v2H7v-2zm0 4h7v2H7v-2z"/>
                </svg>
                JSON Schema
            </a>
        </header>

        <section class="section">
            <h2 class="section-title">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
                </svg>
                Канал подключения
            </h2>
            <div class="card">
                <dl class="channel-info">
                    <dt>Паттерн:</dt>
                    <dd>user:{userId}</dd>
                    <dt>Пример:</dt>
                    <dd>user:550e8400-e29b-41d4-a716-446655440000</dd>
                    <dt>Описание:</dt>
                    <dd>Персональный канал пользователя. Все события доставляются в этот канал.</dd>
                </dl>
            </div>
        </section>

        <section class="section">
            <h2 class="section-title">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm4 18H6V4h7v5h5v11z"/>
                </svg>
                Базовая структура события
            </h2>
            <div class="card base-event">
                <p style="margin-bottom: 1rem; color: var(--text-secondary);">
                    Все события имеют общую структуру-обёртку:
                </p>
                <code><pre>{
  "type": "message.created",      // Тип события
  "timestamp": "2024-01-15T10:30:00Z",  // Время создания
  "actor_id": "uuid",             // ID инициатора
  "chat_id": "uuid",              // ID чата
  "data": { ... }                 // Полезная нагрузка
}</pre></code>
            </div>
        </section>

        <section class="section">
            <h2 class="section-title">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/>
                </svg>
                События
            </h2>

            <div class="category">
                <h3 class="category-title">Чаты</h3>
                <div class="events-grid" id="chat-events"></div>
            </div>

            <div class="category">
                <h3 class="category-title">Сообщения</h3>
                <div class="events-grid" id="message-events"></div>
            </div>

            <div class="category">
                <h3 class="category-title">Реакции</h3>
                <div class="events-grid" id="reaction-events"></div>
            </div>

            <div class="category">
                <h3 class="category-title">Треды</h3>
                <div class="events-grid" id="thread-events"></div>
            </div>

            <div class="category">
                <h3 class="category-title">Голосовые вызовы</h3>
                <div class="events-grid" id="voice-events"></div>
            </div>

            <div class="category">
                <h3 class="category-title">Прочее</h3>
                <div class="events-grid" id="other-events"></div>
            </div>
        </section>

        <section class="section">
            <h2 class="section-title">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z"/>
                </svg>
                Пример использования (k6)
            </h2>
            <div class="card">
                <p style="margin-bottom: 1rem; color: var(--text-secondary);">
                    Пример нагрузочного тестирования WebSocket с валидацией событий по документации:
                </p>
                <code><pre style="max-height: 500px; overflow-y: auto;">import ws from 'k6/ws';
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8888';
const WS_URL = BASE_URL.replace('http', 'ws') + '/connection/websocket';

// Загружаем документацию событий
const eventsDoc = JSON.parse(
  http.get(` + "`" + `${BASE_URL}/api/docs/events` + "`" + `).body
);

// Set допустимых типов событий
const validEventTypes = new Set(eventsDoc.events.map(e => e.type));

// Валидация события по документации
function validateEvent(event) {
  const eventDef = eventsDoc.events.find(e => e.type === event.type);
  if (!eventDef) return false;

  for (const [field, schema] of Object.entries(eventDef.payload)) {
    if (schema.required && !(field in event.data)) {
      return false;
    }
  }
  return true;
}

export default function () {
  // 1. Логин
  const loginRes = http.post(` + "`" + `${BASE_URL}/api/auth/login` + "`" + `, JSON.stringify({
    email: 'test@example.com',
    password: 'password123',
  }), { headers: { 'Content-Type': 'application/json' } });

  const { access_token, user } = JSON.parse(loginRes.body);

  // 2. Токен Centrifugo
  const tokenRes = http.post(` + "`" + `${BASE_URL}/api/centrifugo/connection_token` + "`" + `, null, {
    headers: { 'Authorization': ` + "`" + `Bearer ${access_token}` + "`" + ` },
  });
  const { token } = JSON.parse(tokenRes.body);

  // 3. WebSocket подключение
  ws.connect(WS_URL, {}, function (socket) {
    socket.on('open', () => {
      socket.send(JSON.stringify({
        id: 1,
        connect: { token, name: 'k6-test' },
      }));
    });

    socket.on('message', (msg) => {
      const data = JSON.parse(msg);

      if (data.id === 1 && data.connect) {
        socket.send(JSON.stringify({
          id: 2,
          subscribe: { channel: ` + "`" + `user:${user.id}` + "`" + ` },
        }));
      }

      if (data.push?.pub?.data) {
        const event = data.push.pub.data;

        check(event, {
          'event type documented': (e) => validEventTypes.has(e.type),
          'event structure valid': () => validateEvent(event),
        });
      }
    });

    socket.setTimeout(() => socket.close(), 30000);
  });
}</pre></code>
                <p style="margin-top: 1rem; color: var(--text-secondary);">
                    <strong>Запуск:</strong> <code style="background: var(--code-bg); padding: 0.25rem 0.5rem; border-radius: 4px;">k6 run --env BASE_URL=http://localhost:8888 ws-test.js</code>
                </p>
            </div>
        </section>
    </div>

    <script>
        // Fetch events from API and render
        fetch('/api/docs/events')
            .then(r => r.json())
            .then(data => {
                const categories = {
                    'chat-events': ['chat.created', 'chat.updated', 'chat.deleted'],
                    'message-events': ['message.created', 'message.updated', 'message.deleted', 'message.restored'],
                    'reaction-events': ['reaction.added', 'reaction.removed'],
                    'thread-events': ['thread.created', 'thread.archived'],
                    'voice-events': ['conference.created', 'conference.ended', 'participant.joined', 'participant.left', 'call.initiated', 'call.answered', 'call.ended'],
                    'other-events': ['typing']
                };

                data.events.forEach(event => {
                    let container = document.getElementById('other-events');
                    for (const [catId, types] of Object.entries(categories)) {
                        if (types.includes(event.type)) {
                            container = document.getElementById(catId);
                            break;
                        }
                    }

                    const card = document.createElement('div');
                    card.className = 'event-card';
                    card.innerHTML = ` + "`" + `
                        <div class="event-header" onclick="this.parentElement.classList.toggle('open')">
                            <div>
                                <div class="event-type">${event.type}</div>
                                <div class="event-description">${event.description}</div>
                            </div>
                            <span class="toggle-icon">▼</span>
                        </div>
                        <div class="event-body">
                            <table class="payload-table">
                                <thead>
                                    <tr>
                                        <th>Поле</th>
                                        <th>Тип</th>
                                        <th>Описание</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${Object.entries(event.payload).map(([name, field]) => ` + "`" + `
                                        <tr>
                                            <td>
                                                <span class="field-name">${name}</span>
                                                ${field.required ? '<span class="required">*</span>' : ''}
                                            </td>
                                            <td><span class="field-type">${field.type}</span></td>
                                            <td>${field.description || ''}</td>
                                        </tr>
                                    ` + "`" + `).join('')}
                                </tbody>
                            </table>
                        </div>
                    ` + "`" + `;
                    container.appendChild(card);
                });
            });
    </script>
</body>
</html>
`
