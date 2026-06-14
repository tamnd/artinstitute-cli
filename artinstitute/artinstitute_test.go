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
	_, _ = c.SearchArtworks(context.Background(), "monet", 5)

	if !strings.Contains(gotUA, "artinstitute-cli") {
		t.Errorf("User-Agent = %q, want it to contain artinstitute-cli", gotUA)
	}
}

// TestSearchArtworks checks that search response is parsed into Artwork fields.
func TestSearchArtworks(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":                  16568,
				"title":               "Water Lilies",
				"artist_title":        "Claude Monet",
				"date_display":        "1906",
				"medium_display":      "Oil on canvas",
				"dimensions":          "89.9 × 94.1 cm",
				"artwork_type_title":  "Painting",
				"place_of_origin":     "France",
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
	artworks, err := c.SearchArtworks(context.Background(), "monet", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(artworks) != 1 {
		t.Fatalf("got %d artworks, want 1", len(artworks))
	}
	a := artworks[0]
	if a.ID != 16568 {
		t.Errorf("ID = %d, want 16568", a.ID)
	}
	if a.Title != "Water Lilies" {
		t.Errorf("Title = %q", a.Title)
	}
	if a.Artist != "Claude Monet" {
		t.Errorf("Artist = %q, want Claude Monet", a.Artist)
	}
	if a.Date != "1906" {
		t.Errorf("Date = %q, want 1906", a.Date)
	}
	if a.Medium != "Oil on canvas" {
		t.Errorf("Medium = %q", a.Medium)
	}
	if a.Type != "Painting" {
		t.Errorf("Type = %q, want Painting", a.Type)
	}
	if a.Origin != "France" {
		t.Errorf("Origin = %q, want France", a.Origin)
	}
}

// TestSearchArtworksUsesSearchEndpoint checks that q is forwarded via /search path.
func TestSearchArtworksUsesSearchEndpoint(t *testing.T) {
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
	_, err := c.SearchArtworks(context.Background(), "monet", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "search") {
		t.Errorf("path %q should contain search", gotPath)
	}
}

// TestGetArtwork checks single artwork parsing including image_id and credit_line.
func TestGetArtwork(t *testing.T) {
	fixture := map[string]any{
		"data": map[string]any{
			"id":                 16568,
			"title":              "Water Lilies",
			"artist_title":       "Claude Monet",
			"date_display":       "1906",
			"medium_display":     "Oil on canvas",
			"dimensions":         "89.9 × 94.1 cm",
			"artwork_type_title": "Painting",
			"place_of_origin":    "France",
			"description":        "<p>Painted at Giverny...</p>",
			"image_id":           "3c27b499-af56-f0d5-93b5-a7f2f1ad5813",
			"credit_line":        "Mr. and Mrs. Martin A. Ryerson Collection",
		},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	a, err := c.GetArtwork(context.Background(), 16568)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != 16568 {
		t.Errorf("ID = %d, want 16568", a.ID)
	}
	if a.Description == "" {
		t.Error("expected non-empty description")
	}
	if a.ImageID != "3c27b499-af56-f0d5-93b5-a7f2f1ad5813" {
		t.Errorf("ImageID = %q", a.ImageID)
	}
	if a.CreditLine == "" {
		t.Error("expected non-empty credit_line")
	}
}

// TestSearchArtists checks that artist list is parsed with birth_date and null death_date.
func TestSearchArtists(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":          35577,
				"title":       "Claude Monet",
				"birth_date":  1840,
				"death_date":  nil,
				"birth_place": "France, Paris",
				"death_place": nil,
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
	artists, err := c.SearchArtists(context.Background(), "monet", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists, want 1", len(artists))
	}
	ar := artists[0]
	if ar.Name != "Claude Monet" {
		t.Errorf("Name = %q", ar.Name)
	}
	if ar.BirthDate != 1840 {
		t.Errorf("BirthDate = %d, want 1840", ar.BirthDate)
	}
	if ar.DeathDate != 0 {
		t.Errorf("DeathDate = %d, want 0 for null", ar.DeathDate)
	}
	if ar.BirthPlace != "France, Paris" {
		t.Errorf("BirthPlace = %q", ar.BirthPlace)
	}
}

// TestListExhibitions checks that exhibitions are parsed with description and dates.
func TestListExhibitions(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":                9999,
				"title":             "John Massey",
				"short_description": "A show about design.",
				"status":            "Closed",
				"aic_start_at":      "2024-01-15T00:00:00-06:00",
				"aic_end_at":        "2024-04-07T00:00:00-05:00",
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
	exs, err := c.ListExhibitions(context.Background(), "", 5)
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
	if ex.Description != "A show about design." {
		t.Errorf("Description = %q", ex.Description)
	}
	if ex.StartAt == "" {
		t.Error("expected non-empty start_at")
	}
	if ex.EndAt == "" {
		t.Error("expected non-empty end_at")
	}
}

// TestListExhibitionsStatusFilter checks client-side status filtering.
func TestListExhibitionsStatusFilter(t *testing.T) {
	fixture := map[string]any{
		"data": []any{
			map[string]any{
				"id":                1,
				"title":             "Open Show",
				"short_description": "Running now.",
				"status":            "Open",
				"aic_start_at":      "2024-01-01T00:00:00-06:00",
				"aic_end_at":        "2024-12-31T00:00:00-06:00",
			},
			map[string]any{
				"id":                2,
				"title":             "Closed Show",
				"short_description": "Past show.",
				"status":            "Closed",
				"aic_start_at":      "2023-01-01T00:00:00-06:00",
				"aic_end_at":        "2023-06-01T00:00:00-05:00",
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
	exs, err := c.ListExhibitions(context.Background(), "Open", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(exs) != 1 {
		t.Fatalf("got %d exhibitions after filter, want 1", len(exs))
	}
	if exs[0].Status != "Open" {
		t.Errorf("Status = %q, want Open", exs[0].Status)
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
	_, err := c.SearchArtworks(context.Background(), "monet", 5)
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

// TestGetArtistByNumericQuery checks that a numeric query fetches a single artist by ID.
func TestGetArtistByNumericQuery(t *testing.T) {
	fixture := map[string]any{
		"data": map[string]any{
			"id":          35577,
			"title":       "Claude Monet",
			"birth_date":  1840,
			"death_date":  1926,
			"birth_place": "France, Paris",
		},
	}
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := json.Marshal(fixture)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	artists, err := c.SearchArtists(context.Background(), "35577", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists, want 1", len(artists))
	}
	if artists[0].DeathDate != 1926 {
		t.Errorf("DeathDate = %d, want 1926", artists[0].DeathDate)
	}
	// Should hit /artists/35577, not /artists?q=...
	if !strings.Contains(gotPath, "35577") {
		t.Errorf("path %q should contain 35577", gotPath)
	}
}
