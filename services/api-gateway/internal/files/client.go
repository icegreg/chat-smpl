package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FileAttachment represents file metadata for message attachments
type FileAttachment struct {
	LinkID           string `json:"link_id"`
	ID               string `json:"id"`
	Filename         string `json:"filename"`
	OriginalFilename string `json:"original_filename"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
}

// Client is an HTTP client for the files service
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new files service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetFilesByLinkIDs fetches file metadata for multiple link IDs
func (c *Client) GetFilesByLinkIDs(ctx context.Context, linkIDs []string) ([]FileAttachment, error) {
	if len(linkIDs) == 0 {
		return []FileAttachment{}, nil
	}

	reqBody := struct {
		LinkIDs []string `json:"link_ids"`
	}{
		LinkIDs: linkIDs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/files/batch", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("files service returned status %d", resp.StatusCode)
	}

	var files []FileAttachment
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return files, nil
}

// GrantPermissions grants file permissions to a list of users
func (c *Client) GrantPermissions(ctx context.Context, linkIDs []string, userIDs []string, uploaderID string) error {
	if len(linkIDs) == 0 || len(userIDs) == 0 {
		return nil
	}

	reqBody := struct {
		LinkIDs []string `json:"link_ids"`
		UserIDs []string `json:"user_ids"`
	}{
		LinkIDs: linkIDs,
		UserIDs: userIDs,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/files/grant-permissions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", uploaderID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("files service returned status %d", resp.StatusCode)
	}

	return nil
}
