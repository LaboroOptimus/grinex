package service

import (
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
