package grinex

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultBaseURL = "https://grinex.io"
	depthPath      = "/api/v1/spot/depth"
	defaultSymbol  = "usdta7a5"
)

// Level represents a single order-book level.
type Level struct {
	Price  float64
	Amount float64
}

// OrderBook contains asks and bids from the exchange.
type OrderBook struct {
	Asks []Level
	Bids []Level
}

type depthResponse struct {
	Asks []depthLevel `json:"asks"`
	Bids []depthLevel `json:"bids"`
}

type depthLevel struct {
	Price  string `json:"price"`
	Amount string `json:"amount"`
}

// Client fetches spot depth from Grinex.
type Client struct {
	http *resty.Client
}

var clientTracer = otel.Tracer("grinex/client")

// NewClient creates a Grinex HTTP client with default settings.
func NewClient() *Client {
	rc := resty.New().
		SetBaseURL(defaultBaseURL).
		SetHeader("Accept", "application/json")

	return &Client{http: rc}
}

// NewClientWithHTTP creates a Grinex client with provided resty client.
func NewClientWithHTTP(httpClient *resty.Client) *Client {
	return &Client{http: httpClient}
}

// FetchOrderBook returns asks and bids for symbol usdta7a5.
func (c *Client) FetchOrderBook(ctx context.Context) (OrderBook, error) {
	ctx, span := clientTracer.Start(ctx, "grinex.fetch_order_book", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.url", defaultBaseURL+depthPath),
		attribute.String("grinex.symbol", defaultSymbol),
	)

	var payload depthResponse

	resp, err := c.http.R().
		SetContext(ctx).
		SetQueryParam("symbol", defaultSymbol).
		SetResult(&payload).
		Get(depthPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
		return OrderBook{}, fmt.Errorf("grinex request failed: %w", err)
	}
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode()))

	if resp.IsError() {
		span.SetStatus(codes.Error, "grinex returned error status")
		return OrderBook{}, fmt.Errorf("grinex returned status %d", resp.StatusCode())
	}

	asks, err := parseLevels(payload.Asks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse asks failed")
		return OrderBook{}, fmt.Errorf("parse asks: %w", err)
	}

	bids, err := parseLevels(payload.Bids)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse bids failed")
		return OrderBook{}, fmt.Errorf("parse bids: %w", err)
	}

	span.SetAttributes(
		attribute.Int("grinex.asks.count", len(asks)),
		attribute.Int("grinex.bids.count", len(bids)),
	)
	span.SetStatus(codes.Ok, "ok")

	return OrderBook{Asks: asks, Bids: bids}, nil
}

func parseLevels(raw []depthLevel) ([]Level, error) {
	levels := make([]Level, 0, len(raw))
	for i, item := range raw {
		if item.Price == "" || item.Amount == "" {
			return nil, fmt.Errorf("invalid level at index %d: expected price and amount", i)
		}

		price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price at index %d: %w", i, err)
		}

		amount, err := strconv.ParseFloat(item.Amount, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid amount at index %d: %w", i, err)
		}

		levels = append(levels, Level{Price: price, Amount: amount})
	}

	return levels, nil
}
