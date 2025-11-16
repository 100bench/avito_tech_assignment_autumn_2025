package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"

	"github.com/100bench/avito_tech_assignment_autumn_2025/deployment/config"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/adapters/storage/postgres"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/ports/http/public"
	"github.com/100bench/avito_tech_assignment_autumn_2025/internal/usecases"
)

func RunApp() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.Load()

	if err := runMigrations(cfg.PostgresDSN()); err != nil {
		return errors.Wrap(err, "failed to run migrations")
	}

	storage, err := postgres.NewPgxClient(ctx, cfg.PostgresDSN())
	if err != nil {
		return errors.Wrap(err, "postgres.NewPgxClient")
	}
	defer storage.Close()

	service, err := usecases.NewServiceStorage(storage)
	if err != nil {
		return errors.Wrap(err, "usecases.NewServiceStorage")
	}

	server, err := public.NewServer(service)
	if err != nil {
		return errors.Wrap(err, "public.NewServer")
	}

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: server.GetRouter(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return errors.Wrap(err, "http server error")
	case <-stop:
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancelShutdown()

		if err := httpServer.Shutdown(ctxShutdown); err != nil {
			return errors.Wrap(err, "server shutdown failed")
		}

		return nil
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return err
	}
	defer func() {
    	sourceErr , DbErr := m.Close()
        if DbErr != nil && err == nil {
            err = DbErr
        }
		if sourceErr != nil && err == nil {
            err = sourceErr
        }
    }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
