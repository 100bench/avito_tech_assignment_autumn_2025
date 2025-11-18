package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	cfg := config.Load()

	if err := runMigrations(cfg.PostgresDSN()); err != nil {
		return errors.Wrap(err, "failed to run migrations")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage, err := postgres.NewPgxClient(ctx, cfg.PostgresDSN())
	if err != nil {
		return errors.Wrap(err, "postgres.NewPgxClient")
	}

	service, err := usecases.NewServiceStorage(storage)
	if err != nil {
		storage.Close()
		return errors.Wrap(err, "usecases.NewServiceStorage")
	}

	server, err := public.NewServer(service)
	if err != nil {
		storage.Close()
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
		log.Printf("Starting HTTP server on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		cancel()
		storage.Close()
		return errors.Wrap(err, "http server error")
	case sig := <-stop:
		log.Printf("Received signal: %v. Starting graceful shutdown...", sig)

		// Отменяем основной контекст
		cancel()

		// Создаём контекст для shutdown с таймаутом
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()

		// Останавливаем HTTP сервер
		log.Println("Shutting down HTTP server...")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
			storage.Close()
			return errors.Wrap(err, "server shutdown failed")
		}
		log.Println("HTTP server stopped")

		// Даём время на завершение активных операций с БД
		time.Sleep(100 * time.Millisecond)

		// Закрываем соединение с БД
		log.Println("Closing database connection...")
		storage.Close()
		log.Println("Database connection closed")

		log.Println("Graceful shutdown completed")
		return nil
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://deployment/migrations/postgres",
		dsn,
	)
	if err != nil {
		return err
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if dbErr != nil && err == nil {
			err = dbErr
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
