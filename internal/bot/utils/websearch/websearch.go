package websearch

import (
	"LanMei/internal/bot/config"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOpenSerpURL = "http://openserp:7000/mega/search"
	defaultTimeout     = 8 * time.Second
)

type Result struct {
	Title   string
	Snippet string
	URL     string
}

type Client struct {
	baseURL   string
	timeout   time.Duration
	userAgent string
	engine    string
}

func NewClient() *Client {
	baseURL := defaultOpenSerpURL
	timeout := defaultTimeout
	userAgent := "LanMeiBot/1.0 (+https://example.com)"
	engine := ""
	if config.K != nil {
		if v := strings.TrimSpace(config.K.String("Search.BaseURL")); v != "" {
			baseURL = v
		}
		if v := config.K.Int("Search.TimeoutSeconds"); v > 0 {
			timeout = time.Duration(v) * time.Second
		}
		if v := strings.TrimSpace(config.K.String("Search.UserAgent")); v != "" {
			userAgent = v
		}
		if v := strings.TrimSpace(config.K.String("Search.Engines")); v != "" {
			engine = v
		}
	}
	return &Client{
		baseURL:   baseURL,
		timeout:   timeout,
		userAgent: userAgent,
		engine:    engine,
	}
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 4
	}
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = defaultOpenSerpURL
	}
	endpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	params := endpoint.Query()
	params.Set("text", query)
	if c.engine != "" {
		params.Set("engines", c.engine)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	endpoint.RawQuery = params.Encode()

	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseOpenSerpResults(body, limit)
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func cleanSnippet(snippet string) string {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return ""
	}
	snippet = html.UnescapeString(snippet)
	snippet = htmlTagRe.ReplaceAllString(snippet, "")
	return strings.TrimSpace(snippet)
}

func parseOpenSerpResults(payload []byte, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = 4
	}
	var items []openSerpResult
	if err := json.Unmarshal(payload, &items); err != nil {
		var wrapper map[string]any
		if err := json.Unmarshal(payload, &wrapper); err != nil {
			return nil, err
		}
		items = extractOpenSerpItems(wrapper)
	}
	results := make([]Result, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		urlText := strings.TrimSpace(item.URL)
		snippet := strings.TrimSpace(item.Description)
		if snippet == "" {
			snippet = strings.TrimSpace(item.Snippet)
		}
		if title == "" && urlText == "" && snippet == "" {
			continue
		}
		results = append(results, Result{
			Title:   cleanSnippet(title),
			Snippet: cleanSnippet(snippet),
			URL:     urlText,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

type openSerpResult struct {
	Rank        int    `json:"rank"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Snippet     string `json:"snippet"`
	Engine      string `json:"engine"`
}

func extractOpenSerpItems(wrapper map[string]any) []openSerpResult {
	keys := []string{"results", "data", "items", "search_results"}
	for _, key := range keys {
		if raw := wrapper[key]; raw != nil {
			if items := coerceOpenSerpResults(raw); len(items) > 0 {
				return items
			}
		}
	}
	if raw, ok := wrapper["result"]; ok {
		if sub, ok := raw.(map[string]any); ok {
			for _, key := range keys {
				if items := coerceOpenSerpResults(sub[key]); len(items) > 0 {
					return items
				}
			}
		}
	}
	return nil
}

func coerceOpenSerpResults(raw any) []openSerpResult {
	items := []openSerpResult{}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				items = append(items, mapToOpenSerpResult(m))
			}
		}
	case map[string]any:
		items = append(items, mapToOpenSerpResult(v))
	}
	return items
}

func mapToOpenSerpResult(m map[string]any) openSerpResult {
	return openSerpResult{
		URL:         toString(m["url"]),
		Title:       toString(m["title"]),
		Description: toString(m["description"]),
		Snippet:     toString(m["snippet"]),
	}
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}
