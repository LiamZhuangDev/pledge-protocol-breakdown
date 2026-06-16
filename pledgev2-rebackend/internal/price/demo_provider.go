package price

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DemoProvider struct {
	quotes map[string]Quote
	now    func() time.Time
}

func NewDemoProvider() *DemoProvider {
	return &DemoProvider{
		quotes: map[string]Quote{
			"PLGR": {
				Symbol:   "PLGR",
				Currency: "USDT",
				Price:    "0.0027",
				Source:   "demo",
			},
		},
		now: time.Now,
	}
}

func (p *DemoProvider) Latest(_ context.Context, symbol string) (Quote, error) {
	quote, ok := p.quotes[strings.ToUpper(symbol)]
	if !ok {
		return Quote{}, fmt.Errorf("price for %s not found", symbol)
	}

	quote.UpdatedAt = p.now().UTC()
	return quote, nil
}
