package auth

import (
	"context"
	"fmt"
	"log"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// Player represents a game player with Spotify data
type Player struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SpotifyID   string   `json:"spotify_id"`
	AccessToken string   `json:"-"`
	TopTracks   []Track  `json:"-"`
}

// Track represents a Spotify track
type Track struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Artists    []string `json:"artists"`
	Rank       int      `json:"rank"`
	URI        string   `json:"uri"`
	ImageURL   string   `json:"image_url"`
	PreviewURL string   `json:"preview_url"`
}

// SpotifyAuthenticator handles Spotify OAuth
type SpotifyAuthenticator struct {
	auth *spotifyauth.Authenticator
}

// NewSpotifyAuthenticator creates a new authenticator
func NewSpotifyAuthenticator(clientID, clientSecret, redirectURI string) *SpotifyAuthenticator {
	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(spotifyauth.ScopeUserTopRead),
	)

	return &SpotifyAuthenticator{
		auth: auth,
	}
}

// GetAuthURL returns the Spotify authorization URL
func (sa *SpotifyAuthenticator) GetAuthURL(state string) string {
	return sa.auth.AuthURL(state)
}

// ExchangeCode exchanges authorization code for access token
func (sa *SpotifyAuthenticator) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return sa.auth.Exchange(ctx, code)
}

// NewClient creates a new Spotify client with the given token
func (sa *SpotifyAuthenticator) NewClient(ctx context.Context, token *oauth2.Token) *spotify.Client {
	httpClient := sa.auth.Client(ctx, token)
	return spotify.New(httpClient)
}

// FetchPlayerInfo retrieves the current user's profile information
func FetchPlayerInfo(ctx context.Context, client *spotify.Client) (*Player, error) {
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	player := &Player{
		ID:        user.ID,
		Name:      user.DisplayName,
		SpotifyID: user.ID,
	}

	if player.Name == "" {
		player.Name = "Player " + user.ID[:4]
	}

	return player, nil
}

// FetchPlayerTopTracks retrieves the user's top 50 tracks from the past 6 months
func FetchPlayerTopTracks(ctx context.Context, client *spotify.Client) ([]Track, error) {
	topTracksPage, err := client.CurrentUsersTopTracks(
		ctx,
		spotify.Limit(50),
		spotify.Timerange(spotify.MediumTermRange),
	)
	if err != nil {
		log.Printf("Error fetching top tracks: %v", err)
		return nil, fmt.Errorf("failed to fetch top tracks: %w", err)
	}

	tracks := make([]Track, len(topTracksPage.Tracks))
	for i, track := range topTracksPage.Tracks {
		tracks[i] = Track{
			ID:         string(track.ID),
			Name:       track.Name,
			Artists:    getArtistNames(track.Artists),
			Rank:       i + 1,
			URI:        string(track.URI),
			ImageURL:   getAlbumImage(track.Album),
			PreviewURL: track.PreviewURL,
		}
	}

	return tracks, nil
}

func getArtistNames(artists []spotify.SimpleArtist) []string {
	names := make([]string, len(artists))
	for i, artist := range artists {
		names[i] = artist.Name
	}
	return names
}

func getAlbumImage(album spotify.SimpleAlbum) string {
	if len(album.Images) > 0 {
		return album.Images[0].URL
	}
	return ""
}