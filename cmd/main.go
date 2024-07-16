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

func main() {
	ctx := context.Background()

	spotifyCli := wspotify.NewClient(
		cmp.Or(os.Getenv("SPOTIFY_ID"), "EMPTY_ID"),
		cmp.Or(os.Getenv("SPOTIFY_SECRET"), "EMPTY_SECRET"),
		localHostURI,
		&spotify.Client{},
	)
	// If we don't have token file saved yet ask user to grant the needed permissions
	if _, err := os.Stat("token.json"); err != nil && errors.Is(err, os.ErrNotExist) {
		// Run in a func so it's possible to close the web server when returning
		func() {
			spotifyCli.InteractiveAuth(ctx)
			defer spotifyCli.ShutdownWebserver(ctx)

			<-spotifyCli.ClientInitDone
			log.Println("Spotify client initialized")
		}()
	} else {
		if err = spotifyCli.NonInteractiveAuth(ctx); err != nil {
			log.Panicf("Non interactive login paniced: %v\n", err)
		}
	}

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
