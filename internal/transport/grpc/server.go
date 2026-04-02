package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ratesv1 "github.com/LaboroOptimus/grinex/api/proto/rates/v1"
	"github.com/LaboroOptimus/grinex/internal/service"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RatesGetter abstracts the rates business logic used by gRPC handlers.
type RatesGetter interface {
	GetRates(ctx context.Context, method service.CalculationMethod, n, m uint32) (service.RateResult, error)
}

// Server implements rates.v1.RatesService gRPC endpoints.
type Server struct {
	ratesv1.UnimplementedRatesServiceServer
	rates RatesGetter
}

func NewServer(rates RatesGetter) *Server {
	return &Server{rates: rates}
}

func Register(s ggrpc.ServiceRegistrar, server *Server) {
	ratesv1.RegisterRatesServiceServer(s, server)
}

func (s *Server) GetRates(ctx context.Context, req *ratesv1.GetRatesRequest) (*ratesv1.GetRatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	method, err := mapCalculationMethod(req.GetMethod())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := service.ValidateCalculationInput(method, req.GetN(), req.GetM()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	result, err := s.rates.GetRates(ctx, method, req.GetN(), req.GetM())
	if err != nil {
		if strings.Contains(err.Error(), "out of range") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ratesv1.GetRatesResponse{
		Ask:        result.Ask,
		Bid:        result.Bid,
		ReceivedAt: timestamppb.New(result.ReceivedAt),
	}, nil
}

func (s *Server) Healthcheck(context.Context, *ratesv1.HealthcheckRequest) (*ratesv1.HealthcheckResponse, error) {
	return &ratesv1.HealthcheckResponse{
		Status:    "SERVING",
		CheckedAt: timestamppb.New(time.Now().UTC()),
	}, nil
}

func mapCalculationMethod(in ratesv1.CalculationMethod) (service.CalculationMethod, error) {
	switch in {
	case ratesv1.CalculationMethod_CALCULATION_METHOD_TOP_N:
		return service.MethodTopN, nil
	case ratesv1.CalculationMethod_CALCULATION_METHOD_AVG_N_M:
		return service.MethodAvgNM, nil
	case ratesv1.CalculationMethod_CALCULATION_METHOD_UNSPECIFIED:
		return "", errors.New("calculation method is required")
	default:
		return "", fmt.Errorf("unsupported calculation method: %s", in.String())
	}
}
