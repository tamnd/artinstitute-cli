package artinstitute

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the artinstitute driver.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "artinstitute",
		Hosts:  []string{Host, SiteHost},
		Identity: kit.Identity{
			Binary: "artinstitute",
			Short:  "A command line for the Art Institute of Chicago.",
			Long: `A command line for the Art Institute of Chicago public API.

artinstitute reads artwork, artist, and exhibition data from api.artic.edu
over HTTPS, shapes it into clean records, and prints output that pipes into
the rest of your tools. No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/artinstitute-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", List: true,
		Summary: "Search artworks by query",
		Args:    []kit.Arg{{Name: "query", Help: "search query e.g. monet"}}}, searchArtworks)

	kit.Handle(app, kit.OpMeta{Name: "artwork", Group: "read", Single: true,
		Summary: "Get a single artwork by ID",
		Args:    []kit.Arg{{Name: "id", Help: "artwork numeric ID"}}}, getArtwork)

	kit.Handle(app, kit.OpMeta{Name: "artist", Group: "read", List: true,
		Summary: "Search artists by name or get by ID",
		Args:    []kit.Arg{{Name: "query", Help: "artist name or ID"}}}, searchArtists)

	kit.Handle(app, kit.OpMeta{Name: "exhibitions", Group: "read", List: true,
		Summary: "List museum exhibitions"}, listExhibitions)
}

// newClient builds the Client from kit config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- input structs ---

type searchInput struct {
	Query  string  `kit:"arg" help:"search query e.g. monet"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type artworkInput struct {
	ID     int     `kit:"arg" help:"artwork ID"`
	Client *Client `kit:"inject"`
}

type artistInput struct {
	Query  string  `kit:"arg" help:"artist name or ID"`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

type exhibitionsInput struct {
	Status string  `kit:"flag" help:"filter by status: Open|Closed" default:""`
	Limit  int     `kit:"flag,inherit" help:"max results" default:"10"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchArtworks(ctx context.Context, in searchInput, emit func(*Artwork) error) error {
	items, err := in.Client.SearchArtworks(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func getArtwork(ctx context.Context, in artworkInput, emit func(*Artwork) error) error {
	item, err := in.Client.GetArtwork(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(item)
}

func searchArtists(ctx context.Context, in artistInput, emit func(*Artist) error) error {
	items, err := in.Client.SearchArtists(ctx, in.Query, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listExhibitions(ctx context.Context, in exhibitionsInput, emit func(*Exhibition) error) error {
	items, err := in.Client.ListExhibitions(ctx, in.Status, in.Limit)
	if err != nil {
		return err
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

// Classify turns a URL or ID string into (type, id).
// Numeric input is classified as "artwork"; anything else as "query".
func (Domain) Classify(input string) (string, string, error) {
	if _, err := strconv.Atoi(input); err == nil {
		return "artwork", input, nil
	}
	return "query", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "artwork":
		return fmt.Sprintf("https://%s/artworks/%s", SiteHost, id), nil
	case "query":
		return fmt.Sprintf("https://%s/collection?q=%s", SiteHost, id), nil
	case "artist":
		return fmt.Sprintf("https://%s/artists/%s", SiteHost, id), nil
	case "exhibition":
		return fmt.Sprintf("https://%s/exhibitions/%s", SiteHost, id), nil
	default:
		return "", errs.Usage("artinstitute has no resource type %q", t)
	}
}
