package artinstitute

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "artinstitute" {
		t.Errorf("Scheme = %q, want artinstitute", info.Scheme)
	}
	if len(info.Hosts) == 0 {
		t.Error("Hosts is empty")
	}
	if info.Identity.Binary != "artinstitute" {
		t.Errorf("Identity.Binary = %q, want artinstitute", info.Identity.Binary)
	}
}

func TestClassifyNumeric(t *testing.T) {
	typ, id, err := Domain{}.Classify("27992")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "artwork" {
		t.Errorf("type = %q, want artwork", typ)
	}
	if id != "27992" {
		t.Errorf("id = %q, want 27992", id)
	}
}

func TestClassifyQuery(t *testing.T) {
	typ, id, err := Domain{}.Classify("monet")
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if typ != "query" {
		t.Errorf("type = %q, want query", typ)
	}
	if id != "monet" {
		t.Errorf("id = %q, want monet", id)
	}
}

func TestLocateArtwork(t *testing.T) {
	got, err := Domain{}.Locate("artwork", "27992")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	want := "https://www.artic.edu/artworks/27992"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateQuery(t *testing.T) {
	got, err := Domain{}.Locate("query", "monet")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	want := "https://www.artic.edu/collection?q=monet"
	if got != want {
		t.Errorf("Locate = %q, want %q", got, want)
	}
}

func TestLocateUnknown(t *testing.T) {
	_, err := Domain{}.Locate("bogus", "123")
	if err == nil {
		t.Error("expected error for unknown resource type")
	}
}

func TestFlattenArtwork(t *testing.T) {
	w := wireArtwork{
		ID:               16568,
		Title:            "Water Lilies",
		ArtistTitle:      "Claude Monet",
		DateDisplay:      "1906",
		MediumDisplay:    "Oil on canvas",
		Dimensions:       "89.9 × 94.1 cm",
		ArtworkTypeTitle: "Painting",
		PlaceOfOrigin:    "France",
		Description:      "Famous pond.",
		CreditLine:       "Mr. and Mrs. Martin A. Ryerson Collection",
		ImageID:          "3c27b499-af56-f0d5-93b5-a7f2f1ad5813",
	}
	a := flattenArtwork(w)
	if a.ID != 16568 {
		t.Errorf("ID = %d, want 16568", a.ID)
	}
	if a.Artist != "Claude Monet" {
		t.Errorf("Artist = %q", a.Artist)
	}
	if a.Date != "1906" {
		t.Errorf("Date = %q", a.Date)
	}
	if a.Medium != "Oil on canvas" {
		t.Errorf("Medium = %q", a.Medium)
	}
	if a.Type != "Painting" {
		t.Errorf("Type = %q", a.Type)
	}
	if a.Origin != "France" {
		t.Errorf("Origin = %q", a.Origin)
	}
	if a.ImageID != "3c27b499-af56-f0d5-93b5-a7f2f1ad5813" {
		t.Errorf("ImageID = %q", a.ImageID)
	}
	if a.CreditLine == "" {
		t.Error("CreditLine should not be empty")
	}
}

func TestFlattenArtistNullDeath(t *testing.T) {
	w := wireArtist{
		ID:        35577,
		Title:     "Claude Monet",
		BirthDate: 1840,
		DeathDate: nil, // null in JSON
		BirthPlace: "France, Paris",
	}
	a := flattenArtist(w)
	if a.Name != "Claude Monet" {
		t.Errorf("Name = %q", a.Name)
	}
	if a.BirthDate != 1840 {
		t.Errorf("BirthDate = %d, want 1840", a.BirthDate)
	}
	if a.DeathDate != 0 {
		t.Errorf("DeathDate = %d, want 0 for null", a.DeathDate)
	}
	if a.BirthPlace != "France, Paris" {
		t.Errorf("BirthPlace = %q", a.BirthPlace)
	}
}

func TestFlattenArtistWithDeath(t *testing.T) {
	death := 1926
	w := wireArtist{
		ID:        35577,
		Title:     "Claude Monet",
		BirthDate: 1840,
		DeathDate: &death,
		BirthPlace: "France, Paris",
	}
	a := flattenArtist(w)
	if a.DeathDate != 1926 {
		t.Errorf("DeathDate = %d, want 1926", a.DeathDate)
	}
}

func TestFlattenExhibition(t *testing.T) {
	w := wireExhibition{
		ID:               9999,
		Title:            "John Massey",
		ShortDescription: "A show about design.",
		Status:           "Closed",
		StartAt:          "2024-01-15T00:00:00-06:00",
		EndAt:            "2024-04-07T00:00:00-05:00",
	}
	ex := flattenExhibition(w)
	if ex.ID != 9999 {
		t.Errorf("ID = %d, want 9999", ex.ID)
	}
	if ex.Description != "A show about design." {
		t.Errorf("Description = %q", ex.Description)
	}
	if ex.Status != "Closed" {
		t.Errorf("Status = %q", ex.Status)
	}
	if ex.StartAt != "2024-01-15T00:00:00-06:00" {
		t.Errorf("StartAt = %q", ex.StartAt)
	}
}
