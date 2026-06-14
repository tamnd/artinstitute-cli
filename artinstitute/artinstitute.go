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
	"strconv"
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
	ID          int    `kit:"id" json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Date        string `json:"date"`
	Medium      string `json:"medium"`
	Dimensions  string `json:"dimensions"`
	Type        string `json:"type"`
	Origin      string `json:"origin"`
	Description string `json:"description"`
	CreditLine  string `json:"credit_line"`
	ImageID     string `json:"image_id"`
}

// Artist holds the public data about an artist or organization.
type Artist struct {
	ID          int    `kit:"id" json:"id"`
	Name        string `json:"name"`
	BirthDate   int    `json:"birth_date"`
	DeathDate   int    `json:"death_date"`
	BirthPlace  string `json:"birth_place"`
	DeathPlace  string `json:"death_place"`
	Description string `json:"description"`
}

// Exhibition holds the public data about a museum exhibition.
type Exhibition struct {
	ID          int    `kit:"id" json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	StartAt     string `json:"start_at"`
	EndAt       string `json:"end_at"`
}

// --- wire types ---

type wireArtwork struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	ArtistTitle      string `json:"artist_title"`
	DateDisplay      string `json:"date_display"`
	MediumDisplay    string `json:"medium_display"`
	Dimensions       string `json:"dimensions"`
	ArtworkTypeTitle string `json:"artwork_type_title"`
	PlaceOfOrigin    string `json:"place_of_origin"`
	Description      string `json:"description"`
	ImageID          string `json:"image_id"`
	CreditLine       string `json:"credit_line"`
}

// wireArtist uses *int for DeathDate to handle null JSON values.
type wireArtist struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	BirthDate   int     `json:"birth_date"`
	DeathDate   *int    `json:"death_date"`
	BirthPlace  string  `json:"birth_place"`
	DeathPlace  string  `json:"death_place"`
	Description string  `json:"description"`
}

type wireExhibition struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	ShortDescription string `json:"short_description"`
	Status           string `json:"status"`
	StartAt          string `json:"aic_start_at"`
	EndAt            string `json:"aic_end_at"`
}

type wireArtworksResp struct {
	Data []wireArtwork `json:"data"`
}

type wireSingleArtworkResp struct {
	Data wireArtwork `json:"data"`
}

type wireArtistsResp struct {
	Data []wireArtist `json:"data"`
}

type wireSingleArtistResp struct {
	Data wireArtist `json:"data"`
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
		Rate:      500 * time.Millisecond,
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

// field sets for each resource type.
const artworkSearchFields = "id,title,artist_title,date_display,medium_display,dimensions,artwork_type_title,place_of_origin"
const artworkDetailFields = "id,title,artist_title,date_display,medium_display,dimensions,artwork_type_title,place_of_origin,description,image_id,credit_line"
const artistFields = "id,title,birth_date,death_date,birth_place,death_place,description"
const exhibitionFields = "id,title,short_description,status,aic_start_at,aic_end_at"

// SearchArtworks searches artworks by query string via Elasticsearch.
func (c *Client) SearchArtworks(ctx context.Context, query string, limit int) ([]Artwork, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("fields", artworkSearchFields)
	params.Set("limit", strconv.Itoa(limit))
	rawURL := c.cfg.BaseURL + "/artworks/search?" + params.Encode()

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

// SearchArtists searches artists by name query string.
// If query is numeric, it fetches a single artist by ID.
func (c *Client) SearchArtists(ctx context.Context, query string, limit int) ([]Artist, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// numeric query → single artist by ID
	if id, err := strconv.Atoi(query); err == nil {
		artist, err := c.GetArtist(ctx, id)
		if err != nil {
			return nil, err
		}
		return []Artist{*artist}, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("fields", artistFields)
	params.Set("limit", strconv.Itoa(limit))
	rawURL := c.cfg.BaseURL + "/artists?" + params.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireArtistsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse artists: %w", err)
	}
	return flattenArtists(resp.Data), nil
}

// GetArtist fetches a single artist by numeric ID.
func (c *Client) GetArtist(ctx context.Context, id int) (*Artist, error) {
	params := url.Values{}
	params.Set("fields", artistFields)
	rawURL := fmt.Sprintf("%s/artists/%d?%s", c.cfg.BaseURL, id, params.Encode())

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireSingleArtistResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse artist %d: %w", id, err)
	}
	a := flattenArtist(resp.Data)
	return &a, nil
}

// ListExhibitions lists museum exhibitions, optionally filtered by status.
func (c *Client) ListExhibitions(ctx context.Context, status string, limit int) ([]Exhibition, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	params := url.Values{}
	params.Set("fields", exhibitionFields)
	params.Set("limit", strconv.Itoa(limit))
	if status != "" {
		params.Set("query[term][status]", status)
	}
	rawURL := c.cfg.BaseURL + "/exhibitions?" + params.Encode()

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var resp wireExhibitionsResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse exhibitions: %w", err)
	}
	out := flattenExhibitions(resp.Data)

	// client-side status filter if the API doesn't support it
	if status != "" {
		filtered := out[:0]
		for _, ex := range out {
			if ex.Status == status {
				filtered = append(filtered, ex)
			}
		}
		out = filtered
	}
	return out, nil
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
		ID:          w.ID,
		Title:       w.Title,
		Artist:      w.ArtistTitle,
		Date:        w.DateDisplay,
		Medium:      w.MediumDisplay,
		Dimensions:  w.Dimensions,
		Type:        w.ArtworkTypeTitle,
		Origin:      w.PlaceOfOrigin,
		Description: w.Description,
		CreditLine:  w.CreditLine,
		ImageID:     w.ImageID,
	}
}

func flattenArtworks(ws []wireArtwork) []Artwork {
	out := make([]Artwork, len(ws))
	for i, w := range ws {
		out[i] = flattenArtwork(w)
	}
	return out
}

func flattenArtist(w wireArtist) Artist {
	death := 0
	if w.DeathDate != nil {
		death = *w.DeathDate
	}
	return Artist{
		ID:          w.ID,
		Name:        w.Title,
		BirthDate:   w.BirthDate,
		DeathDate:   death,
		BirthPlace:  w.BirthPlace,
		DeathPlace:  w.DeathPlace,
		Description: w.Description,
	}
}

func flattenArtists(ws []wireArtist) []Artist {
	out := make([]Artist, len(ws))
	for i, w := range ws {
		out[i] = flattenArtist(w)
	}
	return out
}

func flattenExhibition(w wireExhibition) Exhibition {
	return Exhibition{
		ID:          w.ID,
		Title:       w.Title,
		Description: w.ShortDescription,
		Status:      w.Status,
		StartAt:     w.StartAt,
		EndAt:       w.EndAt,
	}
}

func flattenExhibitions(ws []wireExhibition) []Exhibition {
	out := make([]Exhibition, len(ws))
	for i, w := range ws {
		out[i] = flattenExhibition(w)
	}
	return out
}
