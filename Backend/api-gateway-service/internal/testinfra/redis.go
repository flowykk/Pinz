package testinfra

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type RedisContainer struct {
	Container testcontainers.Container
	Host string
	Port string
}

func (c *RedisContainer) Addr() string { return fmt.Sprintf("%s:%s", c.Host, c.Port) }

func StartRedis(ctx context.Context) (*RedisContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForListeningPort("6379/tcp"),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started: true,
	})
	if err != nil {
		return nil, err
	}
	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, err
	}
	port, err := ctr.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, err
	}
	return &RedisContainer{Container: ctr, Host: host, Port: port.Port()}, nil
}
