APP_NAME	?= savespotifyweekly

GO		?= go
DAGGER		?= dagger
VERSION		?= $(shell git log --pretty=format:%h -n 1)
BUILD_TIME	?= $(shell date)
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		?= -asmflags -trimpath -ldflags \
		   "-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'"
GOARCH		?=
GOOS		?=
# CGO_ENABLED=0 == static by default
CGO_ENABLED	?= 0

COMPOSE_FILE	?= docker-compose.yml


#all: test lint savespotifyweekly
all: test build

build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(LDFLAGS) \
		-o dist/$(APP_NAME)_$(GOOS)_$(GOARCH) \
		cmd/main.go

dagger-build-all-archs:
	$(DAGGER) call build --src=. export --path=./dist

dagger-update:
	$(DAGGER) develop --source=ci

.PHONY: clean
clean:
	rm -rf dist/

install-dependencies:
	@go get -d -v ./...

lint:
	@golangci-lint run ./...

vulncheck:
	@govulncheck ./...

escape-analysis:
	$(GO) build -gcflags="-m" 2>&1

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test:
	go test ./...

# This runs all tests, including integration tests
test-integration:
	-@go test -tags=integration ./...
	#@docker compose down

