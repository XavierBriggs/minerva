package google

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const (
	// BaseURL for Google Sports searches
	BaseURL = "https://www.google.com/search"
	
	// UserAgent for requests
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	
	// MinRequestInterval to prevent rate limiting
	MinRequestInterval = 2 * time.Second
)

// Client handles Google Sports scraping with rate limiting
type Client struct {
	lastRequest time.Time
	interval    time.Duration
	
	// Chromedp context for headless browser
	allocCtx context.Context
	cancel   context.CancelFunc
}

// NewClient creates a new Google Sports scraper client
func NewClient() (*Client, error) {
	// Create chrome instance with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(UserAgent),
	)
	
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	
	return &Client{
		lastRequest: time.Time{},
		interval:    MinRequestInterval,
		allocCtx:    allocCtx,
		cancel:      cancel,
	}, nil
}

// Close releases resources
func (c *Client) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// FetchLiveGames fetches current NBA games from Google Sports
func (c *Client) FetchLiveGames(ctx context.Context) (string, error) {
	return c.fetchWithRateLimit(ctx, "nba games today")
}

// FetchGameDetails fetches detailed information for a specific game
func (c *Client) FetchGameDetails(ctx context.Context, homeTeam, awayTeam string) (string, error) {
	query := fmt.Sprintf("nba %s vs %s", homeTeam, awayTeam)
	return c.fetchWithRateLimit(ctx, query)
}

// fetchWithRateLimit fetches content with automatic rate limiting
func (c *Client) fetchWithRateLimit(ctx context.Context, query string) (string, error) {
	// Enforce rate limiting
	if !c.lastRequest.IsZero() {
		elapsed := time.Since(c.lastRequest)
		if elapsed < c.interval {
			waitTime := c.interval - elapsed
			log.Printf("Rate limiting: waiting %v before next request", waitTime)
			time.Sleep(waitTime)
		}
	}
	
	html, err := c.fetch(ctx, query)
	c.lastRequest = time.Now()
	
	return html, err
}

// fetch performs the actual HTTP fetch using chromedp
func (c *Client) fetch(ctx context.Context, query string) (string, error) {
	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Create a new browser context
	browserCtx, cancel := chromedp.NewContext(c.allocCtx)
	defer cancel()
	
	// Combine with timeout context
	browserCtx, cancel = context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()
	
	var htmlContent string
	url := fmt.Sprintf("%s?q=%s", BaseURL, strings.ReplaceAll(query, " ", "+"))
	
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second), // Allow JS to render
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)
	
	if err != nil {
		return "", fmt.Errorf("chromedp error: %w", err)
	}
	
	if htmlContent == "" {
		return "", fmt.Errorf("empty HTML content returned")
	}
	
	return htmlContent, nil
}

// ParseHTML converts raw HTML to a goquery Document for parsing
func ParseHTML(htmlContent string) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}


