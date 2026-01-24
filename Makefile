VERSION=$(shell git describe --tags --exact-match --always)

.PHONY: build frontend clean build-cli

# Build the binary with embedded frontend
build: frontend
	CGO_ENABLED=0 go build -v -a -tags=netgo,embed_frontend \
		-ldflags '-s -w -X github.com/paniccaaa/tsunami/cmd.Version=$(VERSION)' \
		-o tsunami main.go

# Build frontend
frontend:
	cd frontend && npm ci && npm run build

# Build CLI-only (no web UI)
build-cli:
	CGO_ENABLED=0 go build -v -a -tags=netgo \
		-ldflags '-s -w -X github.com/paniccaaa/tsunami/cmd.Version=$(VERSION)' \
		-o tsunami main.go

# Clean build artifacts
clean:
	rm -f tsunami
	rm -rf frontend/dist
