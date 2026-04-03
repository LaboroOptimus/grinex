package service

import (
	"context"
	"errors"
	"testing"

	"github.com/LaboroOptimus/grinex/internal/client/grinex"
)

func TestValidateCalculationInput(t *testing.T) {
	tests := []struct {
		name      string
		method    CalculationMethod
		n         uint32
		m         uint32
		wantError bool
	}{
		{name: "topN valid", method: MethodTopN, n: 1, m: 0, wantError: false},
		{name: "topN invalid n", method: MethodTopN, n: 0, m: 0, wantError: true},
		{name: "topN invalid m", method: MethodTopN, n: 1, m: 2, wantError: true},
		{name: "avgNM valid", method: MethodAvgNM, n: 1, m: 3, wantError: false},
		{name: "avgNM invalid zero", method: MethodAvgNM, n: 0, m: 3, wantError: true},
		{name: "avgNM invalid range", method: MethodAvgNM, n: 4, m: 3, wantError: true},
		{name: "invalid method", method: CalculationMethod("x"), n: 1, m: 1, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCalculationInput(tt.method, tt.n, tt.m)
			if tt.wantError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestCalculateTopN(t *testing.T) {
	levels := []grinex.Level{
		{Price: 90.0, Amount: 1},
		{Price: 91.0, Amount: 1},
		{Price: 92.0, Amount: 1},
	}

	got, err := calculate(levels, MethodTopN, 2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 91.0 {
		t.Fatalf("expected 91.0, got %v", got)
	}

	_, err = calculate(levels, MethodTopN, 4, 0)
	if err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

func TestCalculateAvgNM(t *testing.T) {
	levels := []grinex.Level{
		{Price: 100.0, Amount: 1},
		{Price: 110.0, Amount: 1},
		{Price: 130.0, Amount: 1},
		{Price: 160.0, Amount: 1},
	}

	got, err := calculate(levels, MethodAvgNM, 2, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// (110 + 130 + 160) / 3
	if got != 133.33333333333334 {
		t.Fatalf("unexpected average, got %v", got)
	}

	_, err = calculate(levels, MethodAvgNM, 2, 5)
	if err == nil {
		t.Fatalf("expected out-of-range error")
	}
}

type providerMock struct {
	book grinex.OrderBook
	err  error
}

func (m providerMock) FetchOrderBook(context.Context) (grinex.OrderBook, error) {
	if m.err != nil {
		return grinex.OrderBook{}, m.err
	}
	return m.book, nil
}

type saverMock struct {
	saved []StoredRate
	err   error
}

func (m *saverMock) SaveRate(_ context.Context, rate StoredRate) error {
	if m.err != nil {
		return m.err
	}
	m.saved = append(m.saved, rate)
	return nil
}

func TestRatesServiceGetRatesSavesCalculatedRate(t *testing.T) {
	provider := providerMock{
		book: grinex.OrderBook{
			Asks: []grinex.Level{{Price: 100}, {Price: 101}},
			Bids: []grinex.Level{{Price: 99}, {Price: 98}},
		},
	}
	saver := &saverMock{}
	svc := NewRatesService(provider, saver)

	result, err := svc.GetRates(context.Background(), MethodTopN, 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Ask != 100 || result.Bid != 99 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(saver.saved) != 1 {
		t.Fatalf("expected one saved record, got %d", len(saver.saved))
	}

	record := saver.saved[0]
	if record.Ask != 100 || record.Bid != 99 {
		t.Fatalf("unexpected saved record values: %+v", record)
	}
	if record.CalculationType != MethodTopN || record.N != 1 || record.M != 0 {
		t.Fatalf("unexpected saved record method params: %+v", record)
	}
	if record.Timestamp.IsZero() {
		t.Fatalf("expected non-zero timestamp in saved record")
	}
}

func TestRatesServiceGetRatesReturnsErrorWhenSaveFails(t *testing.T) {
	provider := providerMock{
		book: grinex.OrderBook{
			Asks: []grinex.Level{{Price: 100}},
			Bids: []grinex.Level{{Price: 99}},
		},
	}
	saver := &saverMock{err: errors.New("db is down")}
	svc := NewRatesService(provider, saver)

	_, err := svc.GetRates(context.Background(), MethodTopN, 1, 0)
	if err == nil {
		t.Fatalf("expected save error")
	}
}
