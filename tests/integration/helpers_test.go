//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testBaseURL is shared across all tests in this package.
// The httpbin container starts once in TestMain and stops after all tests finish.
var testBaseURL string

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mccutchen/go-httpbin",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/get").WithPort("8080/tcp"),
		},
		Started: true,
	})
	if err != nil {
		log.Fatalf("failed to start httpbin container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		log.Fatalf("failed to get container port: %v", err)
	}

	testBaseURL = fmt.Sprintf("http://%s:%s", host, port.Port())
	log.Printf("httpbin ready at %s", testBaseURL)

	code := m.Run()

	if err := container.Terminate(ctx); err != nil {
		log.Printf("failed to terminate container: %v", err)
	}

	os.Exit(code)
}
