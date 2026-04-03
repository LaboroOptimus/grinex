package grinex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestParseLevelsSuccess(t *testing.T) {
	raw := []depthLevel{
		{Price: "95.10", Amount: "12.5"},
		{Price: "95.20", Amount: "10.0"},
	}

	levels, err := parseLevels(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(levels))
	}
	if levels[0].Price != 95.10 || levels[0].Amount != 12.5 {
		t.Fatalf("unexpected first level: %+v", levels[0])
	}
}

func TestParseLevelsInvalidShape(t *testing.T) {
	_, err := parseLevels([]depthLevel{{Price: "95.10"}})
	if err == nil {
		t.Fatalf("expected error for invalid level shape")
	}
}

func TestParseLevelsInvalidNumber(t *testing.T) {
	_, err := parseLevels([]depthLevel{{Price: "bad", Amount: "12.5"}})
	if err == nil {
		t.Fatalf("expected error for invalid price")
	}
}

func TestFetchOrderBookSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != depthPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != defaultSymbol {
			t.Fatalf("unexpected symbol query: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"asks":[{"price":"81.24","amount":"10.5"}],
			"bids":[{"price":"81.17","amount":"11.5"}]
		}`))
	}))
	defer server.Close()

	httpClient := resty.New().SetBaseURL(server.URL)
	client := NewClientWithHTTP(httpClient)

	book, err := client.FetchOrderBook(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(book.Asks) != 1 || len(book.Bids) != 1 {
		t.Fatalf("unexpected book levels: %+v", book)
	}
	if book.Asks[0].Price != 81.24 || book.Bids[0].Price != 81.17 {
		t.Fatalf("unexpected rates: %+v", book)
	}
}

func TestFetchOrderBookBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	httpClient := resty.New().SetBaseURL(server.URL)
	client := NewClientWithHTTP(httpClient)

	_, err := client.FetchOrderBook(context.Background())
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}
