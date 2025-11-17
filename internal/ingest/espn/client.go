package espn

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"
)

const (
	BaseURL       = "https://site.api.espn.com/apis/site/v2/sports"
	BasketballNBA = "basketball/nba"
)

// Client handles ESPN API requests
// Note: Uses curl internally because ESPN blocks Go's HTTP client fingerprint
type Client struct {
	baseURL string
}

// New creates a new ESPN API client with a custom base URL
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = BaseURL
	}
	log.Printf("[espn-client] New() called with baseURL: %s", baseURL)
	return &Client{
		baseURL: baseURL,
	}
}

// NewClient creates a new ESPN API client with default settings
func NewClient() *Client {
	return New(BaseURL)
}

// FetchScoreboard fetches games for a specific date
// If date is zero, fetches ESPN's "today" (includes games within ~24 hours)
func (c *Client) FetchScoreboard(ctx context.Context, sportPath string, date time.Time) (map[string]interface{}, error) {
	var url string
	if date.IsZero() {
		// No date specified - get ESPN's "today"
		url = fmt.Sprintf("%s/%s/scoreboard", c.baseURL, sportPath)
	} else {
		// Specific date in YYYYMMDD format
		dateStr := date.Format("20060102")
		url = fmt.Sprintf("%s/%s/scoreboard?dates=%s", c.baseURL, sportPath, dateStr)
	}

	return c.fetch(ctx, url)
}

// FetchGameSummary fetches detailed game summary with box scores
func (c *Client) FetchGameSummary(ctx context.Context, sportPath string, gameID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s/summary?event=%s", c.baseURL, sportPath, gameID)
	return c.fetch(ctx, url)
}

// fetch makes an HTTP GET request using curl
// ESPN blocks Go's HTTP client but curl works reliably
func (c *Client) fetch(ctx context.Context, url string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "curl", "-s", "-L", "-m", "15", url)
	
	// Debug: log the command being run
	log.Printf("[espn-client] Running: curl -s -L -m 15 %s", url)
	
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[espn-client] ❌ curl failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("curl failed: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("curl execution failed: %w", err)
	}

	// Debug: log first 200 chars of output
	log.Printf("[espn-client] ✓ Response (first 200 chars): %s", string(output[:min(len(output), 200)]))

	// Check if we got HTML error page (403, 404, etc.)
	if len(output) > 0 && output[0] == '<' {
		return nil, fmt.Errorf("ESPN returned HTML error page: %s", string(output[:min(len(output), 200)]))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w (body: %s)", err, string(output[:min(len(output), 200)]))
	}

	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
