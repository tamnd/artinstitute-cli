// Package artinstitute is the library behind the artinstitute command line:
// the HTTP client, request shaping, and typed data models for the Art Institute
// of Chicago public API (https://api.artic.edu/api/v1).
//
// No API key is required. The Client paces requests, sets a real User-Agent,
// and retries transient failures (429 and 5xx).
package artinstitute

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the Art Institute API.
const DefaultUserAgent = "artinstitute-cli/0.1.0 (github.com/tamnd/artinstitute-cli)"

// Host is the API hostname.
const Host = "api.artic.edu"

// SiteHost is the public website host (for synthesized URLs).
const SiteHost = "www.artic.edu"

// BaseURL is the root every API request is built from.
const BaseURL = "https://" + Host + "/api/v1"

// Artwork holds the public data about a single artwork.
type Artwork struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	ArtistDisplay string `json:"artist_display"`
	DateDisplay   string `json:"date_display"`
	MediumDisplay string `json:"medium_display"`
	Dimensions    string `json:"dimensions"`
	PlaceOfOrigin string `json:"place_of_origin"`
	Description   string `json:"description,omitempty"`
	ThumbnailURL  string `json:"thumbnail_url,omitempty"`
	URL           string `json:"url"`
}

// Agent holds the public data about an artist or organization.
type Agent struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	AgentType   string `json:"agent_type"`
	BirthDate   int    `json:"birth_date,omitempty"`
	DeathDate   int    `json:"death_date,omitempty"`
	URL         string `json:"url"`
}

// Exhibition holds the public data about a museum exhibition.
type Exhibition struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	StartAt string `json:"start_at,omitempty"`
	EndAt   string `json:"end_at,omitempty"`
	URL     string `json:"url"`
}

// --- wire types ---

type wireArtwork struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	ArtistDisplay string `json:"artist_display"`
	DateDisplay   string `json:"date_display"`
	MediumDisplay string `json:"medium_display"`
	Dimensions    string `json:"dimensions"`
	PlaceOfOrigin string `json:"place_of_origin"`
	Description   string `json:"description"`
	Thumbnail     struct {
		URL string `json:"url"`
	} `json:"thumbnail"`
}

