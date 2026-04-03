package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"os/signal"

	"github.com/LaboroOptimus/grinex/internal/client/grinex"
	"github.com/LaboroOptimus/grinex/internal/config"
	"github.com/LaboroOptimus/grinex/internal/repository/postgres"
	"github.com/LaboroOptimus/grinex/internal/service"
	transportgrpc "github.com/LaboroOptimus/grinex/internal/transport/grpc"
	otelsetup "github.com/LaboroOptimus/grinex/pkg/otel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	baseCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	otelShutdown, err := otelsetup.Init(baseCtx, cfg.OTELService)
	if err != nil {
		log.Fatalf("failed to init open telemetry: %v", err)
	}

	dbPool, err := pgxpool.New(baseCtx, cfg.DSN())
	if err != nil {
		log.Fatalf("failed to create postgres pool: %v", err)
	}

	if err = dbPool.Ping(baseCtx); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	if err = runMigrations(baseCtx, dbPool, cfg.MigrationsDir); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", cfg.GRPCAddr, err)
	}

	grinexClient := grinex.NewClient()
	ratesRepo := postgres.NewRatesRepository(dbPool)
	ratesService := service.NewRatesService(grinexClient, ratesRepo)
	ratesServer := transportgrpc.NewServer(ratesService)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	transportgrpc.Register(grpcServer, ratesServer)
	reflection.Register(grpcServer)

	metricsServer := &http.Server{
		Addr:    cfg.MetricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		log.Printf("gRPC server listening on %s", cfg.GRPCAddr)
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			log.Printf("gRPC server stopped with error: %v", serveErr)
		}
	}()

	go func() {
		log.Printf("metrics server listening on %s", cfg.MetricsAddr)
		if serveErr := metricsServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Printf("metrics server stopped with error: %v", serveErr)
		}
	}()

	<-baseCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Printf("shutting down application")
	shutdownGRPC(grpcServer)
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown metrics server: %v", err)
	}
	dbPool.Close()
	if err := otelShutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown open telemetry: %v", err)
	}
}

func shutdownGRPC(server *grpc.Server) {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		server.Stop()
	}
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
