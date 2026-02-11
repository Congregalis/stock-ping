package tui

import (
	"time"

	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/stock"
)

// Messages
type tickMsg time.Time
type splashTimeoutMsg struct{}
type refreshMsg struct{}
type configReloadMsg struct{ cfg *config.Config }
type stockUpdateMsg struct {
	symbol string
	quote  *stock.Quote
	err    error
}
type marketStatusMsg struct {
	open     bool
	nextOpen time.Time
}

type candleUpdateMsg struct {
	symbol  string
	candles *stock.Candle
	err     error
}