type wireAgent struct {
	ID             int    `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	AgentTypeTitle string `json:"agent_type_title"`
	BirthDate      int    `json:"birth_date"`
	DeathDate      int    `json:"death_date"`
}

type wireExhibition struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	StartAt string `json:"aic_start_at"`
	EndAt   string `json:"aic_end_at"`
}

type wireArtworksResp struct {
	Data []wireArtwork `json:"data"`
}

type wireSingleArtworkResp struct {
	Data wireArtwork `json:"data"`
}

type wireAgentsResp struct {
	Data []wireAgent `json:"data"`
}

type wireExhibitionsResp struct {
	Data []wireExhibition `json:"data"`
}

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client talks to the Art Institute of Chicago API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client with the given configuration.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// artworkFields for list/search endpoints.
const artworkFields = "id,title,artist_display,date_display,medium_display,dimensions,place_of_origin"

// artworkDetailFields for single artwork.
const artworkDetailFields = "id,title,artist_display,date_display,medium_display,dimensions,place_of_origin,description,thumbnail"

// agentFields for agents.
const agentFields = "id,title,description,birth_date,death_date,agent_type_title"

// exhibitionFields for exhibitions.
const exhibitionFields = "id,title,status,aic_start_at,aic_end_at"

// ListArtworks lists or searches artworks. If search is non-empty, it queries
// /artworks/search?q=search; otherwise it queries /artworks.
func (c *Client) ListArtworks(ctx context.Context, search string, limit int) ([]Artwork, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var rawURL string
	if search != "" {
		params := url.Values{}
		params.Set("q", search)
		params.Set("fields", artworkFields)
		params.Set("limit", fmt.Sprintf("%d", limit))
		rawURL = c.cfg.BaseURL + "/artworks/search?" + params.Encode()
	} else {
		params := url.Values{}
		params.Set("fields", artworkFields)
		params.Set("limit", fmt.Sprintf("%d", limit))
		rawURL = c.cfg.BaseURL + "/artworks?" + params.Encode()
	}

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireArtworksResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse artworks: %w", err)
	}
	return flattenArtworks(resp.Data), nil
}

// GetArtwork fetches a single artwork by numeric ID.
func (c *Client) GetArtwork(ctx context.Context, id int) (*Artwork, error) {
	params := url.Values{}
	params.Set("fields", artworkDetailFields)
	rawURL := fmt.Sprintf("%s/artworks/%d?%s", c.cfg.BaseURL, id, params.Encode())

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireSingleArtworkResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse artwork %d: %w", id, err)
	}
	a := flattenArtwork(resp.Data)
	return &a, nil
}

// ListAgents lists or searches agents (artists/organizations). If search is
// non-empty, it queries /agents/search?q=search; otherwise /agents.
func (c *Client) ListAgents(ctx context.Context, search string, limit int) ([]Agent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var rawURL string
	if search != "" {
		params := url.Values{}
		params.Set("q", search)
		params.Set("fields", agentFields)
		params.Set("limit", fmt.Sprintf("%d", limit))
		rawURL = c.cfg.BaseURL + "/agents/search?" + params.Encode()
	} else {
		params := url.Values{}
		params.Set("fields", agentFields)
		params.Set("limit", fmt.Sprintf("%d", limit))
		rawURL = c.cfg.BaseURL + "/agents?" + params.Encode()
	}

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireAgentsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse agents: %w", err)
	}
	return flattenAgents(resp.Data), nil
}

// ListExhibitions lists museum exhibitions.
func (c *Client) ListExhibitions(ctx context.Context, limit int) ([]Exhibition, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	params := url.Values{}
	params.Set("fields", exhibitionFields)
	params.Set("limit", fmt.Sprintf("%d", limit))
	rawURL := c.cfg.BaseURL + "/exhibitions?" + params.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireExhibitionsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse exhibitions: %w", err)
	}
	return flattenExhibitions(resp.Data), nil
}

// get fetches a URL and returns the body, pacing and retrying as configured.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// --- flatten helpers ---

func flattenArtwork(w wireArtwork) Artwork {
	return Artwork{
		ID:            w.ID,
		Title:         w.Title,
		ArtistDisplay: w.ArtistDisplay,
		DateDisplay:   w.DateDisplay,
		MediumDisplay: w.MediumDisplay,
		Dimensions:    w.Dimensions,
		PlaceOfOrigin: w.PlaceOfOrigin,
		Description:   w.Description,
		ThumbnailURL:  w.Thumbnail.URL,
		URL:           fmt.Sprintf("https://%s/artworks/%d", SiteHost, w.ID),
	}
}

func flattenArtworks(ws []wireArtwork) []Artwork {
	out := make([]Artwork, len(ws))
	for i, w := range ws {
		out[i] = flattenArtwork(w)
	}
	return out
}

func flattenAgent(w wireAgent) Agent {
	return Agent{
		ID:          w.ID,
		Title:       w.Title,
		Description: w.Description,
		AgentType:   w.AgentTypeTitle,
		BirthDate:   w.BirthDate,
		DeathDate:   w.DeathDate,
		URL:         fmt.Sprintf("https://%s/artists/%d", SiteHost, w.ID),
	}
}

func flattenAgents(ws []wireAgent) []Agent {
	out := make([]Agent, len(ws))
	for i, w := range ws {
		out[i] = flattenAgent(w)
	}
	return out
}

func flattenExhibition(w wireExhibition) Exhibition {
	return Exhibition{
		ID:      w.ID,
		Title:   w.Title,
		Status:  w.Status,
		StartAt: w.StartAt,
		EndAt:   w.EndAt,
		URL:     fmt.Sprintf("https://%s/exhibitions/%d", SiteHost, w.ID),
	}
}

func flattenExhibitions(ws []wireExhibition) []Exhibition {
	out := make([]Exhibition, len(ws))
	for i, w := range ws {
		out[i] = flattenExhibition(w)
	}
	return out
}
