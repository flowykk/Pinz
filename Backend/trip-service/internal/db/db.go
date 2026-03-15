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
	invitationLinksDDL = `
CREATE TABLE IF NOT EXISTS invitation_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  token TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	tripSettingsDDL = `
CREATE TABLE IF NOT EXISTS trip_settings (
  user_id UUID NOT NULL,
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  notifications_enabled BOOLEAN NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, trip_id)
);`
	tripPrivacyDDL = `
CREATE TABLE IF NOT EXISTS trip_privacy (
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (trip_id, user_id)
);`
	// PINZ-97: pins, media, tags, pin_privacy, media_privacy (tripCreationFlow.md)
	pinsDDL = `
CREATE TABLE IF NOT EXISTS pins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  location GEOMETRY(Point, 4326),
  category TEXT NOT NULL,
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  media_count INT NOT NULL DEFAULT 0,
  start_time TIMESTAMPTZ,
  end_time TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	mediaDDL = `
CREATE TABLE IF NOT EXISTS media (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  pin_id UUID REFERENCES pins(id) ON DELETE SET NULL,
  s3_key TEXT NOT NULL,
  media_type TEXT NOT NULL,
  location GEOMETRY(Point, 4326),
  captured_at TIMESTAMPTZ,
  battle_rating INT NOT NULL DEFAULT 0,
  privacy_level TEXT NOT NULL DEFAULT 'Private',
  similar_group_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	tagsDDL = `
CREATE TABLE IF NOT EXISTS tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
  pin_id UUID REFERENCES pins(id) ON DELETE CASCADE,
  tag TEXT NOT NULL,
  UNIQUE(trip_id, pin_id, tag)
);`
	pinPrivacyDDL = `
CREATE TABLE IF NOT EXISTS pin_privacy (
  pin_id UUID NOT NULL REFERENCES pins(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (pin_id, user_id)
);`
	mediaPrivacyDDL = `
CREATE TABLE IF NOT EXISTS media_privacy (
  media_id UUID NOT NULL REFERENCES media(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  privacy_level TEXT NOT NULL,
  PRIMARY KEY (media_id, user_id)
);`
)

func InitDB() (*sql.DB, error) {
	dsnStr := dsn()
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

	slog.Info("db: running migrations (postgis, trips, ..., pins, media, tags, pin_privacy, media_privacy)")
	for i, ddl := range []string{postgisDDL, tripsDDL, tripParticipantsDDL, invitationLinksDDL, tripSettingsDDL, tripPrivacyDDL, pinsDDL, mediaDDL, tagsDDL, pinPrivacyDDL, mediaPrivacyDDL} {
		name := []string{"postgis", "trips", "trip_participants", "invitation_links", "trip_settings", "trip_privacy", "pins", "media", "tags", "pin_privacy", "media_privacy"}[i]
		if _, err := db.Exec(ddl); err != nil {
			db.Close()
			slog.Error("db: migration failed", "object", name, "error", err)
			return nil, fmt.Errorf("migration %s: %w", name, err)
		}
		slog.Info("db: migration ok", "object", name)
	}
	// Добавляем similar_group_id для поддержки нескольких групп похожих медиа внутри пина (существующие БД).
	if _, err := db.Exec("ALTER TABLE media ADD COLUMN IF NOT EXISTS similar_group_id UUID"); err != nil {
		db.Close()
		slog.Error("db: migration media.similar_group_id failed", "error", err)
		return nil, fmt.Errorf("migration media.similar_group_id: %w", err)
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
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "pinz_password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "pinz_trips"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}
