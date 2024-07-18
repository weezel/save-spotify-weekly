package wspotify

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

const tokenFilename = "token.json"

// More info about the authorization can be found from here
// https://developer.spotify.com/documentation/web-api/concepts/authorization
// and https://developer.spotify.com/documentation/web-api/tutorials/code-flow

type Option func(*Spotify)

func WithAuth(auth spotifyauth.Authenticator) Option {
	return func(s *Spotify) {
		s.auth = &auth
	}
}

type Spotify struct {
	client         *spotify.Client
	ClientInitDone chan bool
	token          *oauth2.Token
	auth           *spotifyauth.Authenticator
	webserver      *http.Server
	clientID       string
	clientSecret   string
	redirectURL    string
	codeVerifier   string
	codeChallenge  string
	state          string
}

func NewClient(clientID, clientSecret, redirectURL string, client *spotify.Client) *Spotify {
	// Env vars CLIENT_ID and CLIENT_SECRET are automatically read by the spotify auth constructor.
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURL),
		spotifyauth.WithScopes(
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
	)

	// codeVerifier := GenerateRandomString(128)
	codeVerifier := "jormaPulkkisenLSDpyorallamennaanmetsaanhuuuii09282"
	return &Spotify{
		codeVerifier:   codeVerifier,
		ClientInitDone: make(chan bool, 1),
		codeChallenge:  GenerateCodeChallenge(codeVerifier),
		auth:           auth,
		client:         client,
		clientID:       clientID,
		clientSecret:   clientSecret,
		state:          GenerateRandomString(16),
		redirectURL:    redirectURL,
	}
}

func (s *Spotify) GetClient() *spotify.Client {
	return s.client
}

func (s *Spotify) refreshAccessToken(ctx context.Context) error {
	newToken, err := s.auth.RefreshToken(ctx, s.token)
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	if s.token.AccessToken != newToken.AccessToken {
		s.token = newToken
	}

	tokenFile, err := os.OpenFile(tokenFilename, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open token file: %w", err)
	}
	defer tokenFile.Close()

	if err = writeTokenToDisk(s.token, tokenFile); err != nil {
		return fmt.Errorf("write to token file: %w", err)
	}

	return nil
}

func (s *Spotify) GetDiscoverWeeklyPlaylist(ctx context.Context) *spotify.SimplePlaylist {
	res, err := s.client.Search(ctx, "discover weekly", spotify.SearchTypePlaylist)
	if err != nil {
		panic(err)
	}

	var discoverWeekly *spotify.SimplePlaylist
	for _, r := range res.Playlists.Playlists {
		r := r
		if r.Owner.ID == "spotify" {
			discoverWeekly = &r
			break
		}
	}
	return discoverWeekly
}

