package esl

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Event represents a FreeSWITCH event
type Event struct {
	Headers map[string]string
	Body    string
}

// GetHeader returns a header value from the event
func (e *Event) GetHeader(key string) string {
	if e.Headers == nil {
		return ""
	}
	return e.Headers[key]
}

// EventHandler is a callback function for FreeSWITCH events
type EventHandler func(event *Event)

// Client interface for FreeSWITCH ESL operations
type Client interface {
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool

	// Conference operations
	CreateConference(ctx context.Context, confName, profile string) error
	ListConferences(ctx context.Context) ([]ConferenceInfo, error)
	GetConferenceMembers(ctx context.Context, confName string) ([]ConferenceMember, error)
	KickMember(ctx context.Context, confName, memberID string) error
	MuteMember(ctx context.Context, confName, memberID string, mute bool) error
	DeafMember(ctx context.Context, confName, memberID string, deaf bool) error

	// Call operations
	Originate(ctx context.Context, dest string, vars map[string]string) (string, error)
	Hangup(ctx context.Context, uuid, cause string) error
	Transfer(ctx context.Context, uuid, dest string) error

	// Events
	SubscribeEvents(events ...string) error
	OnEvent(handler EventHandler)
}

// ConferenceInfo represents information about a conference
type ConferenceInfo struct {
	Name          string
	MemberCount   int
	RunningTime   int
	Locked        bool
	RecordingPath string
}

// ConferenceMember represents a member in a conference
type ConferenceMember struct {
	ID         string
	UUID       string
	CallerName string
	CallerID   string
	Muted      bool
	Deaf       bool
	Speaking   bool
	JoinTime   int64
}

type eslClient struct {
	conn       net.Conn
	reader     *textproto.Reader
	host       string
	port       int
	password   string
	mu         sync.RWMutex
	handlers   []EventHandler
	connected  bool
	logger     *zap.Logger
	done       chan struct{}
	reconnectC chan struct{}
}

// NewClient creates a new ESL client
func NewClient(host string, port int, password string, logger *zap.Logger) Client {
	return &eslClient{
		host:       host,
		port:       port,
		password:   password,
		handlers:   make([]EventHandler, 0),
		logger:     logger,
		done:       make(chan struct{}),
		reconnectC: make(chan struct{}, 1),
	}
}

// Connect establishes connection to FreeSWITCH
func (c *eslClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	c.logger.Info("connecting to FreeSWITCH ESL", zap.String("addr", addr))

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to FreeSWITCH: %w", err)
	}

	c.conn = conn
	c.reader = textproto.NewReader(bufio.NewReader(conn))

	// Read the initial Content-Type header
	if err := c.readInitialResponse(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read initial response: %w", err)
	}

	// Authenticate
	if err := c.authenticate(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	c.connected = true
	c.logger.Info("connected to FreeSWITCH ESL")

	// Start event reader
	go c.readEvents()

	// Start reconnection handler
	go c.handleReconnect()

	return nil
}

// readInitialResponse reads the initial response from FreeSWITCH
func (c *eslClient) readInitialResponse() error {
	// FreeSWITCH sends: Content-Type: auth/request
	headers, err := c.readHeaders()
	if err != nil {
		return err
	}

	contentType := headers["Content-Type"]
	if contentType != "auth/request" {
		return fmt.Errorf("unexpected initial response: %s", contentType)
	}

	return nil
}

// authenticate sends the password to FreeSWITCH
func (c *eslClient) authenticate() error {
	cmd := fmt.Sprintf("auth %s\n\n", c.password)
	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to send auth command: %w", err)
	}

	headers, err := c.readHeaders()
	if err != nil {
		return err
	}

	reply := headers["Reply-Text"]
	if !strings.HasPrefix(reply, "+OK") {
		return fmt.Errorf("authentication failed: %s", reply)
	}

	return nil
}

// readHeaders reads MIME headers from the connection
func (c *eslClient) readHeaders() (map[string]string, error) {
	headers := make(map[string]string)

	for {
		line, err := c.reader.ReadLine()
		if err != nil {
			return nil, err
		}

		// Empty line indicates end of headers
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}

	return headers, nil
}

// readEvents reads events from FreeSWITCH and dispatches them
func (c *eslClient) readEvents() {
	for {
		select {
		case <-c.done:
			return
		default:
		}

		c.mu.RLock()
		connected := c.connected
		c.mu.RUnlock()

		if !connected {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		event, err := c.readEvent()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			c.logger.Error("error reading event", zap.Error(err))

			// Trigger reconnection
			select {
			case c.reconnectC <- struct{}{}:
			default:
			}
			return
		}

		if event != nil {
			c.handleEvent(event)
		}
	}
}

