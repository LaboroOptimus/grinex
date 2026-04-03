package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LaboroOptimus/grinex/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
)

type execMock struct {
	err    error
	called bool
	query  string
	args   []any
}

func (m *execMock) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.called = true
	m.query = sql
	m.args = arguments
	if m.err != nil {
		return pgconn.CommandTag{}, m.err
	}
	return pgconn.CommandTag{}, nil
}

func TestSaveRateSuccess(t *testing.T) {
	db := &execMock{}
	repo := &RatesRepository{db: db}

	tm := time.Now().UTC()
	rate := service.StoredRate{
		Ask:             81.24,
		Bid:             81.17,
		CalculationType: service.MethodTopN,
		N:               1,
		M:               0,
		Timestamp:       tm,
	}

	err := repo.SaveRate(context.Background(), rate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !db.called {
		t.Fatalf("expected Exec to be called")
	}
	if db.query == "" {
		t.Fatalf("expected query to be set")
	}
	if len(db.args) != 6 {
		t.Fatalf("expected 6 sql args, got %d", len(db.args))
	}
	if got, ok := db.args[0].(float64); !ok || got != rate.Ask {
		t.Fatalf("unexpected ask arg: %#v", db.args[0])
	}
	if got, ok := db.args[1].(float64); !ok || got != rate.Bid {
		t.Fatalf("unexpected bid arg: %#v", db.args[1])
	}
	if got, ok := db.args[2].(string); !ok || got != string(rate.CalculationType) {
		t.Fatalf("unexpected calculation type arg: %#v", db.args[2])
	}
}

func TestSaveRateReturnsError(t *testing.T) {
	db := &execMock{err: errors.New("db fail")}
	repo := &RatesRepository{db: db}

	err := repo.SaveRate(context.Background(), service.StoredRate{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
