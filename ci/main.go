package main

import "fmt"

const goCacheName = "go-%s-122"

var (
	gooses   = []string{"linux", "darwin"}
	goarches = []string{"amd64", "arm64"}
)

type SaveSpotifyWeekly struct{}

// Build service
func (m *SaveSpotifyWeekly) BuildEnv(src *Directory, goos, goarch string) *Container {
	filteredSource := src.WithoutDirectory("./ci").
		WithoutFile("**/**token.json")

	return dag.Container().
		From("golang:1.22").
		WithDirectory("/src", filteredSource).
		WithMountedCache("/go/pkg/mod", dag.CacheVolume(fmt.Sprintf(goCacheName, "mod"))).
		WithEnvVariable("GOMODCACHE", "/go/pkg/mod").
		WithMountedCache("/go/build-cache", dag.CacheVolume(fmt.Sprintf(goCacheName, "build"))).
		WithEnvVariable("GOCACHE", "/go/build-cache").
		WithWorkdir("/src").
		WithExec([]string{
			"make",
			fmt.Sprintf("GOOS=%s", goos),
			fmt.Sprintf("GOARCH=%s", goarch),
			"build",
		})
}

// Build binary for all the archs
func (m *SaveSpotifyWeekly) Build(src *Directory) *Directory {
	outputs := dag.Directory()
	for _, goos := range gooses {
		for _, goarch := range goarches {
			build := m.BuildEnv(src, goos, goarch)
			outputs = outputs.WithDirectory(".", build.Directory("./dist"))
		}
	}
	return outputs
}
