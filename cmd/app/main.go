package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/LaboroOptimus/grinex/internal/client/grinex"
	"github.com/LaboroOptimus/grinex/internal/repository/postgres"
	"github.com/LaboroOptimus/grinex/internal/service"
	transportgrpc "github.com/LaboroOptimus/grinex/internal/transport/grpc"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()
	defaultAddr := getenv("GRPC_ADDR", ":50051")
	defaultDBURL := getenv("DATABASE_URL", "")
	defaultDBHost := getenv("DB_HOST", "localhost")
	defaultDBPort := getenv("DB_PORT", "5432")
	defaultDBUser := getenv("DB_USER", "postgres")
	defaultDBPassword := getenv("DB_PASSWORD", "postgres")
	defaultDBName := getenv("DB_NAME", "postgres")
	defaultDBSSLMode := getenv("DB_SSLMODE", "disable")
	defaultMigrationsDir := getenv("MIGRATIONS_DIR", "migrations")

	grpcAddr := flag.String("grpc-addr", defaultAddr, "gRPC listen address")
	dbURL := flag.String("db-url", defaultDBURL, "PostgreSQL connection URL")
	dbHost := flag.String("db-host", defaultDBHost, "PostgreSQL host")
	dbPort := flag.String("db-port", defaultDBPort, "PostgreSQL port")
	dbUser := flag.String("db-user", defaultDBUser, "PostgreSQL user")
	dbPassword := flag.String("db-password", defaultDBPassword, "PostgreSQL password")
	dbName := flag.String("db-name", defaultDBName, "PostgreSQL database name")
	dbSSLMode := flag.String("db-sslmode", defaultDBSSLMode, "PostgreSQL SSL mode")
	migrationsDir := flag.String("migrations-dir", defaultMigrationsDir, "Path to SQL migrations directory")
	flag.Parse()

	dsn := *dbURL
	if dsn == "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			*dbUser,
			*dbPassword,
			*dbHost,
			*dbPort,
			*dbName,
			*dbSSLMode,
		)
	}

	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}
	defer dbPool.Close()

	if err = dbPool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	if err = runMigrations(ctx, dbPool, *migrationsDir); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	listener, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *grpcAddr, err)
	}

	grinexClient := grinex.NewClient()
	ratesRepo := postgres.NewRatesRepository(dbPool)
	ratesService := service.NewRatesService(grinexClient, ratesRepo)
	ratesServer := transportgrpc.NewServer(ratesService)

	grpcServer := grpc.NewServer()
	transportgrpc.Register(grpcServer, ratesServer)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server listening on %s", *grpcAddr)
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			log.Printf("gRPC server stopped with error: %v", serveErr)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Printf("shutting down gRPC server")
	grpcServer.GracefulStop()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func runMigrations(ctx context.Context, dbPool *pgxpool.Pool, dir string) error {
	const createMigrationsTable = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	if _, err := dbPool.Exec(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	upFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upFiles = append(upFiles, name)
		}
	}
	sort.Strings(upFiles)

	for _, fileName := range upFiles {
		version := strings.TrimSuffix(fileName, ".up.sql")
		applied, err := isMigrationApplied(ctx, dbPool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(dir, fileName)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", path, err)
		}

		if _, err = dbPool.Exec(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %q: %w", fileName, err)
		}
		if _, err = dbPool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("track migration %q: %w", fileName, err)
		}

		log.Printf("applied migration %s", fileName)
	}

	return nil
}

func isMigrationApplied(ctx context.Context, dbPool *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	if err := dbPool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %q: %w", version, err)
	}
	return exists, nil
}
