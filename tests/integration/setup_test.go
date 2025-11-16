package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/adapters/storage/postgres"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/ports/http/public"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/usecases"
)

type TestEnv struct {
	PostgresContainer testcontainers.Container
	DSN               string
	Server            *httptest.Server
	Client            *http.Client
	ctx               context.Context
}

func SetupTestEnv(t *testing.T) *TestEnv {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf(
		"postgres://testuser:testpass@%s:%s/testdb?sslmode=disable",
		host,
		port.Port(),
	)

	err = runMigrations(dsn)
	require.NoError(t, err)

	storage, err := postgres.NewPgxClient(ctx, dsn)
	require.NoError(t, err)

	service, err := usecases.NewServiceStorage(storage)
	require.NoError(t, err)

	server, err := public.NewServer(service)
	require.NoError(t, err)

	testServer := httptest.NewServer(server.GetRouter())

	return &TestEnv{
		PostgresContainer: postgresContainer,
		DSN:               dsn,
		Server:            testServer,
		Client:            &http.Client{Timeout: 10 * time.Second},
		ctx:               ctx,
	}
}

func (e *TestEnv) Cleanup(t *testing.T) {
	e.Server.Close()
	if err := e.PostgresContainer.Terminate(e.ctx); err != nil {
		t.Logf("failed to terminate container: %v", err)
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://../../deployment/migrations/postgres",
		dsn,
	)
	if err != nil {
		return err
	}
	defer func() {
		if sourceErr, dbErr := m.Close(); sourceErr != nil || dbErr != nil {
			log.Printf("close metric: %v, %v", sourceErr, dbErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
