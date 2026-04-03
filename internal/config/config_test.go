package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("METRICS_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DB_SSLMODE", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("OTEL_SERVICE_NAME", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCAddr != ":50051" {
		t.Fatalf("unexpected default grpc addr: %s", cfg.GRPCAddr)
	}
	if cfg.MetricsAddr != ":9090" {
		t.Fatalf("unexpected default metrics addr: %s", cfg.MetricsAddr)
	}
	if cfg.DBHost != "localhost" || cfg.DBPort != "5432" {
		t.Fatalf("unexpected default db host/port: %s:%s", cfg.DBHost, cfg.DBPort)
	}
	if cfg.MigrationsDir != "migrations" {
		t.Fatalf("unexpected default migrations dir: %s", cfg.MigrationsDir)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("GRPC_ADDR", ":7777")
	t.Setenv("METRICS_ADDR", ":9191")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "6432")
	t.Setenv("DB_USER", "usr")
	t.Setenv("DB_PASSWORD", "pwd")
	t.Setenv("DB_NAME", "rates")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("MIGRATIONS_DIR", "/tmp/migrations")
	t.Setenv("OTEL_SERVICE_NAME", "grinex-env")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCAddr != ":7777" {
		t.Fatalf("expected env grpc addr, got %s", cfg.GRPCAddr)
	}
	if cfg.MetricsAddr != ":9191" {
		t.Fatalf("expected env metrics addr, got %s", cfg.MetricsAddr)
	}
	if cfg.DBHost != "db" || cfg.DBPort != "6432" || cfg.DBUser != "usr" {
		t.Fatalf("unexpected env db config: %+v", cfg)
	}
	if cfg.MigrationsDir != "/tmp/migrations" {
		t.Fatalf("expected env migrations dir, got %s", cfg.MigrationsDir)
	}
	if cfg.OTELService != "grinex-env" {
		t.Fatalf("expected env otel service, got %s", cfg.OTELService)
	}
}

func TestLoadFlagsOverrideEnv(t *testing.T) {
	t.Setenv("GRPC_ADDR", ":7777")
	t.Setenv("METRICS_ADDR", ":9191")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "6432")
	t.Setenv("MIGRATIONS_DIR", "/tmp/migrations")
	t.Setenv("OTEL_SERVICE_NAME", "env-service")

	cfg, err := Load([]string{
		"-grpc-addr=:9999",
		"-metrics-addr=:9998",
		"-db-host=postgres",
		"-db-port=5432",
		"-migrations-dir=/app/migrations",
		"-otel-service-name=grinex-prod",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPCAddr != ":9999" {
		t.Fatalf("flags should override env grpc addr, got %s", cfg.GRPCAddr)
	}
	if cfg.MetricsAddr != ":9998" {
		t.Fatalf("flags should override env metrics addr, got %s", cfg.MetricsAddr)
	}
	if cfg.DBHost != "postgres" || cfg.DBPort != "5432" {
		t.Fatalf("flags should override env db host/port, got %s:%s", cfg.DBHost, cfg.DBPort)
	}
	if cfg.MigrationsDir != "/app/migrations" {
		t.Fatalf("flags should override env migrations dir, got %s", cfg.MigrationsDir)
	}
	if cfg.OTELService != "grinex-prod" {
		t.Fatalf("flags should override env otel service, got %s", cfg.OTELService)
	}
}

func TestDSNPriority(t *testing.T) {
	cfg := Config{
		DatabaseURL: "postgres://direct/url",
		DBHost:      "ignored",
		DBPort:      "5432",
		DBUser:      "postgres",
		DBPassword:  "postgres",
		DBName:      "postgres",
		DBSSLMode:   "disable",
	}

	if got := cfg.DSN(); got != "postgres://direct/url" {
		t.Fatalf("expected direct database url, got %s", got)
	}

	cfg.DatabaseURL = ""
	cfg.DBHost = "localhost"
	cfg.DBPort = "5432"
	cfg.DBUser = "postgres"
	cfg.DBPassword = "postgres"
	cfg.DBName = "rates"
	cfg.DBSSLMode = "disable"

	want := "postgres://postgres:postgres@localhost:5432/rates?sslmode=disable"
	if got := cfg.DSN(); got != want {
		t.Fatalf("unexpected dsn: got %s want %s", got, want)
	}
}
