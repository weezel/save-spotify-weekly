# Save Spotify weekly

## Description

Archives Spotify's Discover Weekly playlists for later use.
This is achieved by creating a new playlist which contains a week number and a year by default.

Every now and then I forget to listen a discover weekly and occasionally it would be nice to revisit the good lists.
Hence, archive them just in case and listen afterwards.

The application stores JWT tokens into `token.json` file for later use.

## Dependencies

* Go 1.22 (maybe older works too?)
* Spotify
  * Spotify ID and Spotify secret
* Place where the binary can be running constantly to keep the refresh token refreshed

## Installing

There are pre compiled binaries available for the following operating systems and architectures:

<!-- TODO add links to binaries -->
* Darwin/amd64
* Darwin/arm64
* Linux/amd64
* Linux/arm64
* OpenBSD/amd64
* OpenBSD/arm64

### From source

Either run `make build` to build binary by using the golang compiler directly or build by using a Dagger.
The dagger is a docker equivalent tool and therefore removes many incompatibilities between the different systems.
Dagger build is launched by executing `make build-all-archs`.
The latter method is being used for compiling the distributed binaries.

## Configuration

Following environmental variables can be set:

| Env name             | Description                                             | Default                          |
| --------------       | ------------------------------------------------------- | -------------------------------- |
| PLAYLIST_NAME        | Name of the archived playlist                           | Archived discover weekly %d-%d", year, week                                |
| SPOTIFY_ID           | Spotify ID (this is mandatory)                          | -                                |
| SPOTIFY_SECRET       | Spotify secret (this is mandatory)                      | -                                |
| HTTP_HOST            | HTTP host where to listen (hosting callback URL)        | localhost                        |
| HTTP_PORT            | HTTP port where to listen (hosting callback URL)        | 8080                             |
| SPOTIFY_CALLBACK_URL | Call back URL used during the first time authentication | <http://localhost:8080/callback> |

## TODO

- [ ] Add binaries to release
- [ ] Add functionality to sleep for almost a day so refresh token can be refreshed
  - Currently using this workaround `while true; do ./savespotifyweekly_linux_amd64; sleep 86300; done`
