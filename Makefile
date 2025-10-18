VERSION=$(shell git describe --tags --exact-match --always)


# Build the binary
build:
	CGO_ENABLED=0 go build -v -a -tags=netgo \
		-ldflags '-s -w -X github.com/paniccaaa/tsunami/cmd.Version=$(VERSION)' \
		-o tsunami main.go
