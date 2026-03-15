package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	postgisDDL = `CREATE EXTENSION IF NOT EXISTS postgis;`
	tripsDDL   = `
CREATE TABLE IF NOT EXISTS trips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id UUID NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  category TEXT NOT NULL,
  season TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'Created',
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  start_date TIMESTAMPTZ,
  end_date TIMESTAMPTZ,
  likes_count INT NOT NULL DEFAULT 0,
  dislikes_count INT NOT NULL DEFAULT 0,
  cover_url TEXT,
  is_published BOOLEAN NOT NULL DEFAULT false,
  is_generated BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	tripParticipantsDDL = `
CREATE TABLE IF NOT EXISTS trip_participants (
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  is_admin BOOLEAN NOT NULL DEFAULT false,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (trip_id, user_id)
);`
)

func InitDB() (*sql.DB, error) {
	dsnStr := dsn()
	// Log connection target without password
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "pinz_trips"
	}
	slog.Info("db: opening connection", "host", host, "port", port, "database", dbname)

	db, err := otelsql.Open("pgx", dsnStr,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			Ping:           false,
			RowsNext:       false,
			DisableErrSkip: true,
		}),
	)
	if err != nil {
		slog.Error("db: open failed", "error", err)
		return nil, fmt.Errorf("open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	slog.Info("db: pinging")
	if err := db.Ping(); err != nil {
		db.Close()
		slog.Error("db: ping failed", "error", err)
		return nil, fmt.Errorf("ping: %w", err)
	}
	slog.Info("db: connected")

	if _, err := otelsql.RegisterDBStatsMetrics(db,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	); err != nil {
		slog.Warn("db: failed to register metrics", "error", err)
	}

	slog.Info("db: running migrations (postgis, trips, trip_participants)")
	for i, ddl := range []string{postgisDDL, tripsDDL, tripParticipantsDDL} {
		name := []string{"postgis", "trips", "trip_participants"}[i]
		if _, err := db.Exec(ddl); err != nil {
			db.Close()
			slog.Error("db: migration failed", "object", name, "error", err)
			return nil, fmt.Errorf("migration %s: %w", name, err)
		}
		slog.Info("db: migration ok", "object", name)
	}
	slog.Info("db: init complete")
	return db, nil
}

func dsn() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "pinz_user"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "pinz_password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "pinz_trips"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}
