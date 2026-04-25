package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

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
			Ping: false,
			RowsNext: false,
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

	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		db.Close()
		return nil, fmt.Errorf("goose dialect: %w", err)
	}
	slog.Info("db: running migrations")
	if err := goose.Up(db, "migrations"); err != nil {
		db.Close()
		slog.Error("db: migrations failed", "error", err)
		return nil, fmt.Errorf("migrations: %w", err)
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
