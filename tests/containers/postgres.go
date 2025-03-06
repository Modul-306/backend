package containers

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestContainer struct {
	Container testcontainers.Container
	URI       string
	Port      string
}

func NewTestPostgres(t *testing.T) (*TestContainer, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %v", err)
	}

	uri := fmt.Sprintf("postgresql://test:test@localhost:%s/testdb?sslmode=disable", port.Port())

	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("DB_USER", "test")
	os.Setenv("DB_PASSWORD", "test")
	os.Setenv("DB_NAME", "testdb")

	return &TestContainer{
		Container: container,
		URI:       uri,
		Port:      port.Port(),
	}, nil
}

func (tc *TestContainer) Cleanup(t *testing.T) {
	ctx := context.Background()
	if err := tc.Container.Terminate(ctx); err != nil {
		t.Fatalf("failed to terminate container: %v", err)
	}
}
