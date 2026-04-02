package grinex

import "testing"

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
