package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// EventsHandler handles WebSocket events documentation
type EventsHandler struct{}

func NewEventsHandler() *EventsHandler {
	return &EventsHandler{}
}

func (h *EventsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetEventsHTML)
	r.Get("/schema", h.GetEventsSchema)
	return r
}

// EventField describes a field in event payload
type EventField struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Description string       `json:"description"`
	Required    bool         `json:"required"`
	Example     interface{}  `json:"example,omitempty"`
	Enum        []string     `json:"enum,omitempty"`
	Fields      []EventField `json:"fields,omitempty"` // For nested objects
}

// EventSchema describes a single event type
type EventSchema struct {
	Type        string       `json:"type"`
	Exchange    string       `json:"exchange"`
	RoutingKey  string       `json:"routing_key"`
	Description string       `json:"description"`
	Channel     string       `json:"channel"`
	Payload     []EventField `json:"payload"`
	Example     interface{}  `json:"example,omitempty"`
}

// EventsSchemaResponse contains all event schemas
type EventsSchemaResponse struct {
	Version     string                  `json:"version"`
	Description string                  `json:"description"`
	BaseEvent   []EventField            `json:"base_event"`
	Events      map[string][]EventSchema `json:"events"`
}

// GetEventsSchema godoc
// @Summary Get WebSocket events schema
// @Description Returns the schema of all WebSocket events delivered via Centrifugo
// @Tags events
// @Produce json
// @Success 200 {object} EventsSchemaResponse "Events schema"
// @Router /events/schema [get]
func (h *EventsHandler) GetEventsSchema(w http.ResponseWriter, r *http.Request) {
	schema := buildEventsSchema()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schema)
}

