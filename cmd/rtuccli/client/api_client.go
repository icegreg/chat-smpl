package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is HTTP client for admin-service REST API
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Conference represents a conference
type Conference struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	EventType     string    `json:"event_type"`
	ChatID        *string   `json:"chat_id"`
	Status        string    `json:"status"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	StartedAt     *time.Time `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at"`
	Duration      *int64    `json:"duration_seconds"`
	Participants  int       `json:"participants_count"`
	Quality       string    `json:"quality"`
}

// Participant represents a participant
type Participant struct {
	ID           string     `json:"id"`
	ConferenceID string     `json:"conference_id"`
	UserID       string     `json:"user_id"`
	Username     string     `json:"username"`
	Extension    string     `json:"extension"`
	Status       string     `json:"status"`
	JoinedAt     *time.Time `json:"joined_at"`
	LeftAt       *time.Time `json:"left_at"`
	Duration     *int64     `json:"duration_seconds"`
	Device       string     `json:"device"`
	Quality      string     `json:"quality"`
}

// Service represents a microservice
type Service struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Host        string     `json:"host"`
	Port        int        `json:"port"`
	Status      string     `json:"status"`
	Health      string     `json:"health"`
	Uptime      *int64     `json:"uptime_seconds"`
	CPU         float64    `json:"cpu_percent"`
	Memory      int64      `json:"memory_bytes"`
	Connections int        `json:"connections"`
	LastCheck   *time.Time `json:"last_check"`
}

// ListConferences gets list of conferences
func (c *APIClient) ListConferences(status string) ([]Conference, error) {
	url := c.BaseURL + "/api/conferences"
	if status != "" {
		url += "?status=" + status
	}

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Conferences []Conference `json:"conferences"`
		Total       int          `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Conferences, nil
}

// GetConference gets a single conference
func (c *APIClient) GetConference(id string) (*Conference, error) {
	url := c.BaseURL + "/api/conferences/" + id

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var conf Conference
	if err := json.NewDecoder(resp.Body).Decode(&conf); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &conf, nil
}

// ListParticipants gets participants of a conference
func (c *APIClient) ListParticipants(conferenceID string) ([]Participant, error) {
	url := c.BaseURL + "/api/conferences/" + conferenceID + "/participants"

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Participants []Participant `json:"participants"`
		Total        int           `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Participants, nil
}

// ListServices gets list of services
func (c *APIClient) ListServices() ([]Service, error) {
	url := c.BaseURL + "/api/services"

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Services []Service `json:"services"`
		Total    int       `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Services, nil
}

// GetService gets a single service
func (c *APIClient) GetService(id string) (*Service, error) {
	url := c.BaseURL + "/api/services/" + id

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var svc Service
	if err := json.NewDecoder(resp.Body).Decode(&svc); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &svc, nil
}
