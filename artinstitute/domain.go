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

	kit.Handle(app, kit.OpMeta{Name: "artworks", Group: "read", List: true,
		Summary: "List or search artworks (--search, --limit flags)"}, listArtworks)

	kit.Handle(app, kit.OpMeta{Name: "artwork", Group: "read", Single: true,
		Summary: "Get a single artwork by ID",
		Args:    []kit.Arg{{Name: "id", Help: "artwork numeric ID"}}}, getArtwork)

	kit.Handle(app, kit.OpMeta{Name: "artists", Group: "read", List: true,
		Summary: "List or search artists/agents (--search, --limit flags)"}, listArtists)

	kit.Handle(app, kit.OpMeta{Name: "exhibitions", Group: "read", List: true,
		Summary: "List exhibitions (--limit flag)"}, listExhibitions)
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

type artworksInput struct {
	Search string  `kit:"flag" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type artworkInput struct {
	ID     string  `kit:"arg" help:"artwork numeric ID"`
	Client *Client `kit:"inject"`
}

type artistsInput struct {
	Search string  `kit:"flag" help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type exhibitionsInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listArtworks(ctx context.Context, in artworksInput, emit func(*Artwork) error) error {
	items, err := in.Client.ListArtworks(ctx, in.Search, in.Limit)
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
	id, err := strconv.Atoi(in.ID)
	if err != nil {
		return errs.Usage("artwork id must be a number, got %q", in.ID)
	}
	item, err := in.Client.GetArtwork(ctx, id)
	if err != nil {
		return err
	}
	return emit(item)
}

func listArtists(ctx context.Context, in artistsInput, emit func(*Agent) error) error {
	items, err := in.Client.ListAgents(ctx, in.Search, in.Limit)
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
	items, err := in.Client.ListExhibitions(ctx, in.Limit)
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
func (Domain) Classify(input string) (string, string, error) {
	return "artwork", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "artwork":
		return fmt.Sprintf("https://%s/artworks/%s", SiteHost, id), nil
	case "artist":
		return fmt.Sprintf("https://%s/artists/%s", SiteHost, id), nil
	case "exhibition":
		return fmt.Sprintf("https://%s/exhibitions/%s", SiteHost, id), nil
	default:
		return "", errs.Usage("artinstitute has no resource type %q", t)
	}
}
