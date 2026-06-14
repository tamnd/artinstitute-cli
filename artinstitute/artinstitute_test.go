package artinstitute_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/artinstitute-cli/artinstitute"
)

func newTestClient(ts *httptest.Server) *artinstitute.Client {
	cfg := artinstitute.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return artinstitute.NewClient(cfg)
}

// TestUserAgent checks that every request carries artinstitute-cli in User-Agent.
func TestUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		resp := map[string]any{"data": []any{}}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, _ = c.ListArtworks(context.Background(), "", 5)

	if !strings.Contains(gotUA, "artinstitute-cli") {
		t.Errorf("User-Agent = %q, want it to contain artinstitute-cli", gotUA)
	}
}

// TestListArtworks checks that list response is parsed correctly.
func TestListArtworks(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":             27992,
				"title":          "A Sunday on La Grande Jatte",
				"artist_display": "Georges Seurat\nFrench, 1859-1891",
				"date_display":   "1884-86",
				"medium_display": "Oil on canvas",
				"dimensions":     "207.5 x 308.1 cm",
				"place_of_origin": "France",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	artworks, err := c.ListArtworks(context.Background(), "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(artworks) != 1 {
		t.Fatalf("got %d artworks, want 1", len(artworks))
	}
	a := artworks[0]
	if a.ID != 27992 {
		t.Errorf("ID = %d, want 27992", a.ID)
	}
	if a.Title != "A Sunday on La Grande Jatte" {
		t.Errorf("Title = %q", a.Title)
	}
	if !strings.Contains(a.URL, "27992") {
		t.Errorf("URL %q should contain 27992", a.URL)
	}
	if a.ArtistDisplay == "" {
		t.Error("ArtistDisplay should not be empty")
	}
}

// TestSearchArtworks checks that the q param is forwarded when searching.
func TestSearchArtworks(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		resp := map[string]any{"data": []any{}}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.ListArtworks(context.Background(), "monet", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search") {
		t.Errorf("path %q should contain search", gotPath)
	}
}

// TestGetArtwork checks single artwork parsing including thumbnail_url.
func TestGetArtwork(t *testing.T) {
	fixture := map[string]any{
		"data": map[string]any{
			"id":             27992,
			"title":          "A Sunday on La Grande Jatte",
			"artist_display": "Georges Seurat",
			"date_display":   "1884-86",
			"medium_display": "Oil on canvas",
			"description":    "<p>Painted in dots...</p>",
			"thumbnail": map[string]any{
				"url": "https://www.artic.edu/iiif/2/abc/full/843,/0/default.jpg",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	a, err := c.GetArtwork(context.Background(), 27992)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != 27992 {
		t.Errorf("ID = %d, want 27992", a.ID)
	}
	if a.Description == "" {
		t.Error("expected non-empty description")
	}
	if a.ThumbnailURL == "" {
		t.Error("expected non-empty thumbnail_url")
	}
}

// TestListAgents checks that agents list is parsed with birth_date and agent_type.
func TestListAgents(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":               33736,
				"title":            "Georges Seurat",
				"birth_date":       1859,
				"death_date":       1891,
				"agent_type_title": "Individual",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	agents, err := c.ListAgents(context.Background(), "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	ag := agents[0]
	if ag.Title != "Georges Seurat" {
		t.Errorf("Title = %q", ag.Title)
	}
	if ag.BirthDate != 1859 {
		t.Errorf("BirthDate = %d, want 1859", ag.BirthDate)
	}
	if ag.AgentType != "Individual" {
		t.Errorf("AgentType = %q, want Individual", ag.AgentType)
	}
}

// TestListExhibitions checks that exhibitions are parsed with status and dates.
func TestListExhibitions(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":          9526,
				"title":       "Apostles of Beauty",
				"status":      "Closed",
				"aic_start_at": "2009-10-31T00:00:00.000Z",
				"aic_end_at":  "2010-01-31T00:00:00.000Z",
			},
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	exs, err := c.ListExhibitions(context.Background(), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(exs) != 1 {
		t.Fatalf("got %d exhibitions, want 1", len(exs))
	}
	ex := exs[0]
	if ex.Status != "Closed" {
		t.Errorf("Status = %q, want Closed", ex.Status)
	}
	if ex.StartAt == "" {
		t.Error("expected non-empty start_at")
	}
}

// TestRetryOn503 checks that the client retries on 503 and succeeds eventually.
func TestRetryOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		resp := map[string]any{"data": []any{}}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	cfg := artinstitute.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := artinstitute.NewClient(cfg)

	start := time.Now()
	_, err := c.ListArtworks(context.Background(), "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}
