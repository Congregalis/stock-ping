package stock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Quote represents stock quote data from Finnhub
type Quote struct {
	Symbol        string
	CurrentPrice  float64 // c - Current price
	Change        float64 // d - Change
	PercentChange float64 // dp - Percent change
	High          float64 // h - High price of the day
	Low           float64 // l - Low price of the day
	Open          float64 // o - Open price of the day
	PrevClose     float64 // pc - Previous close price
	Timestamp     int64   // t - Timestamp
}

// finnhubResponse is the raw API response structure
type finnhubResponse struct {
	C  float64 `json:"c"`  // Current price
	D  float64 `json:"d"`  // Change
	DP float64 `json:"dp"` // Percent change
	H  float64 `json:"h"`  // High
	L  float64 `json:"l"`  // Low
	O  float64 `json:"o"`  // Open
	PC float64 `json:"pc"` // Previous close
	T  int64   `json:"t"`  // Timestamp
}

// Candle represents historical stock data (candles)
type Candle struct {
	C []float64 `json:"c"` // List of close prices
	H []float64 `json:"h"` // List of high prices
	L []float64 `json:"l"` // List of low prices
	O []float64 `json:"o"` // List of open prices
	S string    `json:"s"` // Status of the response
	T []int64   `json:"t"` // List of timestamp
	V []float64 `json:"v"` // List of volume data
}

// Client is a Finnhub API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Finnhub client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://finnhub.io/api/v1",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetQuote fetches the current quote for a symbol
func (c *Client) GetQuote(symbol string, market string) (*Quote, error) {
	// Use Finnhub for US market, Yahoo for others
	if market == MarketUS || market == "" {
		return c.getFinnhubQuote(symbol)
	}
	return FetchYahooQuote(symbol)
}

func (c *Client) getFinnhubQuote(symbol string) (*Quote, error) {
	url := fmt.Sprintf("%s/quote?symbol=%s&token=%s", c.baseURL, symbol, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var data finnhubResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if we got valid data (c=0 usually means invalid symbol)
	if data.C == 0 && data.PC == 0 {
		return nil, fmt.Errorf("no data available for symbol: %s", symbol)
	}

	return &Quote{
		Symbol:        symbol,
		CurrentPrice:  data.C,
		Change:        data.D,
		PercentChange: data.DP,
		High:          data.H,
		Low:           data.L,
		Open:          data.O,
		PrevClose:     data.PC,
		Timestamp:     data.T,
	}, nil
}

// GetCandles fetches historical candle data
// NOTE: Uses Yahoo Finance (Free) instead of Finnhub (Paid)
func (c *Client) GetCandles(symbol string, resolution string, from, to int64) (*Candle, error) {
	// We ignore resolution (hardcoded to 1d in Yahoo for now as per requirement)
	// and API key since we use free Yahoo API.
	return FetchYahooCandles(symbol, from, to)
}

// FormatQuote returns a formatted string representation of the quote
func (q *Quote) FormatQuote(name string) string {
	displayName := q.Symbol
	if name != "" {
		displayName = fmt.Sprintf("%s (%s)", q.Symbol, name)
	}

	changeSign := ""
	if q.Change >= 0 {
		changeSign = "+"
	}

	return fmt.Sprintf(`📈 %s
   价格: $%.2f
   涨跌: %s$%.2f (%s%.2f%%)
   今日: $%.2f ~ $%.2f`,
		displayName,
		q.CurrentPrice,
		changeSign, q.Change, changeSign, q.PercentChange,
		q.Low, q.High)
}