// GetEventsHTML godoc
// @Summary Get WebSocket events documentation page
// @Description Returns HTML documentation page for WebSocket events
// @Tags events
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /events [get]
func (h *EventsHandler) GetEventsHTML(w http.ResponseWriter, r *http.Request) {
	schema := buildEventsSchema()
	html := generateEventsHTML(schema)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func generateEventsHTML(schema EventsSchemaResponse) string {
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebSocket Events API</title>
    <style>
        :root {
            --primary: #2563eb;
            --primary-dark: #1d4ed8;
            --bg: #f8fafc;
            --card-bg: #ffffff;
            --text: #1e293b;
            --text-muted: #64748b;
            --border: #e2e8f0;
            --code-bg: #f1f5f9;
            --success: #22c55e;
            --warning: #f59e0b;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--bg);
            color: var(--text);
            line-height: 1.6;
        }
        .container { max-width: 1200px; margin: 0 auto; padding: 2rem; }
        header {
            background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%);
            color: white;
            padding: 3rem 2rem;
            margin-bottom: 2rem;
        }
        header h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
        header p { opacity: 0.9; font-size: 1.1rem; }
        .version {
            display: inline-block;
            background: rgba(255,255,255,0.2);
            padding: 0.25rem 0.75rem;
            border-radius: 1rem;
            font-size: 0.875rem;
            margin-top: 1rem;
        }
        .nav {
            background: var(--card-bg);
            border-radius: 0.5rem;
            padding: 1rem;
            margin-bottom: 2rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            position: sticky;
            top: 1rem;
            z-index: 100;
        }
        .nav-title { font-weight: 600; margin-bottom: 0.5rem; color: var(--text-muted); font-size: 0.875rem; }
        .nav-links { display: flex; flex-wrap: wrap; gap: 0.5rem; }
        .nav-links a {
            color: var(--primary);
            text-decoration: none;
            padding: 0.375rem 0.75rem;
            border-radius: 0.375rem;
            background: var(--code-bg);
            font-size: 0.875rem;
            transition: all 0.2s;
        }
        .nav-links a:hover { background: var(--primary); color: white; }
        .section { margin-bottom: 3rem; }
        .section-title {
            font-size: 1.5rem;
            color: var(--primary);
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 2px solid var(--primary);
        }
        .base-event {
            background: var(--card-bg);
            border-radius: 0.5rem;
            padding: 1.5rem;
            margin-bottom: 2rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
        }
        .base-event h3 { margin-bottom: 1rem; }
        .event-card {
            background: var(--card-bg);
            border-radius: 0.5rem;
            margin-bottom: 1rem;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .event-header {
            padding: 1rem 1.5rem;
            background: var(--code-bg);
            cursor: pointer;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .event-header:hover { background: #e2e8f0; }
        .event-type {
            font-family: 'Monaco', 'Menlo', monospace;
            font-weight: 600;
            color: var(--primary);
        }
        .event-meta {
            display: flex;
            gap: 1rem;
            font-size: 0.75rem;
            color: var(--text-muted);
        }
        .event-meta span {
            background: var(--card-bg);
            padding: 0.125rem 0.5rem;
            border-radius: 0.25rem;
        }
        .event-body { padding: 1.5rem; display: none; }
        .event-card.open .event-body { display: block; }
        .event-card.open .event-header { border-bottom: 1px solid var(--border); }
        .event-description {
            color: var(--text-muted);
            margin-bottom: 1rem;
            font-size: 0.95rem;
        }
        .payload-title {
            font-weight: 600;
            margin: 1rem 0 0.5rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        .payload-title::before {
            content: '';
            display: block;
            width: 4px;
            height: 1rem;
            background: var(--primary);
            border-radius: 2px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            font-size: 0.875rem;
        }
        th, td {
            text-align: left;
            padding: 0.75rem;
            border-bottom: 1px solid var(--border);
        }
        th {
            background: var(--code-bg);
            font-weight: 600;
            color: var(--text-muted);
        }
        .field-name {
            font-family: 'Monaco', 'Menlo', monospace;
            color: var(--primary);
        }
        .field-type {
            font-family: 'Monaco', 'Menlo', monospace;
            color: var(--text-muted);
            font-size: 0.8rem;
        }
        .required {
            color: var(--success);
            font-size: 0.75rem;
            font-weight: 600;
        }
        .optional {
            color: var(--text-muted);
            font-size: 0.75rem;
        }
        .enum-values {
            display: flex;
            flex-wrap: wrap;
            gap: 0.25rem;
            margin-top: 0.25rem;
        }
        .enum-value {
            background: var(--code-bg);
            padding: 0.125rem 0.375rem;
            border-radius: 0.25rem;
            font-family: monospace;
            font-size: 0.75rem;
        }
        .example-block {
            margin-top: 1rem;
            background: #1e293b;
            border-radius: 0.5rem;
            overflow: hidden;
        }
        .example-title {
            background: #334155;
            color: #94a3b8;
            padding: 0.5rem 1rem;
            font-size: 0.75rem;
            font-weight: 600;
        }
        .example-code {
            padding: 1rem;
            color: #e2e8f0;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.8rem;
            overflow-x: auto;
            white-space: pre;
        }
        .toggle-icon {
            transition: transform 0.2s;
            color: var(--text-muted);
        }
        .event-card.open .toggle-icon { transform: rotate(180deg); }
        .json-key { color: #7dd3fc; }
        .json-string { color: #86efac; }
        .json-number { color: #fde047; }
        .json-bool { color: #f472b6; }
        @media (max-width: 768px) {
            .container { padding: 1rem; }
            header { padding: 2rem 1rem; }
            header h1 { font-size: 1.75rem; }
            .event-meta { display: none; }
        }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>WebSocket Events API</h1>
            <p>` + schema.Description + `</p>
            <span class="version">v` + schema.Version + `</span>
        </div>
    </header>

    <div class="container">
        <nav class="nav">
            <div class="nav-title">Категории событий</div>
            <div class="nav-links">`

	// Add navigation links
	categories := []string{"chat", "message", "typing", "reaction", "thread", "conference", "participant", "call"}
	for _, cat := range categories {
		if events, ok := schema.Events[cat]; ok && len(events) > 0 {
			html += `<a href="#` + cat + `">` + cat + ` (` + itoa(len(events)) + `)</a>`
		}
	}

	html += `
            </div>
        </nav>

        <section class="base-event">
            <h3>Базовая структура события</h3>
            <p style="color: var(--text-muted); margin-bottom: 1rem;">Все события имеют общую обёртку:</p>
            <table>
                <thead>
                    <tr><th>Поле</th><th>Тип</th><th>Описание</th></tr>
                </thead>
                <tbody>`

	for _, field := range schema.BaseEvent {
		html += `<tr>
            <td class="field-name">` + field.Name + `</td>
            <td class="field-type">` + field.Type + `</td>
            <td>` + field.Description + `</td>
        </tr>`
	}

	html += `
                </tbody>
            </table>
        </section>`

	// Generate sections for each category
	for _, cat := range categories {
		events, ok := schema.Events[cat]
		if !ok || len(events) == 0 {
			continue
		}

		html += `
        <section class="section" id="` + cat + `">
            <h2 class="section-title">` + getCategoryTitle(cat) + `</h2>`

		for _, event := range events {
			html += generateEventCard(event)
		}

		html += `</section>`
	}

	html += `
    </div>

    <script>
        document.querySelectorAll('.event-header').forEach(header => {
            header.addEventListener('click', () => {
                header.parentElement.classList.toggle('open');
            });
        });
    </script>
</body>
</html>`

	return html
}

func generateEventCard(event EventSchema) string {
	html := `
            <div class="event-card">
                <div class="event-header">
                    <span class="event-type">` + event.Type + `</span>
                    <div class="event-meta">
                        <span>` + event.Exchange + `</span>
                        <span>` + event.Channel + `</span>
                    </div>
                    <svg class="toggle-icon" width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"/>
                    </svg>
                </div>
                <div class="event-body">
                    <p class="event-description">` + event.Description + `</p>
                    <div class="payload-title">Payload (data)</div>
                    <table>
                        <thead>
                            <tr><th>Поле</th><th>Тип</th><th></th><th>Описание</th></tr>
                        </thead>
                        <tbody>`

	for _, field := range event.Payload {
		reqClass := "optional"
		reqText := "optional"
		if field.Required {
			reqClass = "required"
			reqText = "required"
		}

		html += `<tr>
                <td class="field-name">` + field.Name + `</td>
                <td class="field-type">` + field.Type + `</td>
                <td><span class="` + reqClass + `">` + reqText + `</span></td>
                <td>` + field.Description

		if len(field.Enum) > 0 {
			html += `<div class="enum-values">`
			for _, v := range field.Enum {
				html += `<span class="enum-value">` + v + `</span>`
			}
			html += `</div>`
		}

		html += `</td></tr>`
	}

	html += `
                        </tbody>
                    </table>`

	if event.Example != nil {
		exampleJSON, _ := json.MarshalIndent(event.Example, "", "  ")
		html += `
                    <div class="example-block">
                        <div class="example-title">Пример</div>
                        <div class="example-code">` + syntaxHighlightJSON(string(exampleJSON)) + `</div>
                    </div>`
	}

	html += `
                </div>
            </div>`

	return html
}

func getCategoryTitle(cat string) string {
	titles := map[string]string{
		"chat":        "Chat Events",
		"message":     "Message Events",
		"typing":      "Typing Events",
		"reaction":    "Reaction Events",
		"thread":      "Thread Events",
		"conference":  "Conference Events",
		"participant": "Participant Events",
		"call":        "Call Events",
	}
	if title, ok := titles[cat]; ok {
		return title
	}
	return cat
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func syntaxHighlightJSON(jsonStr string) string {
	// Simple syntax highlighting
	result := ""
	inString := false
	for i := 0; i < len(jsonStr); i++ {
		c := jsonStr[i]
		if c == '"' && (i == 0 || jsonStr[i-1] != '\\') {
			if !inString {
				// Check if this is a key (followed by :)
				endQuote := -1
				for j := i + 1; j < len(jsonStr); j++ {
					if jsonStr[j] == '"' && jsonStr[j-1] != '\\' {
						endQuote = j
						break
					}
				}
				if endQuote != -1 {
					// Look for colon after the closing quote
					isKey := false
					for j := endQuote + 1; j < len(jsonStr); j++ {
						if jsonStr[j] == ' ' || jsonStr[j] == '\n' || jsonStr[j] == '\t' {
							continue
						}
						if jsonStr[j] == ':' {
							isKey = true
						}
						break
					}
					if isKey {
						result += `<span class="json-key">`
					} else {
						result += `<span class="json-string">`
					}
				}
				inString = true
			} else {
				result += string(c) + `</span>`
				inString = false
				continue
			}
		}
		result += string(c)
	}
	return result
}

func buildEventsSchema() EventsSchemaResponse {
	return EventsSchemaResponse{
		Version:     "1.0.0",
		Description: "WebSocket события, доставляемые клиентам через Centrifugo. События публикуются в персональные каналы пользователей (user:{userId}).",
		BaseEvent: []EventField{
			{Name: "type", Type: "string", Description: "Тип события", Required: true, Example: "message.created"},
			{Name: "timestamp", Type: "string (RFC3339)", Description: "Время события", Required: true, Example: "2024-01-15T10:30:00Z"},
			{Name: "actor_id", Type: "string (uuid)", Description: "ID пользователя, инициировавшего событие", Required: true},
			{Name: "chat_id", Type: "string (uuid)", Description: "ID чата", Required: true},
			{Name: "data", Type: "object", Description: "Payload события (зависит от типа)", Required: true},
		},
		Events: map[string][]EventSchema{
			"chat": buildChatEvents(),
			"message": buildMessageEvents(),
			"typing": buildTypingEvents(),
			"reaction": buildReactionEvents(),
			"thread": buildThreadEvents(),
			"conference": buildConferenceEvents(),
			"participant": buildParticipantEvents(),
			"call": buildCallEvents(),
		},
	}
}

func buildChatEvents() []EventSchema {
	chatDataFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID чата", Required: true},
		{Name: "name", Type: "string", Description: "Название чата", Required: true},
		{Name: "chat_type", Type: "string", Description: "Тип чата", Required: true, Enum: []string{"direct", "group", "channel"}},
		{Name: "created_by", Type: "string (uuid)", Description: "ID создателя", Required: true},
	}

	return []EventSchema{
		{
			Type:        "chat.created",
			Exchange:    "chat.events",
			RoutingKey:  "chat.created",
			Description: "Создан новый чат. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     chatDataFields,
			Example: map[string]interface{}{
				"type":      "chat.created",
				"timestamp": "2024-01-15T10:30:00Z",
				"actor_id":  "550e8400-e29b-41d4-a716-446655440000",
				"chat_id":   "660e8400-e29b-41d4-a716-446655440001",
				"data": map[string]interface{}{
					"id":         "660e8400-e29b-41d4-a716-446655440001",
					"name":       "Новый чат",
					"chat_type":  "group",
					"created_by": "550e8400-e29b-41d4-a716-446655440000",
				},
			},
		},
		{
			Type:        "chat.updated",
			Exchange:    "chat.events",
			RoutingKey:  "chat.updated",
			Description: "Чат обновлён (название, описание). Отправляется всем участникам.",
			Channel:     "user:{userId}",
			Payload:     chatDataFields,
		},
		{
			Type:        "chat.deleted",
			Exchange:    "chat.events",
			RoutingKey:  "chat.deleted",
			Description: "Чат удалён. Отправляется всем участникам.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "chat_id", Type: "string (uuid)", Description: "ID удалённого чата", Required: true},
			},
		},
	}
}

func buildMessageEvents() []EventSchema {
	messageDataFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID сообщения", Required: true},
		{Name: "chat_id", Type: "string (uuid)", Description: "ID чата", Required: true},
		{Name: "sender_id", Type: "string (uuid)", Description: "ID отправителя", Required: true},
		{Name: "content", Type: "string", Description: "Текст сообщения", Required: true},
		{Name: "sent_at", Type: "string (RFC3339)", Description: "Время отправки", Required: true},
		{Name: "updated_at", Type: "string (RFC3339)", Description: "Время редактирования", Required: false},
		{Name: "parent_id", Type: "string (uuid)", Description: "ID родительского сообщения (reply)", Required: false},
		{Name: "sender_username", Type: "string", Description: "Username отправителя", Required: false},
		{Name: "sender_display_name", Type: "string", Description: "Display name отправителя", Required: false},
		{Name: "sender_avatar_url", Type: "string (url)", Description: "URL аватара отправителя", Required: false},
		{Name: "file_link_ids", Type: "array[string (uuid)]", Description: "IDs прикреплённых файлов", Required: false},
	}

	return []EventSchema{
		{
			Type:        "message.created",
			Exchange:    "chat.events",
			RoutingKey:  "message.created",
			Description: "Новое сообщение в чате. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     messageDataFields,
			Example: map[string]interface{}{
				"type":      "message.created",
				"timestamp": "2024-01-15T10:30:00Z",
				"actor_id":  "550e8400-e29b-41d4-a716-446655440000",
				"chat_id":   "660e8400-e29b-41d4-a716-446655440001",
				"data": map[string]interface{}{
					"id":                  "770e8400-e29b-41d4-a716-446655440002",
					"chat_id":             "660e8400-e29b-41d4-a716-446655440001",
					"sender_id":           "550e8400-e29b-41d4-a716-446655440000",
					"content":             "Привет! Как дела?",
					"sent_at":             "2024-01-15T10:30:00Z",
					"sender_username":     "john_doe",
					"sender_display_name": "John Doe",
				},
			},
		},
		{
			Type:        "message.updated",
			Exchange:    "chat.events",
			RoutingKey:  "message.updated",
			Description: "Сообщение отредактировано. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     messageDataFields,
		},
		{
			Type:        "message.deleted",
			Exchange:    "chat.events",
			RoutingKey:  "message.deleted",
			Description: "Сообщение удалено (soft delete). Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "message_id", Type: "string (uuid)", Description: "ID удалённого сообщения", Required: true},
				{Name: "chat_id", Type: "string (uuid)", Description: "ID чата", Required: true},
				{Name: "is_moderated_deletion", Type: "boolean", Description: "True если удалено модератором (не автором)", Required: true},
			},
		},
		{
			Type:        "message.restored",
			Exchange:    "chat.events",
			RoutingKey:  "message.restored",
			Description: "Сообщение восстановлено после удаления. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "message_id", Type: "string (uuid)", Description: "ID восстановленного сообщения", Required: true},
				{Name: "chat_id", Type: "string (uuid)", Description: "ID чата", Required: true},
				{Name: "sender_id", Type: "string (uuid)", Description: "ID отправителя", Required: true},
				{Name: "content", Type: "string", Description: "Текст сообщения", Required: true},
			},
		},
	}
}

func buildTypingEvents() []EventSchema {
	return []EventSchema{
		{
			Type:        "typing",
			Exchange:    "chat.events",
			RoutingKey:  "typing",
			Description: "Индикатор набора текста. Отправляется всем участникам чата кроме автора.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя, который печатает", Required: true},
				{Name: "is_typing", Type: "boolean", Description: "True если печатает, false если прекратил", Required: true},
			},
			Example: map[string]interface{}{
				"type":      "typing",
				"timestamp": "2024-01-15T10:30:00Z",
				"actor_id":  "550e8400-e29b-41d4-a716-446655440000",
				"chat_id":   "660e8400-e29b-41d4-a716-446655440001",
				"data": map[string]interface{}{
					"user_id":   "550e8400-e29b-41d4-a716-446655440000",
					"is_typing": true,
				},
			},
		},
	}
}

func buildReactionEvents() []EventSchema {
	reactionFields := []EventField{
		{Name: "message_id", Type: "string (uuid)", Description: "ID сообщения", Required: true},
		{Name: "emoji", Type: "string", Description: "Emoji реакции", Required: true, Example: "👍"},
		{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
	}

	return []EventSchema{
		{
			Type:        "reaction.added",
			Exchange:    "chat.events",
			RoutingKey:  "reaction.added",
			Description: "Добавлена реакция на сообщение. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     reactionFields,
		},
		{
			Type:        "reaction.removed",
			Exchange:    "chat.events",
			RoutingKey:  "reaction.removed",
			Description: "Удалена реакция с сообщения. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     reactionFields,
		},
	}
}

func buildThreadEvents() []EventSchema {
	threadFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID треда", Required: true},
		{Name: "chat_id", Type: "string (uuid)", Description: "ID чата", Required: true},
		{Name: "parent_message_id", Type: "string (uuid)", Description: "ID родительского сообщения", Required: false},
		{Name: "thread_type", Type: "string", Description: "Тип треда", Required: true, Enum: []string{"user", "system", "conference"}},
		{Name: "title", Type: "string", Description: "Заголовок треда", Required: false},
		{Name: "message_count", Type: "integer", Description: "Количество сообщений", Required: true},
		{Name: "created_by", Type: "string (uuid)", Description: "ID создателя", Required: false},
		{Name: "is_archived", Type: "boolean", Description: "Флаг архивации", Required: true},
		{Name: "restricted_participants", Type: "boolean", Description: "Ограничен ли список участников", Required: true},
	}

	return []EventSchema{
		{
			Type:        "thread.created",
			Exchange:    "chat.events",
			RoutingKey:  "thread.created",
			Description: "Создан новый тред. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     threadFields,
		},
		{
			Type:        "thread.archived",
			Exchange:    "chat.events",
			RoutingKey:  "thread.archived",
			Description: "Тред заархивирован. Отправляется всем участникам чата.",
			Channel:     "user:{userId}",
			Payload:     threadFields,
		},
	}
}

func buildConferenceEvents() []EventSchema {
	conferenceFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID конференции", Required: true},
		{Name: "name", Type: "string", Description: "Название конференции", Required: true},
		{Name: "chat_id", Type: "string (uuid)", Description: "ID связанного чата", Required: false},
		{Name: "created_by", Type: "string (uuid)", Description: "ID создателя", Required: true},
		{Name: "status", Type: "string", Description: "Статус конференции", Required: true, Enum: []string{"active", "ended"}},
		{Name: "max_members", Type: "integer", Description: "Максимум участников", Required: true},
		{Name: "participant_count", Type: "integer", Description: "Текущее количество участников", Required: true},
		{Name: "started_at", Type: "string (RFC3339)", Description: "Время начала", Required: false},
		{Name: "ended_at", Type: "string (RFC3339)", Description: "Время окончания", Required: false},
		{Name: "created_at", Type: "string (RFC3339)", Description: "Время создания", Required: true},
	}

	scheduledFields := append(conferenceFields,
		EventField{Name: "event_type", Type: "string", Description: "Тип события", Required: true, Enum: []string{"adhoc", "adhoc_chat", "scheduled", "recurring"}},
		EventField{Name: "scheduled_at", Type: "string (RFC3339)", Description: "Запланированное время", Required: false},
		EventField{Name: "series_id", Type: "string (uuid)", Description: "ID серии (для recurring)", Required: false},
		EventField{Name: "accepted_count", Type: "integer", Description: "Количество принявших", Required: true},
		EventField{Name: "declined_count", Type: "integer", Description: "Количество отклонивших", Required: true},
	)

	return []EventSchema{
		{
			Type:        "conference.created",
			Exchange:    "voice.events",
			RoutingKey:  "conference.created",
			Description: "Создана новая конференция. Отправляется участникам связанного чата.",
			Channel:     "user:{userId}",
			Payload:     conferenceFields,
		},
		{
			Type:        "conference.ended",
			Exchange:    "voice.events",
			RoutingKey:  "conference.ended",
			Description: "Конференция завершена. Отправляется всем участникам.",
			Channel:     "user:{userId}",
			Payload:     conferenceFields,
		},
		{
			Type:        "conference.scheduled",
			Exchange:    "voice.events",
			RoutingKey:  "conference.scheduled",
			Description: "Запланирована новая конференция. Отправляется приглашённым участникам.",
			Channel:     "user:{userId}",
			Payload:     scheduledFields,
		},
		{
			Type:        "conference.cancelled",
			Exchange:    "voice.events",
			RoutingKey:  "conference.cancelled",
			Description: "Запланированная конференция отменена. Отправляется всем приглашённым.",
			Channel:     "user:{userId}",
			Payload:     scheduledFields,
		},
		{
			Type:        "conference.rsvp_updated",
			Exchange:    "voice.events",
			RoutingKey:  "conference.rsvp_updated",
			Description: "Обновлён RSVP статус участника. Отправляется организатору и участнику.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "conference_id", Type: "string (uuid)", Description: "ID конференции", Required: true},
				{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
				{Name: "rsvp_status", Type: "string", Description: "Новый RSVP статус", Required: true, Enum: []string{"pending", "accepted", "declined"}},
			},
		},
		{
			Type:        "conference.reminder",
			Exchange:    "voice.events",
			RoutingKey:  "conference.reminder",
			Description: "Напоминание о предстоящей конференции. Отправляется участнику.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "conference_id", Type: "string (uuid)", Description: "ID конференции", Required: true},
				{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
				{Name: "conference_name", Type: "string", Description: "Название конференции", Required: true},
				{Name: "scheduled_at", Type: "string (RFC3339)", Description: "Запланированное время", Required: true},
				{Name: "minutes_before", Type: "integer", Description: "За сколько минут до начала", Required: true},
			},
		},
	}
}

func buildParticipantEvents() []EventSchema {
	participantFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID записи участника", Required: true},
		{Name: "conference_id", Type: "string (uuid)", Description: "ID конференции", Required: true},
		{Name: "chat_id", Type: "string (uuid)", Description: "ID связанного чата", Required: false},
		{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
		{Name: "status", Type: "string", Description: "Статус участника", Required: true, Enum: []string{"connecting", "joined", "left", "kicked"}},
		{Name: "is_muted", Type: "boolean", Description: "Микрофон выключен", Required: true},
		{Name: "is_deaf", Type: "boolean", Description: "Звук выключен", Required: true},
		{Name: "is_speaking", Type: "boolean", Description: "Участник говорит", Required: true},
		{Name: "username", Type: "string", Description: "Username участника", Required: false},
		{Name: "display_name", Type: "string", Description: "Display name участника", Required: false},
		{Name: "avatar_url", Type: "string (url)", Description: "URL аватара", Required: false},
		{Name: "joined_at", Type: "string (RFC3339)", Description: "Время присоединения", Required: false},
		{Name: "left_at", Type: "string (RFC3339)", Description: "Время выхода", Required: false},
	}

	return []EventSchema{
		{
			Type:        "participant.joined",
			Exchange:    "voice.events",
			RoutingKey:  "participant.joined",
			Description: "Участник присоединился к конференции. Отправляется всем участникам конференции.",
			Channel:     "user:{userId}",
			Payload:     participantFields,
		},
		{
			Type:        "participant.left",
			Exchange:    "voice.events",
			RoutingKey:  "participant.left",
			Description: "Участник покинул конференцию. Отправляется всем участникам конференции.",
			Channel:     "user:{userId}",
			Payload:     participantFields,
		},
		{
			Type:        "participant.muted",
			Exchange:    "voice.events",
			RoutingKey:  "participant.muted",
			Description: "Изменён статус mute участника. Отправляется всем участникам конференции.",
			Channel:     "user:{userId}",
			Payload:     participantFields,
		},
		{
			Type:        "participant.speaking",
			Exchange:    "voice.events",
			RoutingKey:  "participant.speaking",
			Description: "Участник начал/прекратил говорить. Отправляется всем участникам конференции.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "participant_id", Type: "string (uuid)", Description: "ID участника", Required: true},
				{Name: "is_speaking", Type: "boolean", Description: "Говорит ли участник", Required: true},
			},
		},
		{
			Type:        "participant.role_changed",
			Exchange:    "voice.events",
			RoutingKey:  "participant.role_changed",
			Description: "Изменена роль участника в конференции. Отправляется всем участникам.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "conference_id", Type: "string (uuid)", Description: "ID конференции", Required: true},
				{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
				{Name: "old_role", Type: "string", Description: "Старая роль", Required: true, Enum: []string{"originator", "moderator", "speaker", "assistant", "participant"}},
				{Name: "new_role", Type: "string", Description: "Новая роль", Required: true, Enum: []string{"originator", "moderator", "speaker", "assistant", "participant"}},
			},
		},
		{
			Type:        "participant.added",
			Exchange:    "voice.events",
			RoutingKey:  "participant.added",
			Description: "Участник добавлен в конференцию (для scheduled). Отправляется добавленному участнику.",
			Channel:     "user:{userId}",
			Payload:     participantFields,
		},
		{
			Type:        "participant.removed",
			Exchange:    "voice.events",
			RoutingKey:  "participant.removed",
			Description: "Участник удалён из конференции. Отправляется удалённому участнику.",
			Channel:     "user:{userId}",
			Payload: []EventField{
				{Name: "conference_id", Type: "string (uuid)", Description: "ID конференции", Required: true},
				{Name: "user_id", Type: "string (uuid)", Description: "ID пользователя", Required: true},
			},
		},
	}
}

func buildCallEvents() []EventSchema {
	callFields := []EventField{
		{Name: "id", Type: "string (uuid)", Description: "ID звонка", Required: true},
		{Name: "caller_id", Type: "string (uuid)", Description: "ID звонящего", Required: true},
		{Name: "callee_id", Type: "string (uuid)", Description: "ID вызываемого", Required: true},
		{Name: "chat_id", Type: "string (uuid)", Description: "ID связанного чата", Required: false},
		{Name: "conference_id", Type: "string (uuid)", Description: "ID созданной конференции", Required: false},
		{Name: "status", Type: "string", Description: "Статус звонка", Required: true, Enum: []string{"initiated", "ringing", "answered", "ended", "missed", "failed"}},
		{Name: "duration", Type: "integer", Description: "Длительность в секундах", Required: true},
		{Name: "end_reason", Type: "string", Description: "Причина завершения", Required: false},
		{Name: "caller_username", Type: "string", Description: "Username звонящего", Required: false},
		{Name: "caller_display_name", Type: "string", Description: "Display name звонящего", Required: false},
		{Name: "callee_username", Type: "string", Description: "Username вызываемого", Required: false},
		{Name: "callee_display_name", Type: "string", Description: "Display name вызываемого", Required: false},
		{Name: "started_at", Type: "string (RFC3339)", Description: "Время начала", Required: false},
		{Name: "answered_at", Type: "string (RFC3339)", Description: "Время ответа", Required: false},
		{Name: "ended_at", Type: "string (RFC3339)", Description: "Время завершения", Required: false},
	}

	return []EventSchema{
		{
			Type:        "call.initiated",
			Exchange:    "voice.events",
			RoutingKey:  "call.initiated",
			Description: "Инициирован звонок. Отправляется вызываемому пользователю.",
			Channel:     "user:{userId}",
			Payload:     callFields,
			Example: map[string]interface{}{
				"id":                   "880e8400-e29b-41d4-a716-446655440003",
				"caller_id":            "550e8400-e29b-41d4-a716-446655440000",
				"callee_id":            "550e8400-e29b-41d4-a716-446655440001",
				"status":               "initiated",
				"duration":             0,
				"caller_username":      "john_doe",
				"caller_display_name":  "John Doe",
				"callee_username":      "jane_doe",
				"callee_display_name":  "Jane Doe",
				"started_at":           "2024-01-15T10:30:00Z",
			},
		},
		{
			Type:        "call.answered",
			Exchange:    "voice.events",
			RoutingKey:  "call.answered",
			Description: "Звонок принят. Отправляется обоим участникам.",
			Channel:     "user:{userId}",
			Payload:     callFields,
		},
		{
			Type:        "call.ended",
			Exchange:    "voice.events",
			RoutingKey:  "call.ended",
			Description: "Звонок завершён. Отправляется обоим участникам.",
			Channel:     "user:{userId}",
			Payload:     callFields,
		},
	}
}
