package testinfra

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func WithRedis(t *testing.T) *RedisContainer {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("docker not available for integration tests: %v", err)
	}

	const redisKey = "REDIS_ADDR"
	prev, had := os.LookupEnv(redisKey)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(redisKey, prev)
		} else {
			_ = os.Unsetenv(redisKey)
		}
	})

	ctx := context.Background()
	var rd *RedisContainer
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("testcontainers/docker provider unavailable: %v", r)
			}
		}()
		rd, err = StartRedis(ctx)
	}()
	if err != nil {
		if isInfraStartUnavailable(err) {
			t.Skipf("start redis (docker/infra unavailable): %v", err)
		}
		t.Fatalf("start redis: %v", err)
	}
	t.Cleanup(func() { _ = rd.Container.Terminate(ctx) })

	_ = os.Setenv(redisKey, rd.Addr())
	return rd
}
