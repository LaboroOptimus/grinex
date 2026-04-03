package postgres

import (
	"context"
	"fmt"

	"github.com/LaboroOptimus/grinex/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RatesRepository saves calculated rates into PostgreSQL.
type RatesRepository struct {
	db *pgxpool.Pool
}

func NewRatesRepository(db *pgxpool.Pool) *RatesRepository {
	return &RatesRepository{db: db}
}

func (r *RatesRepository) SaveRate(ctx context.Context, rate service.StoredRate) error {
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
		return fmt.Errorf("insert rate: %w", err)
	}

	return nil
}
