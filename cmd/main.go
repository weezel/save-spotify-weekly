package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"weezel/savespotifyweekly/pkg/wspotify"

	"github.com/zmb3/spotify/v2"
)

const localHostURI = "http://localhost:8080/callback"

// spotifyCli = spfy.NewClient(
// 	os.Getenv("SPOTIFY_ID"),
// 	os.Getenv("SPOTIFY_SECRET"),
// 	localHostURI,
// 	&spotify.Client{},
// )
// oauthConfig = &oauth2.Config{
// 	ClientID:     os.Getenv("SPOTIFY_ID"),
// 	ClientSecret: os.Getenv("SPOTIFY_SECRET"),
// 	Endpoint: oauth2.Endpoint{
// 		AuthURL:   spotifyauth.AuthURL,
// 		TokenURL:  spotifyauth.TokenURL,
// 		AuthStyle: oauth2.AuthStyleAutoDetect,
// 	},
// 	RedirectURL: localHostURI,
// 	Scopes: []string{
// 		spotifyauth.ScopePlaylistReadPrivate,
// 		spotifyauth.ScopePlaylistModifyPrivate,
// 	},
// }

func main() {
	ctx := context.Background()

	spotifyCli := wspotify.NewClient(
		cmp.Or(os.Getenv("SPOTIFY_ID"), "EMPTY"),
		cmp.Or(os.Getenv("SPOTIFY_SECRET"), "EMPTY_TOO"),
		localHostURI,
		&spotify.Client{},
	)
	// If we don't have token file saved yet ask user to grant the needed permissions
	if _, err := os.Stat("token.json"); err != nil && errors.Is(err, os.ErrNotExist) {
		spotifyCli.StartWebserver(ctx)
		select {
		case <-spotifyCli.ClientInitDone:
			log.Println("Spotify client initialized")
		}
	}
	//  else {
	// 	// go nonInteractiveAuth(ctx)
	// }

	user, err := spotifyCli.GetClient().CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)

	// Retrieve the current user's playlists
	discoverWeekly := spotifyCli.GetDiscoverWeeklyPlaylist(ctx)
	if discoverWeekly == nil {
		log.Fatal("Failed to find Discover weekly playlist")
	}

	fullPlaylist, err := spotifyCli.GetClient().GetPlaylist(ctx, discoverWeekly.ID)
	if err != nil {
		panic(err)
	}

	wspotify.PrintSongsInPlaylist(fullPlaylist)
}
