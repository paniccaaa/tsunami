VERSION=$(shell git describe --tags --exact-match --always)

.PHONY: build frontend clean

# Build the binary (includes frontend)
build: frontend
	CGO_ENABLED=0 go build -v -a -tags=netgo \
		-ldflags '-s -w -X github.com/paniccaaa/tsunami/cmd.Version=$(VERSION)' \
		-o tsunami main.go

# Build frontend
frontend:
	cd frontend && npm ci && npm run build

# Build without frontend (for development when frontend is already built)
build-go:
	CGO_ENABLED=0 go build -v -a -tags=netgo \
		-ldflags '-s -w -X github.com/paniccaaa/tsunami/cmd.Version=$(VERSION)' \
		-o tsunami main.go

# Clean build artifacts
clean:
	rm -f tsunami
	rm -rf frontend/dist
