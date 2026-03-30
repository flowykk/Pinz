package testinfra

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func WithTripPostGIS(t *testing.T) *PostgresContainer {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("docker not available for integration tests: %v", err)
	}

	dbKeys := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	saved := make([]struct {
		k   string
		v   string
		set bool
	}, len(dbKeys))
	for i, k := range dbKeys {
		v, ok := os.LookupEnv(k)
		saved[i] = struct {
			k   string
			v   string
			set bool
		}{k, v, ok}
	}
	t.Cleanup(func() {
		for _, e := range saved {
			if e.set {
				_ = os.Setenv(e.k, e.v)
			} else {
				_ = os.Unsetenv(e.k)
			}
		}
	})

	ctx := context.Background()
	var pg *PostgresContainer
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("testcontainers/docker provider unavailable: %v", r)
			}
		}()
		pg, err = StartPostGIS(ctx, "pinz_trips_test")
	}()
	if err != nil {
		if isInfraStartUnavailable(err) {
			t.Skipf("start postgis (docker/infra unavailable): %v", err)
		}
		t.Fatalf("start postgis: %v", err)
	}
	t.Cleanup(func() { _ = pg.Container.Terminate(ctx) })

	_ = os.Setenv("DB_HOST", pg.Host)
	_ = os.Setenv("DB_PORT", pg.Port)
	_ = os.Setenv("DB_USER", pg.User)
	_ = os.Setenv("DB_PASSWORD", pg.Password)
	_ = os.Setenv("DB_NAME", pg.DBName)

	waitForPostgres(t, pg.DSN(), 60*time.Second)

	return pg
}

// waitForPostgres retries until the DB accepts connections (PostGIS стартует дольше обычного postgres).
func waitForPostgres(t *testing.T, dsn string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		_ = db.Close()
		return
	}
	t.Fatalf("postgres did not become ready within %v", timeout)
}
