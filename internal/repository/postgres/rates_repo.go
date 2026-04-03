package postgres

import (
	"context"
	"fmt"

	"github.com/LaboroOptimus/grinex/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RatesRepository saves calculated rates into PostgreSQL.
type RatesRepository struct {
	db dbExecutor
}

var repoTracer = otel.Tracer("postgres/repository")

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func NewRatesRepository(db *pgxpool.Pool) *RatesRepository {
	return &RatesRepository{db: db}
}

func (r *RatesRepository) SaveRate(ctx context.Context, rate service.StoredRate) error {
	ctx, span := repoTracer.Start(ctx, "postgres.save_rate", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("rates.calculation_type", string(rate.CalculationType)),
		attribute.Int64("rates.n", int64(rate.N)),
		attribute.Int64("rates.m", int64(rate.M)),
	)

	const query = `
		INSERT INTO rates (ask, bid, calculation_type, n, m, "timestamp")
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		rate.Ask,
		rate.Bid,
		string(rate.CalculationType),
		int32(rate.N),
		int32(rate.M),
		rate.Timestamp,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "insert failed")
		return fmt.Errorf("insert rate: %w", err)
	}

	span.SetStatus(codes.Ok, "ok")
	return nil
}
