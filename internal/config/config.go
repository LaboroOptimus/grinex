package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	defaultGRPCAddr      = ":50051"
	defaultDBHost        = "localhost"
	defaultDBPort        = "5432"
	defaultDBUser        = "postgres"
	defaultDBPassword    = "postgres"
	defaultDBName        = "postgres"
	defaultDBSSLMode     = "disable"
	defaultMigrationsDir = "migrations"
)

// Config stores application runtime configuration.
type Config struct {
	GRPCAddr      string
	DatabaseURL   string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	MigrationsDir string
}

// DSN returns PostgreSQL DSN. DatabaseURL has higher priority than split DB_* fields.
func (c Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

// Load reads config from flags and env using priority: flags > env > defaults.
func Load(args []string) (Config, error) {
	cfg := Config{
		GRPCAddr:      envOrDefault("GRPC_ADDR", defaultGRPCAddr),
		DatabaseURL:   envOrDefault("DATABASE_URL", ""),
		DBHost:        envOrDefault("DB_HOST", defaultDBHost),
		DBPort:        envOrDefault("DB_PORT", defaultDBPort),
		DBUser:        envOrDefault("DB_USER", defaultDBUser),
		DBPassword:    envOrDefault("DB_PASSWORD", defaultDBPassword),
		DBName:        envOrDefault("DB_NAME", defaultDBName),
		DBSSLMode:     envOrDefault("DB_SSLMODE", defaultDBSSLMode),
		MigrationsDir: envOrDefault("MIGRATIONS_DIR", defaultMigrationsDir),
	}

	fs := flag.NewFlagSet("app", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "gRPC listen address")
	fs.StringVar(&cfg.DatabaseURL, "db-url", cfg.DatabaseURL, "PostgreSQL connection URL")
	fs.StringVar(&cfg.DBHost, "db-host", cfg.DBHost, "PostgreSQL host")
	fs.StringVar(&cfg.DBPort, "db-port", cfg.DBPort, "PostgreSQL port")
	fs.StringVar(&cfg.DBUser, "db-user", cfg.DBUser, "PostgreSQL user")
	fs.StringVar(&cfg.DBPassword, "db-password", cfg.DBPassword, "PostgreSQL password")
	fs.StringVar(&cfg.DBName, "db-name", cfg.DBName, "PostgreSQL database name")
	fs.StringVar(&cfg.DBSSLMode, "db-sslmode", cfg.DBSSLMode, "PostgreSQL SSL mode")
	fs.StringVar(&cfg.MigrationsDir, "migrations-dir", cfg.MigrationsDir, "Path to SQL migrations directory")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
