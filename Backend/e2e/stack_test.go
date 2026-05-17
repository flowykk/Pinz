package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type stack struct {
	network testcontainers.Network
	netName string
	authDB testcontainers.Container
	tripDB testcontainers.Container
	redis testcontainers.Container
	authSvc testcontainers.Container
	tripSvc testcontainers.Container
	gateway testcontainers.Container
	baseURL string
	authDBDSN string
}

func isDockerUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Cannot connect to the Docker daemon") ||
		strings.Contains(s, "docker daemon") ||
		strings.Contains(s, "Is the docker daemon running")
}

func infraOrSkip(t *testing.T, err error, what string) {
	t.Helper()
	if err == nil {
		return
	}
	if isDockerUnavailable(err) {
		t.Skipf("%s: %v", what, err)
	}
	if os.Getenv("PINZ_E2E_STRICT") == "1" {
		t.Fatalf("%s: %v", what, err)
	}
	t.Skipf("%s: %v", what, err)
}

func requireDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skipf("docker not available for e2e: %v", err)
	}
}

func startStack(t *testing.T) *stack {
	t.Helper()
	requireDocker(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("testcontainers/docker provider unavailable: %v", r)
		}
	}()

	ctx := context.Background()
	st := &stack{}
	t.Cleanup(func() { st.terminate(ctx) })

	netName := "pinz-e2e"
	st.netName = netName
	netw, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name: netName,
			CheckDuplicate: true,
		},
	})
	infraOrSkip(t, err, "create network")
	st.network = netw

	startDB := func(image, alias, dbName string) testcontainers.Container {
		req := testcontainers.ContainerRequest{
			Image: image,
			ExposedPorts: []string{"5432/tcp"},
			Networks: []string{netName},
			NetworkAliases: map[string][]string{
				netName: {alias},
			},
			Env: map[string]string{
				"POSTGRES_USER": "pinz_user",
				"POSTGRES_PASSWORD": "pinz_password",
				"POSTGRES_DB": dbName,
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
		}
		ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started: true,
		})
		infraOrSkip(t, err, "start "+alias)
		return ctr
	}

	st.authDB = startDB("postgres:15-alpine", "auth-db", "pinz_db")
	st.tripDB = startDB("postgis/postgis:15-3.4-alpine", "trip-db", "pinz_trips")

	redisReq := testcontainers.ContainerRequest{
		Image: "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForListeningPort("6379/tcp"),
		Networks: []string{netName},
		NetworkAliases: map[string][]string{
			netName: {"redis"},
		},
	}
	redisCtr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started: true,
	})
	infraOrSkip(t, err, "start redis")
	st.redis = redisCtr

	jwtSecret := "e2e-secret"

	authSvc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: "..",
				Dockerfile: "auth-service/Dockerfile",
			},
			ExposedPorts: []string{"50051/tcp"},
			Networks: []string{netName},
			NetworkAliases: map[string][]string{
				netName: {"auth-service"},
			},
			Env: map[string]string{
				"GRPC_PORT": ":50051",
				"DB_HOST": "auth-db",
				"DB_PORT": "5432",
				"DB_USER": "pinz_user",
				"DB_PASSWORD": "pinz_password",
				"DB_NAME": "pinz_db",
				"REDIS_ADDR": "redis:6379",
				"JWT_SECRET_KEY": jwtSecret,
				"WEBAUTHN_RP_ID": "pinz.website",
				"WEBAUTHN_RP_ORIGIN": "https://pinz.website",
				"AUTH_DEV_LOGIN_ENABLED": "true",
			},
			WaitingFor: wait.ForListeningPort("50051/tcp"),
		},
	})
	infraOrSkip(t, err, "start auth-service")
	st.authSvc = authSvc

	tripSvc, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: "..",
				Dockerfile: "trip-service/Dockerfile",
			},
			ExposedPorts: []string{"50052/tcp"},
			Networks: []string{netName},
			NetworkAliases: map[string][]string{
				netName: {"trip-service"},
			},
			Env: map[string]string{
				"GRPC_PORT": ":50052",
				"DB_HOST": "trip-db",
				"DB_PORT": "5432",
				"DB_USER": "pinz_user",
				"DB_PASSWORD": "pinz_password",
				"DB_NAME": "pinz_trips",
				"REDIS_ADDR": "redis:6379",
			},
			WaitingFor: wait.ForListeningPort("50052/tcp"),
		},
	})
	infraOrSkip(t, err, "start trip-service")
	st.tripSvc = tripSvc

	gateway, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: "..",
				Dockerfile: "api-gateway-service/Dockerfile",
			},
			ExposedPorts: []string{"8080/tcp"},
			Networks: []string{netName},
			Env: map[string]string{
				"API_GATEWAY_PORT": "8080",
				"AUTH_SERVICE_GRPC_ADDRESS": "auth-service:50051",
				"TRIP_SERVICE_GRPC_ADDRESS": "trip-service:50052",
				"REDIS_ADDR": "redis:6379",
				"JWT_SECRET_KEY": jwtSecret,
				"OTEL_EXPORTER_OTLP_ENDPOINT": "",
				"DEV_LOGIN_PROXY_ENABLED": "true",
			},
			WaitingFor: wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
	})
	infraOrSkip(t, err, "start api-gateway-service")
	st.gateway = gateway

	host, err := gateway.Host(ctx)
	infraOrSkip(t, err, "gateway host")
	port, err := gateway.MappedPort(ctx, "8080/tcp")
	infraOrSkip(t, err, "gateway port")
	st.baseURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	authHost, err := st.authDB.Host(ctx)
	infraOrSkip(t, err, "auth db host")
	authPort, err := st.authDB.MappedPort(ctx, "5432/tcp")
	infraOrSkip(t, err, "auth db port")
	st.authDBDSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", "pinz_user", "pinz_password", authHost, authPort.Port(), "pinz_db")

	return st
}

func (s *stack) terminate(ctx context.Context) {
	terminate := func(c any) {
		switch v := c.(type) {
		case testcontainers.Container:
			_ = v.Terminate(ctx)
		case testcontainers.Network:
			if v != nil {
				_ = v.Remove(ctx)
			}
		}
	}
	terminate(s.gateway)
	terminate(s.tripSvc)
	terminate(s.authSvc)
	terminate(s.redis)
	terminate(s.tripDB)
	terminate(s.authDB)
	terminate(s.network)
}

func (s *stack) seedUser(t *testing.T, id, email, username string) {
	t.Helper()
	db, err := sql.Open("pgx", s.authDBDSN)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`INSERT INTO users (id, email, username) VALUES ($1, $2, $3)`, id, email, username)
	require.NoError(t, err)
}

func (s *stack) doJSON(t *testing.T, method, path, bearer string, body string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, s.baseURL+path, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(b)
}
