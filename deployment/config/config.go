package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PostgresHost     string `yaml:"postgres_host"`
	PostgresPort     string `yaml:"postgres_port"`
	PostgresUser     string `yaml:"postgres_user"`
	PostgresPassword string `yaml:"postgres_password"`
	PostgresDB       string `yaml:"postgres_db"`
	HTTPAddr         string `yaml:"http_addr"`
	ShutdownTimeout  time.Duration
}

func Load() *Config {
	cfg := &Config{
		PostgresHost:     "localhost",
		PostgresPort:     "5432",
		PostgresUser:     "postgres",
		PostgresPassword: "postgres",
		PostgresDB:       "pr_review_db",
		HTTPAddr:         ":8080",
		ShutdownTimeout:  5 * time.Second,
	}

	if data, err := os.ReadFile("deployment/config/config.yaml"); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		cfg.PostgresHost = host
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		cfg.PostgresPort = port
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.PostgresUser = user
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		cfg.PostgresPassword = password
	}
	if db := os.Getenv("POSTGRES_DB"); db != "" {
		cfg.PostgresDB = db
	}
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		cfg.HTTPAddr = addr
	}

	return cfg
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
	)
}