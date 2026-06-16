package price

import (
	"context"
	"testing"
	"time"
)

func TestLatestNormalizesSymbol(t *testing.T) {
	provider := NewDemoProvider()
	expectedTime := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	provider.now = func() time.Time { return expectedTime }

	service := NewService(provider)

	quote, err := service.Latest(context.Background(), " plgr ")
	if err != nil {
		t.Fatalf("latest price: %v", err)
	}

	if quote.Symbol != "PLGR" {
		t.Fatalf("expected normalized symbol PLGR, got %s", quote.Symbol)
	}
	if quote.Price != "0.0027" {
		t.Fatalf("expected demo price, got %s", quote.Price)
	}
	if quote.UpdatedAt != expectedTime {
		t.Fatalf("expected updated time %s, got %s", expectedTime, quote.UpdatedAt)
	}
}

func TestLatestRejectsBlankSymbol(t *testing.T) {
	service := NewService(NewDemoProvider())

	_, err := service.Latest(context.Background(), " ")
	if err == nil {
		t.Fatal("expected blank symbol error")
	}
}