// readEvent reads a single event from FreeSWITCH
func (c *eslClient) readEvent() (*Event, error) {
	headers, err := c.readHeaders()
	if err != nil {
		return nil, err
	}

	if len(headers) == 0 {
		return nil, nil
	}

	event := &Event{
		Headers: make(map[string]string),
	}

	// Copy headers and URL-decode them
	for k, v := range headers {
		decoded, err := url.QueryUnescape(v)
		if err != nil {
			event.Headers[k] = v
		} else {
			event.Headers[k] = decoded
		}
	}

	// Check if there's a body
	if contentLength := headers["Content-Length"]; contentLength != "" {
		length, err := strconv.Atoi(contentLength)
		if err == nil && length > 0 {
			body := make([]byte, length)
			_, err := c.reader.R.Read(body)
			if err != nil {
				return nil, err
			}
			event.Body = string(body)
		}
	}

	return event, nil
}

// Close closes the ESL connection
func (c *eslClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.done)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false
	return nil
}

// IsConnected returns connection status
func (c *eslClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// handleReconnect handles automatic reconnection
func (c *eslClient) handleReconnect() {
	for {
		select {
		case <-c.done:
			return
		case <-c.reconnectC:
			c.mu.Lock()
			c.connected = false
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.mu.Unlock()

			// Exponential backoff for reconnection
			backoff := time.Second
			maxBackoff := time.Minute

			for {
				select {
				case <-c.done:
					return
				default:
				}

				c.logger.Info("attempting to reconnect to FreeSWITCH", zap.Duration("backoff", backoff))

				addr := fmt.Sprintf("%s:%d", c.host, c.port)
				conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
				if err != nil {
					c.logger.Error("reconnection failed", zap.Error(err))
					time.Sleep(backoff)
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}

				c.mu.Lock()
				c.conn = conn
				c.reader = textproto.NewReader(bufio.NewReader(conn))
				c.mu.Unlock()

				// Re-authenticate
				if err := c.readInitialResponse(); err != nil {
					c.logger.Error("reconnection auth failed", zap.Error(err))
					conn.Close()
					continue
				}

				if err := c.authenticate(); err != nil {
					c.logger.Error("reconnection auth failed", zap.Error(err))
					conn.Close()
					continue
				}

				c.mu.Lock()
				c.connected = true
				c.mu.Unlock()

				// Restart event reader
				go c.readEvents()

				c.logger.Info("reconnected to FreeSWITCH ESL")
				break
			}
		}
	}
}

// handleEvent processes incoming FreeSWITCH events
func (c *eslClient) handleEvent(event *Event) {
	c.mu.RLock()
	handlers := make([]EventHandler, len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// sendCommand sends a command to FreeSWITCH and returns the response
func (c *eslClient) sendCommand(cmd string) (*Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to FreeSWITCH")
	}

	// Send command
	fullCmd := cmd + "\n\n"
	if _, err := c.conn.Write([]byte(fullCmd)); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	return c.readEvent()
}

// sendAPI sends an API command to FreeSWITCH
func (c *eslClient) sendAPI(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return "", fmt.Errorf("not connected to FreeSWITCH")
	}

	// Send API command
	fullCmd := fmt.Sprintf("api %s\n\n", cmd)
	if _, err := c.conn.Write([]byte(fullCmd)); err != nil {
		return "", fmt.Errorf("failed to send API command: %w", err)
	}

	// Read response
	event, err := c.readEvent()
	if err != nil {
		return "", err
	}

	if event == nil {
		return "", nil
	}

	return event.Body, nil
}

// SubscribeEvents subscribes to specific FreeSWITCH events
func (c *eslClient) SubscribeEvents(events ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected to FreeSWITCH")
	}

	eventStr := strings.Join(events, " ")
	cmd := fmt.Sprintf("event plain %s\n\n", eventStr)

	if _, err := c.conn.Write([]byte(cmd)); err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Read response
	_, err := c.readEvent()
	return err
}

// OnEvent registers an event handler
func (c *eslClient) OnEvent(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// CreateConference creates a new conference (conferences are created dynamically when first user joins)
func (c *eslClient) CreateConference(ctx context.Context, confName, profile string) error {
	// FreeSWITCH creates conferences dynamically when first user joins
	// This is a no-op but we can pre-validate the conference name
	c.logger.Info("conference will be created on first join",
		zap.String("confName", confName),
		zap.String("profile", profile))
	return nil
}

// ListConferences returns list of active conferences
func (c *eslClient) ListConferences(ctx context.Context) ([]ConferenceInfo, error) {
	body, err := c.sendAPI("conference list")
	if err != nil {
		return nil, fmt.Errorf("failed to list conferences: %w", err)
	}

	if body == "" || strings.Contains(body, "No active conferences") {
		return []ConferenceInfo{}, nil
	}

	// Parse conference list output
	conferences := []ConferenceInfo{}
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "+") {
			continue
		}
		// Format: conference_name members: X running: Y locked: Z
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			info := ConferenceInfo{Name: parts[0]}
			// Parse additional fields if present
			for i, p := range parts {
				if p == "members:" && i+1 < len(parts) {
					fmt.Sscanf(parts[i+1], "%d", &info.MemberCount)
				}
			}
			conferences = append(conferences, info)
		}
	}

	return conferences, nil
}

