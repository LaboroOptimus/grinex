package grinex

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"
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
	var payload depthResponse

	resp, err := c.http.R().
		SetContext(ctx).
		SetQueryParam("symbol", defaultSymbol).
		SetResult(&payload).
		Get(depthPath)
	if err != nil {
		return OrderBook{}, fmt.Errorf("grinex request failed: %w", err)
	}

	if resp.IsError() {
		return OrderBook{}, fmt.Errorf("grinex returned status %d", resp.StatusCode())
	}

	asks, err := parseLevels(payload.Asks)
	if err != nil {
		return OrderBook{}, fmt.Errorf("parse asks: %w", err)
	}

	bids, err := parseLevels(payload.Bids)
	if err != nil {
		return OrderBook{}, fmt.Errorf("parse bids: %w", err)
	}

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
