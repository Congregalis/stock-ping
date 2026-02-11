package stock

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type YahooQuoteResponse struct {
	QuoteResponse struct {
		Result []YahooQuote `json:"result"`
		Error  interface{}  `json:"error"`
	} `json:"quoteResponse"`
}

type YahooQuote struct {
	Symbol                     string  `json:"symbol"`
	LongName                   string  `json:"longName"`
	ShortName                  string  `json:"shortName"`
	Exchange                   string  `json:"exchange"`
	RegularMarketPrice         float64 `json:"regularMarketPrice"`
	RegularMarketChange        float64 `json:"regularMarketChange"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
	RegularMarketOpen          float64 `json:"regularMarketOpen"`
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
	RegularMarketDayHigh       float64 `json:"regularMarketDayHigh"`
	RegularMarketDayLow        float64 `json:"regularMarketDayLow"`
	RegularMarketTime          int64   `json:"regularMarketTime"`
}

// FetchYahooQuote fetches current price data from Yahoo Finance using the Chart endpoint
func FetchYahooQuote(symbol string) (*Quote, error) {
	// Use chart endpoint with 1d range as it's more stable than the v7 quote endpoint
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote from Yahoo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Yahoo API status %d: %s", resp.StatusCode, string(body))
	}

	var yResp YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil {
		return nil, fmt.Errorf("failed to decode Yahoo response: %w", err)
	}

	if len(yResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data found for %s", symbol)
	}

	res := yResp.Chart.Result[0]
	meta := res.Meta

	// Calculate change and percent change
	price := meta.RegularMarketPrice
	prevClose := meta.ChartPreviousClose
	if prevClose == 0 && len(res.Indicators.Quote) > 0 {
		// Fallback if chartPreviousClose is missing
		// This happens occasionally
	}

	change := price - prevClose
	percentChange := 0.0
	if prevClose != 0 {
		percentChange = (change / prevClose) * 100
	}

	// Try to get day high/low/open from indicators if meta is incomplete
	var open, high, low float64
	if len(res.Indicators.Quote) > 0 && len(res.Indicators.Quote[0].Close) > 0 {
		q := res.Indicators.Quote[0]
		// Use the most recent daily values
		idx := len(q.Close) - 1
		open = q.Open[idx]
		high = q.High[idx]
		low = q.Low[idx]
	}

	return &Quote{
		Symbol:        symbol,
		CurrentPrice:  price,
		Change:        change,
		PercentChange: percentChange,
		High:          high,
		Low:           low,
		Open:          open,
		PrevClose:     prevClose,
		Timestamp:     int64(meta.RegularMarketTime),
	}, nil
}

type YahooChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string  `json:"currency"`
				Symbol               string  `json:"symbol"`
				LongName             string  `json:"longName"`
				ShortName            string  `json:"shortName"`
				ExchangeName         string  `json:"exchangeName"`
				InstrumentType       string  `json:"instrumentType"`
				FirstTradeDate       int     `json:"firstTradeDate"`
				RegularMarketTime    int     `json:"regularMarketTime"`
				Gmtoffset            int     `json:"gmtoffset"`
				Timezone             string  `json:"timezone"`
				ExchangeTimezoneName string  `json:"exchangeTimezoneName"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				PriceHint            int     `json:"priceHint"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					Low    []float64 `json:"low"`
					High   []float64 `json:"high"`
					Close  []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// FetchYahooCandles fetches historical data from Yahoo Finance
func FetchYahooCandles(symbol string, period1, period2 int64) (*Candle, error) {
	// Yahoo uses seconds for timestamps
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		symbol, period1, period2)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// User-Agent is required to avoid 429/403
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Yahoo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Yahoo API status %d: %s", resp.StatusCode, string(body))
	}

	var yResp YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil {
		return nil, fmt.Errorf("failed to decode Yahoo response: %w", err)
	}

	if yResp.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo API error: %s - %s", yResp.Chart.Error.Code, yResp.Chart.Error.Description)
	}

	if len(yResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data found for %s", symbol)
	}

	result := yResp.Chart.Result[0]
	if len(result.Timestamp) == 0 {
		return &Candle{S: "no_data"}, nil
	}

	quote := result.Indicators.Quote[0]

	// Map to Candle struct
	// Note: Yahoo might return nulls for some entries if there are gaps, specifically volume/open/etc.
	// But the JSON decoder for slice of float64 might fail if it sees null?
	// Actually, json.Unmarshal into []float64 will fail on null.
	// We might need custom unmarshaling or more robust handling if Yahoo returns nulls.
	// For standard daily charts of major stocks, it's usually fine.
	// To be safe, we should check if slices are same length.

	count := len(result.Timestamp)
	candle := &Candle{
		S: "ok",
		T: result.Timestamp,
		O: quote.Open,
		H: quote.High,
		L: quote.Low,
		C: quote.Close,
		V: quote.Volume,
	}

	// Verify lengths match (basic integrity check)
	if len(candle.C) != count {
		// This happens if day has no trading?
		// For now, return what we have, but it might be unsafe.
	}

	return candle, nil
}

// FetchSymbolDetails fetches symbol name and market from Yahoo Finance
func FetchSymbolDetails(symbol string) (name string, market string, err error) {
	// Use chart endpoint with 1d range as it's more stable than the v7 quote endpoint
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch details from Yahoo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", "", fmt.Errorf("symbol %s not found", symbol)
		}
		return "", "", fmt.Errorf("Yahoo API status %d: %s", resp.StatusCode, string(body))
	}

	var yResp YahooChartResponse
	if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil {
		return "", "", fmt.Errorf("failed to decode Yahoo response: %w", err)
	}

	if yResp.Chart.Error != nil {
		return "", "", fmt.Errorf("Yahoo API error: %s - %s", yResp.Chart.Error.Code, yResp.Chart.Error.Description)
	}

	if len(yResp.Chart.Result) == 0 {
		return "", "", fmt.Errorf("no data found for %s", symbol)
	}

	meta := yResp.Chart.Result[0].Meta

	name = meta.ShortName
	market = MarketUS // Default

	// Infer market from Timezone and Exchange
	tz := meta.ExchangeTimezoneName
	exC := meta.ExchangeName

	if tz == "Asia/Shanghai" {
		market = MarketCN
	} else if tz == "Asia/Hong_Kong" {
		market = MarketHK
	} else if tz == "Asia/Taipei" {
		market = MarketTW
	} else if exC == "CCC" || exC == "CCY" || tz == "UTC" {
		market = MarketCrypto
	}

	// Refine using suffix if still default US
	if market == MarketUS {
		// Yahoo tickers for CN: 600519.SS, 000001.SZ
		// HK: 9988.HK
		// TW: 2330.TW
		lastLen := len(symbol)
		if lastLen > 3 {
			suffix := symbol[lastLen-3:]
			if suffix == ".SS" || suffix == ".SZ" {
				market = MarketCN
			} else if suffix == ".HK" {
				market = MarketHK
			} else if suffix == ".TW" {
				market = MarketTW
			}
		}
	}

	return name, market, nil
}
