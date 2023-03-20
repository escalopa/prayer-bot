package testcon

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewRedisContainer(ctx context.Context) (url string, terminate func() error, err error) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("* Ready to accept connections"),
	}
	// Start redis container
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to start redis container")
	}
	// Get container port
	mappedPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get container port")
	}
	// Get container host
	hostIP, err := redisContainer.Host(ctx)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to get container host")
	}
	// Build redis url
	uri := fmt.Sprintf("redis://%s:%s", hostIP, mappedPort.Port())
	// Create terminate function to stop container when done
	terminate = func() error {
		return redisContainer.Terminate(ctx)
	}
	return uri, terminate, nil
}