// GetConferenceMembers returns members of a conference
func (c *eslClient) GetConferenceMembers(ctx context.Context, confName string) ([]ConferenceMember, error) {
	body, err := c.sendAPI(fmt.Sprintf("conference %s list", confName))
	if err != nil {
		return nil, fmt.Errorf("failed to get conference members: %w", err)
	}

	if body == "" || (strings.Contains(body, "Conference") && strings.Contains(body, "not found")) {
		return []ConferenceMember{}, nil
	}

	// Parse member list
	members := []ConferenceMember{}
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: member_id;uuid;caller_id_name;caller_id_number;flags;...
		parts := strings.Split(line, ";")
		if len(parts) >= 4 {
			member := ConferenceMember{
				ID:         parts[0],
				UUID:       parts[1],
				CallerName: parts[2],
				CallerID:   parts[3],
			}
			// Parse flags
			if len(parts) >= 5 {
				flags := parts[4]
				member.Muted = strings.Contains(flags, "mute")
				member.Deaf = strings.Contains(flags, "deaf")
				member.Speaking = strings.Contains(flags, "talking")
			}
			members = append(members, member)
		}
	}

	return members, nil
}

// KickMember kicks a member from conference
func (c *eslClient) KickMember(ctx context.Context, confName, memberID string) error {
	_, err := c.sendAPI(fmt.Sprintf("conference %s kick %s", confName, memberID))
	if err != nil {
		return fmt.Errorf("failed to kick member: %w", err)
	}
	return nil
}

// MuteMember mutes or unmutes a conference member
func (c *eslClient) MuteMember(ctx context.Context, confName, memberID string, mute bool) error {
	action := "mute"
	if !mute {
		action = "unmute"
	}

	_, err := c.sendAPI(fmt.Sprintf("conference %s %s %s", confName, action, memberID))
	if err != nil {
		return fmt.Errorf("failed to %s member: %w", action, err)
	}
	return nil
}

// DeafMember makes a conference member deaf (can't hear) or undeaf
func (c *eslClient) DeafMember(ctx context.Context, confName, memberID string, deaf bool) error {
	action := "deaf"
	if !deaf {
		action = "undeaf"
	}

	_, err := c.sendAPI(fmt.Sprintf("conference %s %s %s", confName, action, memberID))
	if err != nil {
		return fmt.Errorf("failed to %s member: %w", action, err)
	}
	return nil
}

// Originate initiates a new call
func (c *eslClient) Originate(ctx context.Context, dest string, vars map[string]string) (string, error) {
	// Build variable string
	varParts := []string{}
	for k, v := range vars {
		varParts = append(varParts, fmt.Sprintf("%s=%s", k, v))
	}
	varStr := ""
	if len(varParts) > 0 {
		varStr = "{" + strings.Join(varParts, ",") + "}"
	}

	body, err := c.sendAPI(fmt.Sprintf("originate %s%s &park", varStr, dest))
	if err != nil {
		return "", fmt.Errorf("failed to originate call: %w", err)
	}

	// Extract UUID from response
	if strings.HasPrefix(body, "+OK ") {
		return strings.TrimSpace(strings.TrimPrefix(body, "+OK ")), nil
	}

	return "", fmt.Errorf("originate failed: %s", body)
}

// Hangup hangs up a call
func (c *eslClient) Hangup(ctx context.Context, uuid, cause string) error {
	if cause == "" {
		cause = "NORMAL_CLEARING"
	}

	_, err := c.sendAPI(fmt.Sprintf("uuid_kill %s %s", uuid, cause))
	if err != nil {
		return fmt.Errorf("failed to hangup call: %w", err)
	}
	return nil
}

// Transfer transfers a call to a new destination
func (c *eslClient) Transfer(ctx context.Context, uuid, dest string) error {
	_, err := c.sendAPI(fmt.Sprintf("uuid_transfer %s %s", uuid, dest))
	if err != nil {
		return fmt.Errorf("failed to transfer call: %w", err)
	}
	return nil
}

// GetHeader is a helper to get header from textproto.MIMEHeader
func GetHeader(headers textproto.MIMEHeader, key string) string {
	values := headers[textproto.CanonicalMIMEHeaderKey(key)]
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
