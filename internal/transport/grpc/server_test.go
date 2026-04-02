package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	ratesv1 "github.com/LaboroOptimus/grinex/api/proto/rates/v1"
	"github.com/LaboroOptimus/grinex/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ratesGetterMock struct {
	result service.RateResult
	err    error
}

func (m ratesGetterMock) GetRates(context.Context, service.CalculationMethod, uint32, uint32) (service.RateResult, error) {
	if m.err != nil {
		return service.RateResult{}, m.err
	}
	return m.result, nil
}

func TestGetRatesSuccess(t *testing.T) {
	now := time.Now().UTC()
	srv := NewServer(ratesGetterMock{result: service.RateResult{Ask: 95.1, Bid: 94.9, ReceivedAt: now}})

	resp, err := srv.GetRates(context.Background(), &ratesv1.GetRatesRequest{
		Method: ratesv1.CalculationMethod_CALCULATION_METHOD_TOP_N,
		N:      1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetAsk() != 95.1 || resp.GetBid() != 94.9 {
		t.Fatalf("unexpected rates response: %+v", resp)
	}
	if resp.GetReceivedAt() == nil {
		t.Fatalf("expected received_at in response")
	}
}

func TestGetRatesInvalidMethod(t *testing.T) {
	srv := NewServer(ratesGetterMock{})

	_, err := srv.GetRates(context.Background(), &ratesv1.GetRatesRequest{
		Method: ratesv1.CalculationMethod_CALCULATION_METHOD_UNSPECIFIED,
		N:      1,
	})
	if err == nil {
		t.Fatalf("expected invalid argument error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", st.Code())
	}
}

func TestGetRatesInvalidParams(t *testing.T) {
	srv := NewServer(ratesGetterMock{})

	_, err := srv.GetRates(context.Background(), &ratesv1.GetRatesRequest{
		Method: ratesv1.CalculationMethod_CALCULATION_METHOD_AVG_N_M,
		N:      5,
		M:      2,
	})
	if err == nil {
		t.Fatalf("expected invalid argument error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", st.Code())
	}
}

func TestGetRatesServiceOutOfRange(t *testing.T) {
	srv := NewServer(ratesGetterMock{err: errors.New("topN index 10 out of range")})

	_, err := srv.GetRates(context.Background(), &ratesv1.GetRatesRequest{
		Method: ratesv1.CalculationMethod_CALCULATION_METHOD_TOP_N,
		N:      1,
	})
	if err == nil {
		t.Fatalf("expected grpc error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %s", st.Code())
	}
}

func TestHealthcheck(t *testing.T) {
	srv := NewServer(ratesGetterMock{})

	resp, err := srv.Healthcheck(context.Background(), &ratesv1.HealthcheckRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetStatus() != "SERVING" {
		t.Fatalf("expected SERVING, got %q", resp.GetStatus())
	}
	if resp.GetCheckedAt() == nil {
		t.Fatalf("expected checked_at")
	}
}
