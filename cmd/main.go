package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"weezel/savespotifyweekly/pkg/wspotify"

	"github.com/zmb3/spotify/v2"
)

var (
	localHostURI         = cmp.Or(os.Getenv("SPOTIFY_CALLBACK_URL"), "http://localhost:8080/callback")
	spotifyID            = cmp.Or(os.Getenv("SPOTIFY_ID"), "EMPTY_ID")
	spotifySecret        = cmp.Or(os.Getenv("SPOTIFY_SECRET"), "EMPTY_SECRET")
	archivedPlaylistName = os.Getenv("PLAYLIST_NAME")
)

func normalUsage(ctx context.Context, spotifyCli *wspotify.Spotify) {
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

	dwPlaylist, err := spotifyCli.GetClient().GetPlaylist(ctx, discoverWeekly.ID)
	if err != nil {
		log.Fatal(err)
	}
	dwTracks := []spotify.ID{}
	for _, t := range dwPlaylist.Tracks.Tracks {
		dwTracks = append(dwTracks, t.Track.ID)
	}

	if archivedPlaylistName == "" {
		year, week := time.Now().ISOWeek()
		archivedPlaylistName = fmt.Sprintf("Archived discover weekly %d-%d", year, week)
	}
	err = spotifyCli.SaveCurrentWeeksPlaylist(
		ctx,
		user.ID,
		archivedPlaylistName,
		time.Now(),
		dwTracks...,
	)
	if err != nil {
		log.Fatal(err)
	}

	wspotify.PrintSongsInPlaylist(dwPlaylist)
}

func main() {
	ctx := context.Background()

	var flagRefreshOnly bool
	flag.BoolVar(&flagRefreshOnly, "r", false, "Only refresh the existing token")
	flag.Parse()

	spotifyCli := wspotify.NewClient(
		spotifyID,
		spotifySecret,
		localHostURI,
		&spotify.Client{},
	)

	if flagRefreshOnly {
		if err := spotifyCli.NonInteractiveAuth(ctx); err != nil {
			log.Panicf("Non interactive login paniced: %v\n", err)
		}
		return
	}

	normalUsage(ctx, spotifyCli)
}
