package testinfra

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PostgresContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	User      string
	Password  string
	DBName    string
}

func (c *PostgresContainer) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.User, c.Password, c.Host, c.Port, c.DBName)
}

func StartPostGIS(ctx context.Context, dbName string) (*PostgresContainer, error) {
	user := "pinz_user"
	password := "pinz_password"
	if dbName == "" {
		dbName = "pinz_trip_test"
	}

	pg, err := postgres.Run(ctx,
		"postgis/postgis:15-3.4-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
	)
	if err != nil {
		return nil, err
	}

	host, err := pg.Host(ctx)
	if err != nil {
		_ = pg.Terminate(ctx)
		return nil, err
	}
	port, err := pg.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = pg.Terminate(ctx)
		return nil, err
	}

	return &PostgresContainer{
		Container: pg,
		Host:      host,
		Port:      port.Port(),
		User:      user,
		Password:  password,
		DBName:    dbName,
	}, nil
}