func GenerateRandomString(length int) string {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		log.Panicf("Cannot generate random string: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func GenerateCodeChallenge(codeVerifier string) string {
	sum := sha256.New()
	_, err := sum.Write([]byte(codeVerifier))
	if err != nil {
		log.Panicf("Cannot calculate sha256 sum for string %s: %v", codeVerifier, err)
	}
	return base64.RawURLEncoding.EncodeToString(sum.Sum(nil))
}

func joinArtists(artists []spotify.SimpleArtist) string {
	l := []string{}

	for _, a := range artists {
		l = append(l, a.Name)
	}

	return strings.Join(l, " | ")
}

func PrintSongsInPlaylist(fullPlaylist *spotify.FullPlaylist) {
	fmt.Printf("%-30s %-30s %s\n", "ARTIST", "TRACK", "ALBUM")
	for _, song := range fullPlaylist.Tracks.Tracks {
		fmt.Printf("%-30s %-30s %s\n",
			joinArtists(song.Track.Artists),
			song.Track.Name,
			song.Track.Album.Name,
		)
	}
}

const playlistDescription = "Archived weekly playlist for week %d-%d"

func (s *Spotify) SaveCurrentWeeksPlaylist(
	ctx context.Context,
	userID string,
	playlistName string,
	timeNow time.Time,
	tracks ...spotify.ID,
) error {
	year, week := timeNow.ISOWeek()
	pl, err := s.client.CreatePlaylistForUser(
		ctx,
		userID,
		playlistName,
		fmt.Sprintf(playlistDescription, year, week),
		false,
		false,
	)
	if err != nil {
		return fmt.Errorf("creating playlist: %w", err)
	}

	plSnapshotID, err := s.client.AddTracksToPlaylist(ctx, pl.ID, tracks...)
	if err != nil {
		return fmt.Errorf("add tracks: %w", err)
	}

	fmt.Printf("Added tracks into playlist %q (ID=%s) with snapshot ID %s\n", pl.Name, pl.ID.String(), plSnapshotID)

	return nil
}

func writeTokenToDisk(token *oauth2.Token, tokenFile *os.File) error {
	inJSON, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal oauth token: %w", err)
	}

	_, err = tokenFile.Write(inJSON)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func loadTokenFromDisk(file *os.File) (*oauth2.Token, error) {
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	var token *oauth2.Token
	if err := dec.Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return token, nil
}

func (s *Spotify) NonInteractiveAuth(ctx context.Context) error {
	tokenFile, err := os.OpenFile(tokenFilename, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("load file: %w", err)
	}
	defer tokenFile.Close()

	token, err := loadTokenFromDisk(tokenFile)
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	log.Printf("Loaded token from the disk")

	if token == nil {
		return errors.New("nil token")
	}

	s.token = token
	if err = s.refreshAccessToken(ctx); err != nil {
		return fmt.Errorf("refresh access token: %w", err)
	}

	s.client = spotify.New(s.auth.Client(ctx, s.token))

	return nil
}

func (s *Spotify) InteractiveAuth(ctx context.Context) {
	s.startWebserver(ctx)
}

func (s *Spotify) startWebserver(ctx context.Context) {
	listenAddr := cmp.Or(os.Getenv("HTTP_HOST"), "localhost")
	port := cmp.Or(os.Getenv("HTTP_PORT"), "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.completeAuth)
	mux.HandleFunc("/", func(_ http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})

	s.webserver = &http.Server{
		Addr:              net.JoinHostPort(listenAddr, port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	fmt.Printf("Grant Spotify access to this program by visiting the following link: %s\n",
		s.auth.AuthURL(
			s.state,
			oauth2.AccessTypeOffline,
			oauth2.SetAuthURLParam("client_id", s.clientID),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
			oauth2.SetAuthURLParam("code_challenge", s.codeChallenge),
		),
	)
	go func() {
		if err := s.webserver.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start webserver: %v", err)
		}
	}()
}

func (s *Spotify) ShutdownWebserver(ctx context.Context) {
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.webserver.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Couldn't shutdown the webserver: %v\n", err)
		return
	}
}

func (s *Spotify) completeAuth(w http.ResponseWriter, r *http.Request) {
	log.Printf("Incoming host=%s, URL=%s, method=%s, headers=%s\n",
		r.Host,
		r.URL.String(),
		r.Method,
		r.Header,
	)

	tokenFile, err := os.OpenFile(tokenFilename, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		http.Error(w, "Couldn't read token file", http.StatusInternalServerError)
		log.Printf("Failed to load file: %v\n", err)
		return
	}
	defer tokenFile.Close()

	// Get tokens again if refresh token has expired
	token, err := s.auth.Token(
		r.Context(),
		s.state,
		r,
		oauth2.SetAuthURLParam("code_verifier", s.codeVerifier),
	)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Printf("Failed to get token: %v\n", err)
		return
	}

	if st := r.FormValue("state"); st != s.state {
		http.NotFound(w, r)
		log.Printf("State mismatch: %s != %s\n", st, s.state)
		return
	}

	log.Print("Loaded tokens from the web\n")

	// Store refresh token for later use to avoid constant manual approval
	if err = writeTokenToDisk(token, tokenFile); err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		log.Printf("Failed to store token: %v", err)
		return
	}

	// Use the token to get an authenticated client
	s.client = spotify.New(s.auth.Client(r.Context(), token))

	fmt.Fprintf(w, "Login Completed!")

	s.ClientInitDone <- true
}
