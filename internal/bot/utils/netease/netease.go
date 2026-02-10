package netease

import (
	"LanMei/internal/bot/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 8 * time.Second

var defaultBaseURL = "http://ncm-api:3000"

type SongInfo struct {
	ID         int64
	Name       string
	Artists    []string
	Album      string
	DurationMs int64
}

type Client struct {
	baseURL string
	timeout time.Duration
}

func NewClient() *Client {
	baseURL := defaultBaseURL
	timeout := defaultTimeout
	if config.K != nil {
		if v := strings.TrimSpace(config.K.String("Music.BaseURL")); v != "" {
			baseURL = v
		}
	}
	return &Client{
		baseURL: baseURL,
		timeout: timeout,
	}
}

func (c *Client) SearchSongs(keywords string, limit int) ([]SongInfo, error) {
	keywords = strings.TrimSpace(keywords)
	if keywords == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 3
	}

	baseURL := c.baseURL
	endpoint, err := buildURL(baseURL, "/search", map[string]string{
		"keywords": keywords,
		"limit":    fmt.Sprintf("%d", limit),
		"type":     "1",
	})
	if err != nil {
		return nil, err
	}
	var resp searchResponse
	if err := c.getJSON(endpoint, &resp); err != nil {
		return nil, err
	}
	if resp.Result == nil || len(resp.Result.Songs) == 0 {
		return []SongInfo{}, nil
	}
	results := make([]SongInfo, 0, limit)
	for _, song := range resp.Result.Songs {
		if len(results) >= limit {
			break
		}
		artists := pickArtists(song.Artists, song.AR)
		album := pickAlbumName(song.Album, song.AL)
		info := SongInfo{
			ID:         song.ID,
			Name:       song.Name,
			Artists:    artists,
			Album:      album,
			DurationMs: pickDuration(song.Duration, song.Dt),
		}
		results = append(results, info)
	}
	return results, nil
}

func (c *Client) getJSON(endpoint string, out interface{}) error {
	timeout := c.timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("netease status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func buildURL(baseURL string, path string, query map[string]string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", errors.New("empty base url")
	}
	endpoint, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}
	params := endpoint.Query()
	for k, v := range query {
		params.Set(k, v)
	}
	endpoint.RawQuery = params.Encode()
	return endpoint.String(), nil
}

func pickArtists(primary []artistName, fallback []artistName) []string {
	artists := primary
	if len(artists) == 0 {
		artists = fallback
	}
	if len(artists) == 0 {
		return nil
	}
	result := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name == "" {
			continue
		}
		result = append(result, artist.Name)
	}
	return result
}

func pickAlbumName(primary albumName, fallback albumName) string {
	if primary.Name != "" {
		return primary.Name
	}
	return fallback.Name
}

func pickDuration(primary int64, fallback int64) int64 {
	if primary > 0 {
		return primary
	}
	return fallback
}

type artistName struct {
	Name string `json:"name"`
}

type albumName struct {
	Name string `json:"name"`
}

type searchResponse struct {
	Result *struct {
		Songs []struct {
			ID       int64        `json:"id"`
			Name     string       `json:"name"`
			Artists  []artistName `json:"artists"`
			AR       []artistName `json:"ar"`
			Album    albumName    `json:"album"`
			AL       albumName    `json:"al"`
			Duration int64        `json:"duration"`
			Dt       int64        `json:"dt"`
		} `json:"songs"`
	} `json:"result"`
}
