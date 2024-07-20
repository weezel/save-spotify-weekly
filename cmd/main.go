package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"weezel/savespotifyweekly/pkg/wspotify"

	"github.com/zmb3/spotify/v2"
)

var (
	localHostURI         = cmp.Or(os.Getenv("SPOTIFY_CALLBACK_URL"), "http://localhost:8080/callback")
	spotifyID            = os.Getenv("SPOTIFY_ID")
	spotifySecret        = os.Getenv("SPOTIFY_SECRET")
	archivedPlaylistName = os.Getenv("PLAYLIST_NAME")
)

func saveDiscoverWeekly(ctx context.Context, spotifyCli *wspotify.Spotify) {
	user, err := spotifyCli.GetClient().CurrentUser(ctx)
	if err != nil {
		log.Fatalf("Failed to get user: %v\n", err)
	}
	fmt.Printf("You are logged in as: %s\n", user.ID)

	if archivedPlaylistName == "" {
		year, week := time.Now().ISOWeek()
		archivedPlaylistName = fmt.Sprintf("Archived discover weekly %d-%d", year, week)
	}
	// Does archived playlist already exist?
	existingPlaylists, err := spotifyCli.GetClient().GetPlaylistsForUser(ctx, user.ID)
	if err != nil {
		log.Fatalf("Failed retrieving playlists: %v", err)
	}
	for _, pl := range existingPlaylists.Playlists {
		if pl.Name == archivedPlaylistName {
			log.Printf("Playlist %q already exists, exiting...\n", archivedPlaylistName)
			return
		}
	}

	// Retrieve the current user's playlists
	discoverWeekly := spotifyCli.GetDiscoverWeeklyPlaylist(ctx)
	if discoverWeekly == nil {
		log.Fatal("Failed to find Discover weekly playlist, it was empty")
	}
	dwPlaylist, err := spotifyCli.GetClient().GetPlaylist(ctx, discoverWeekly.ID)
	if err != nil {
		log.Fatalf("Failed to get discover weekly playlist: %v\n", err)
	}
	dwTracks := []spotify.ID{}
	for _, t := range dwPlaylist.Tracks.Tracks {
		dwTracks = append(dwTracks, t.Track.ID)
	}
	err = spotifyCli.SaveCurrentWeeksPlaylist(
		ctx,
		user.ID,
		archivedPlaylistName,
		time.Now(),
		dwTracks...,
	)
	if err != nil {
		log.Fatalf("Failed to archive discover weekly playlist: %v\n", err)
	}

	log.Printf("Playlist %q saved\n", archivedPlaylistName)

	wspotify.PrintSongsInPlaylist(dwPlaylist)
}

func main() {
	ctx := context.Background()

	if spotifyID == "" {
		fmt.Println("SPOTIFY_ID is empty")
		os.Exit(1)
	}
	if spotifySecret == "" {
		fmt.Println("SPOTIFY_SECRET is empty")
		os.Exit(1)
	}

	spotifyCli := wspotify.NewClient(
		spotifyID,
		spotifySecret,
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

		fmt.Println("Run the program again now that tokens are stored in a file")

		return
	}

	if err := spotifyCli.NonInteractiveAuth(ctx); err != nil {
		log.Panicf("Non interactive login paniced: %v\n", err)
	}

	saveDiscoverWeekly(ctx, spotifyCli)
}
