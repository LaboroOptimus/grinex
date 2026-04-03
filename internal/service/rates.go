package service

import (
	"context"
	"fmt"
	"time"

	"github.com/LaboroOptimus/grinex/internal/client/grinex"
)

// CalculationMethod defines how final rates are calculated from order-book arrays.
type CalculationMethod string

const (
	MethodTopN  CalculationMethod = "topN"
	MethodAvgNM CalculationMethod = "avgNM"
)

// RateResult is calculated ask/bid with timestamp.
type RateResult struct {
	Ask        float64
	Bid        float64
	ReceivedAt time.Time
}

// RateSaver persists calculated rates.
type RateSaver interface {
	SaveRate(ctx context.Context, rate StoredRate) error
}

// StoredRate is a database record for a calculated rate.
type StoredRate struct {
	Ask             float64
	Bid             float64
	CalculationType CalculationMethod
	N               uint32
	M               uint32
	Timestamp       time.Time
}

// OrderBookProvider abstracts Grinex client for testing.
type OrderBookProvider interface {
	FetchOrderBook(ctx context.Context) (grinex.OrderBook, error)
}

// RatesService calculates USDT rates from Grinex order-book.
type RatesService struct {
	provider OrderBookProvider
	saver    RateSaver
}

func NewRatesService(provider OrderBookProvider, saver RateSaver) *RatesService {
	if saver == nil {
		saver = nopSaver{}
	}
	return &RatesService{provider: provider, saver: saver}
}

type nopSaver struct{}

func (nopSaver) SaveRate(context.Context, StoredRate) error {
	return nil
}

// GetRates fetches order-book and calculates ask/bid with configured method.
func (s *RatesService) GetRates(ctx context.Context, method CalculationMethod, n, m uint32) (RateResult, error) {
	if err := ValidateCalculationInput(method, n, m); err != nil {
		return RateResult{}, err
	}

	book, err := s.provider.FetchOrderBook(ctx)
	if err != nil {
		return RateResult{}, err
	}

	ask, err := calculate(book.Asks, method, n, m)
	if err != nil {
		return RateResult{}, fmt.Errorf("calculate ask: %w", err)
	}

	bid, err := calculate(book.Bids, method, n, m)
	if err != nil {
		return RateResult{}, fmt.Errorf("calculate bid: %w", err)
	}

	result := RateResult{Ask: ask, Bid: bid, ReceivedAt: time.Now().UTC()}

	if err := s.saver.SaveRate(ctx, StoredRate{
		Ask:             result.Ask,
		Bid:             result.Bid,
		CalculationType: method,
		N:               n,
		M:               m,
		Timestamp:       result.ReceivedAt,
	}); err != nil {
		return RateResult{}, fmt.Errorf("save rate: %w", err)
	}

	return result, nil
}

// ValidateCalculationInput validates N/M values for selected calculation method.
func ValidateCalculationInput(method CalculationMethod, n, m uint32) error {
	switch method {
	case MethodTopN:
		if n == 0 {
			return fmt.Errorf("topN requires n > 0")
		}
		if m != 0 {
			return fmt.Errorf("topN does not use m")
		}
		return nil
	case MethodAvgNM:
		if n == 0 || m == 0 {
			return fmt.Errorf("avgNM requires n > 0 and m > 0")
		}
		if n > m {
			return fmt.Errorf("avgNM requires n <= m")
		}
		return nil
	default:
		return fmt.Errorf("unsupported calculation method: %q", method)
	}
}

func calculate(levels []grinex.Level, method CalculationMethod, n, m uint32) (float64, error) {
	if len(levels) == 0 {
		return 0, fmt.Errorf("order-book levels are empty")
	}

	switch method {
	case MethodTopN:
		idx := int(n - 1)
		if idx < 0 || idx >= len(levels) {
			return 0, fmt.Errorf("topN index %d out of range, levels: %d", n, len(levels))
		}
		return levels[idx].Price, nil
	case MethodAvgNM:
		from := int(n - 1)
		to := int(m - 1)
		if from < 0 || to >= len(levels) {
			return 0, fmt.Errorf("avgNM range [%d;%d] out of range, levels: %d", n, m, len(levels))
		}

		var sum float64
		for i := from; i <= to; i++ {
			sum += levels[i].Price
		}
		count := float64(to - from + 1)
		return sum / count, nil
	default:
		return 0, fmt.Errorf("unsupported calculation method: %q", method)
	}
}
