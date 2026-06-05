VERSION=$(shell git describe --tags --exact-match --always)

.PHONY: build frontend clean build-cli test test-integration vet run docker

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

# Run unit tests
test:
	go test ./...

# Run integration tests
test-integration:
	go test -tags=integration ./tests/integration/...

# Vet
vet:
	go vet ./...

# Run without building (attack example)
run:
	go run main.go attack --url https://httpbin.org/get

# Start server without building
run-serve:
	go run main.go serve

# Build and run with Docker Compose
docker:
	docker-compose up --build

# Clean build artifacts
clean:
	rm -f tsunami
	rm -rf frontend/dist
