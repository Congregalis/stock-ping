package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/notify"
	"github.com/congregalis/stock-ping/rule"
	"github.com/congregalis/stock-ping/stock"
	table "github.com/evertras/bubble-table/table"
)

// ViewMode enum
const (
	ViewPortfolio = iota
	ViewDashboard
	ViewTrend
)

const (
	positionActionNone = iota
	positionActionBuy
	positionActionSell
	positionActionEdit
	positionActionDelete
)

// StockData holds the current state of a stock
type StockData struct {
	Symbol        string
	Name          string
	Price         float64
	Change        float64
	PrevClose     float64
	Open          float64
	High          float64
	Low           float64
	LastUpdate    time.Time
	Triggered     bool
	TriggerReason string
	Error         string
	Market        string
	// Portfolio fields
	Quantity    float64
	CostPrice   float64
	RealizedPnL float64
}

// Model represents the TUI application state
type Model struct {
	// Core State
	viewMode       int
	previousView   int
	selectedSymbol string

	// Data
	stocks     map[string]*StockData
	stockOrder []string

	// Trend View State
	trendData    *stock.Candle
	trendLoading bool
	trendError   error

	// Services
	cfg         *config.Config
	stockClient *stock.Client
	notifier    *notify.Notifier
	evaluator   *rule.Evaluator

	// Components
	table          table.Model
	portfolioTable table.Model
	help           help.Model
	keys           keyMap

	// Internal State
	triggeredState map[string]map[string]bool
	lastRefresh    time.Time
	configPath     string
	statusMessage  string
	width          int
	height         int
	quitting       bool
	sortAscending  bool
	showSplash     bool
	holdingsCount  int
	privacyMode    bool

	// Portfolio action state
	positionAction   int
	positionSymbol   string
	positionStep     int
	positionQuantity float64
	positionInput    textinput.Model
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, stockClient *stock.Client, notifier *notify.Notifier, configPath string) Model {
	// Dashboard table columns
	columns := []table.Column{
		table.NewFlexColumn("symbol", "Symbol", 2),
		table.NewColumn("price", "Price", 12),
		table.NewColumn("change", "Change", 25),
		table.NewColumn("open", "Open", 12),
		table.NewColumn("day_range", "Day Range", 25),
		table.NewColumn("prev_close", "Prev Close", 12),
		table.NewColumn("updated", "Updated", 10),
	}

	t := table.New(columns).
		HeaderStyle(tableHeaderStyle).
		HighlightStyle(tableSelectedStyle).
		BorderRounded().
		Focused(true).
		WithPageSize(20)

	// Portfolio table columns (Optimized for space)
	portfolioColumns := []table.Column{
		table.NewFlexColumn("symbol", "Symbol", 2),
		table.NewColumn("price", "Price", 12),
		table.NewColumn("change", "Change", 25),
		table.NewColumn("quantity", "Quantity", 10),
		table.NewColumn("cost", "Cost", 12),
		table.NewColumn("pl", "P/L", 25),
	}

	pt := table.New(portfolioColumns).
		HeaderStyle(tableHeaderStyle).
		HighlightStyle(tableSelectedStyle).
		BorderRounded().
		Focused(true).
		WithPageSize(20)

	positionInput := textinput.New()
	positionInput.CharLimit = 32
	positionInput.Width = 24

	stocks := make(map[string]*StockData)
	var stockOrder []string
	for _, r := range cfg.Rules {
		stocks[r.Symbol] = &StockData{
			Symbol: r.Symbol,
			Name:   r.Name,
			Market: r.Market,
		}
		stockOrder = append(stockOrder, r.Symbol)
	}

	// Load holdings into stocks
	holdingsCount := 0
	for _, h := range cfg.Holdings {
		if data, ok := stocks[h.Symbol]; ok {
			data.Quantity = h.Quantity
			data.CostPrice = h.CostPrice
			data.RealizedPnL = h.RealizedPnL
			holdingsCount++
		}
	}

	m := Model{
		viewMode:       ViewPortfolio, // Default to portfolio view
		table:          t,
		portfolioTable: pt,
		help:           help.New(),
		keys:           defaultKeyMap,
		stocks:         stocks,
		stockOrder:     stockOrder,
		cfg:            cfg,
		stockClient:    stockClient,
		notifier:       notifier,
		evaluator:      rule.NewEvaluator(),
		triggeredState: make(map[string]map[string]bool),
		configPath:     configPath,
		sortAscending:  false, // Default to Descending
		showSplash:     true,
		holdingsCount:  holdingsCount,
		positionInput:  positionInput,
	}

	// Apply initial sort (Change Descending)
	m.SortByChange()
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
			return splashTimeoutMsg{}
		}),
		m.tickCmd(),
		m.refreshAllStocks(true),
	)
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(m.cfg.Interval)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) refreshAllStocks(force bool) tea.Cmd {
	var cmds []tea.Cmd
	for _, symbol := range m.stockOrder {
		s := symbol

		// Check if market is open for this stock
		// If data exists, check its market. If not, look up rule.
		market := stock.MarketUS
		if data, ok := m.stocks[s]; ok && data.Market != "" {
			market = data.Market
		} else if rule := m.cfg.GetRule(s); rule != nil && rule.Market != "" {
			market = rule.Market
		}

		if force || stock.IsMarketOpen(market) {
			cmds = append(cmds, func() tea.Msg {
				quote, err := m.stockClient.GetQuote(s, market)
				return stockUpdateMsg{symbol: s, quote: quote, err: err}
			})
		}
	}
	return tea.Batch(cmds...)
}

func (m Model) fetchTrendData(symbol string) tea.Cmd {
	return func() tea.Msg {
		// Fetch Daily resolution for last 30 days
		to := time.Now().Unix()
		from := time.Now().AddDate(0, 0, -30).Unix()

		candles, err := m.stockClient.GetCandles(symbol, "D", from, to)
		return candleUpdateMsg{symbol: symbol, candles: candles, err: err}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust page size based on height
		// Header + Footer + Padding estimation
		pageSize := msg.Height - 12
		if pageSize < 1 {
			pageSize = 1
		}

		m.table = m.table.WithTargetWidth(msg.Width - 4).WithPageSize(pageSize)
		m.portfolioTable = m.portfolioTable.WithTargetWidth(msg.Width - 4).WithPageSize(pageSize)

	case tea.KeyMsg:
		if m.showSplash {
			m.showSplash = false
			return m, nil
		}

		if m.positionAction != positionActionNone {
			return m, m.handlePositionActionKey(msg)
		}

		// Common keys
		if key.Matches(msg, m.keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

		if key.Matches(msg, m.keys.Privacy) {
			m.privacyMode = !m.privacyMode
			m.updatePortfolioTableRows()
			if m.privacyMode {
				m.statusMessage = "🙈 Privacy Mode: ON"
			} else {
				m.statusMessage = "🐵 Privacy Mode: OFF"
			}
			return m, nil
		}

		// View specific keys
		if m.viewMode == ViewPortfolio {
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.statusMessage = "Refreshing..."
				return m, m.refreshAllStocks(true)
			case key.Matches(msg, m.keys.Sort):
				m.sortAscending = !m.sortAscending
				m.SortByChange()
				orderStr := "Descending"
				if m.sortAscending {
					orderStr = "Ascending"
				}
				m.statusMessage = "Sorted: " + orderStr
				return m, nil
			case key.Matches(msg, m.keys.Dashboard):
				m.viewMode = ViewDashboard
				return m, nil
			case key.Matches(msg, m.keys.Buy):
				return m, m.startPositionAction(positionActionBuy)
			case key.Matches(msg, m.keys.Sell):
				return m, m.startPositionAction(positionActionSell)
			case key.Matches(msg, m.keys.Edit):
				return m, m.startPositionAction(positionActionEdit)
			case key.Matches(msg, m.keys.Delete):
				return m, m.startPositionAction(positionActionDelete)
			case key.Matches(msg, m.keys.Select):
				// Switch to Trend View from portfolio
				selectedRow := m.portfolioTable.HighlightedRow()
				if selectedRow.Data != nil {
					raw, _ := selectedRow.Data["symbol"].(string)
					symbol := parseSymbolFromTable(raw)
					if symbol == "" {
						return m, nil
					}
					m.selectedSymbol = symbol
					m.previousView = ViewPortfolio
					m.viewMode = ViewTrend
					m.trendLoading = true
					m.trendData = nil
					m.trendError = nil
					return m, m.fetchTrendData(symbol)
				}
			}
		} else if m.viewMode == ViewDashboard {
			switch {
			case key.Matches(msg, m.keys.Refresh):
				m.statusMessage = "Refreshing..."
				return m, m.refreshAllStocks(true)
			case key.Matches(msg, m.keys.Sort):
				m.sortAscending = !m.sortAscending
				m.SortByChange()
				orderStr := "Descending"
				if m.sortAscending {
					orderStr = "Ascending"
				}
				m.statusMessage = "Sorted: " + orderStr
				return m, nil
			case key.Matches(msg, m.keys.Portfolio):
				m.viewMode = ViewPortfolio
				return m, nil
			case key.Matches(msg, m.keys.Select):
				// Switch to Trend View
				selectedRow := m.table.HighlightedRow()
				if selectedRow.Data != nil {
					raw, _ := selectedRow.Data["symbol"].(string)
					symbol := parseSymbolFromTable(raw)
					if symbol == "" {
						return m, nil
					}

					m.selectedSymbol = symbol
					m.previousView = ViewDashboard
					m.viewMode = ViewTrend
					m.trendLoading = true
					m.trendData = nil
					m.trendError = nil
					return m, m.fetchTrendData(symbol)
				}
			}
		} else if m.viewMode == ViewTrend {
			switch {
			case key.Matches(msg, m.keys.Back):
				if m.previousView == ViewDashboard {
					m.viewMode = ViewDashboard
				} else {
					m.viewMode = ViewPortfolio
				}
				m.selectedSymbol = ""
				return m, nil
			case key.Matches(msg, m.keys.Portfolio):
				m.viewMode = ViewPortfolio
				m.selectedSymbol = ""
				return m, nil
			case key.Matches(msg, m.keys.Dashboard):
				m.viewMode = ViewDashboard
				m.selectedSymbol = ""
				return m, nil
			}
		}

	case tickMsg:
		cmds = append(cmds, m.refreshAllStocks(false))
		cmds = append(cmds, m.tickCmd())

	case stockUpdateMsg:
		m.updateStock(msg)
		m.SortByChange()
		m.lastRefresh = time.Now()
		m.statusMessage = ""

	case candleUpdateMsg:
		if msg.symbol == m.selectedSymbol {
			m.trendLoading = false
			if msg.err != nil {
				m.trendError = msg.err
			} else {
				m.trendData = msg.candles
			}
		}

	case configReloadMsg:
		m.cfg = msg.cfg
		m.reloadRules()
		m.statusMessage = "🔄 Config reloaded"

	case splashTimeoutMsg:
		m.showSplash = false
	}

	// Update components based on view
	if m.viewMode == ViewPortfolio {
		m.portfolioTable, cmd = m.portfolioTable.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.viewMode == ViewDashboard {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateStock(msg stockUpdateMsg) {
	data, ok := m.stocks[msg.symbol]
	if !ok {
		return
	}

	if msg.err != nil {
		data.Error = msg.err.Error()
		return
	}

	data.Price = msg.quote.CurrentPrice
	data.Change = msg.quote.PercentChange
	data.PrevClose = msg.quote.PrevClose
	data.Open = msg.quote.Open
	data.High = msg.quote.High
	data.Low = msg.quote.Low
	data.LastUpdate = time.Now()
	data.Error = ""

	// Evaluate rules
	r := m.cfg.GetRule(msg.symbol)
	if r == nil {
		return
	}

	result := m.evaluator.Evaluate(r, msg.quote)

	// Build current conditions map
	currentConditions := make(map[string]bool)
	for _, reason := range result.Reasons {
		currentConditions[reason] = true
	}

	// Initialize state if needed
	if m.triggeredState[msg.symbol] == nil {
		m.triggeredState[msg.symbol] = make(map[string]bool)
	}

	// Find newly triggered conditions
	var newlyTriggered []string
	for reason := range currentConditions {
		if !m.triggeredState[msg.symbol][reason] {
			newlyTriggered = append(newlyTriggered, reason)
		}
	}

	// Update state
	m.triggeredState[msg.symbol] = currentConditions

	data.Triggered = result.Triggered()
	if len(result.Reasons) > 0 {
		data.TriggerReason = result.Reasons[0]
	} else {
		data.TriggerReason = ""
	}

	// Send notification only for new triggers
	if len(newlyTriggered) > 0 && m.notifier.IsConfigured() {
		title, body := result.FormatNotification()
		m.notifier.SendWithGroup(title, body, "stock-ping")
	}
}

func (m *Model) updateTableRows() {
	var rows []table.Row
	for _, symbol := range m.stockOrder {
		data := m.stocks[symbol]
		if data == nil {
			continue
		}

		displayName := data.Symbol
		if data.Name != "" {
			displayName = fmt.Sprintf("%s(%s)", data.Symbol, data.Name)
		}

		priceStr := "--"
		changeStr := "--"
		openStr := "--"
		dayRangeStr := "--"
		prevCloseStr := "--"
		updatedStr := "--"

		if data.Error != "" {
			displayName = "❌ " + displayName
			updatedStr = "ERROR"
		} else if data.Price > 0 {
			priceStr = fmt.Sprintf("$%.2f", data.Price)
			openStr = fmt.Sprintf("$%.2f", data.Open)
			prevCloseStr = fmt.Sprintf("$%.2f", data.PrevClose)
			dayRangeStr = fmt.Sprintf("$%.2f ~ $%.2f", data.Low, data.High)

			// Calculate price change accurately
			var priceChange float64
			if data.PrevClose > 0 {
				priceChange = data.Price - data.PrevClose
			} else {
				priceChange = data.Price * data.Change / (100 + data.Change)
			}

			if data.Change >= 0 {
				changeStr = greenStyle.Render(fmt.Sprintf("+$%.2f (+%.2f%%)", priceChange, data.Change))
			} else {
				changeStr = redStyle.Render(fmt.Sprintf("-$%.2f (%.2f%%)", -priceChange, data.Change))
			}

			if !data.LastUpdate.IsZero() {
				updatedStr = data.LastUpdate.Format("15:04:05")
			}
		}

		rows = append(rows, table.NewRow(table.RowData{
			"symbol":     displayName,
			"price":      priceStr,
			"change":     changeStr,
			"open":       openStr,
			"day_range":  dayRangeStr,
			"prev_close": prevCloseStr,
			"updated":    updatedStr,
		}))
	}
	m.table = m.table.WithRows(rows)
}

func (m *Model) reloadRules() {
	// Rebuild stocks map from new config
	newStocks := make(map[string]*StockData)
	var newOrder []string
	for _, r := range m.cfg.Rules {
		if existing, ok := m.stocks[r.Symbol]; ok {
			existing.Name = r.Name
			existing.Market = r.Market
			existing.Quantity = 0
			existing.CostPrice = 0
			existing.RealizedPnL = 0
			newStocks[r.Symbol] = existing
		} else {
			newStocks[r.Symbol] = &StockData{
				Symbol: r.Symbol,
				Name:   r.Name,
				Market: r.Market,
			}
		}
		newOrder = append(newOrder, r.Symbol)
	}
	// Reload holdings
	holdingsCount := 0
	for _, h := range m.cfg.Holdings {
		if data, ok := newStocks[h.Symbol]; ok {
			data.Quantity = h.Quantity
			data.CostPrice = h.CostPrice
			data.RealizedPnL = h.RealizedPnL
			holdingsCount++
		}
	}
	m.stocks = newStocks
	m.stockOrder = newOrder
	m.holdingsCount = holdingsCount
	// Re-apply sort
	m.SortByChange()
}

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "👋 Goodbye!\n"
	}

	if m.showSplash {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			logoStyle.Render(splashLogo),
		)
	}

	switch m.viewMode {
	case ViewPortfolio:
		return m.ViewPortfolio()
	case ViewDashboard:
		return m.ViewDashboard()
	case ViewTrend:
		return m.ViewTrend()
	default:
		return "Unknown view"
	}
}

// ReloadConfig sends a config reload message
func ReloadConfig(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		return configReloadMsg{cfg: cfg}
	}
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// SortBySymbol sorts stocks alphabetically
func (m *Model) SortBySymbol() {
	sort.Strings(m.stockOrder)
	m.updateTableRows()
	m.updatePortfolioTableRows()
}

// SortByChange sorts stocks by percent change
func (m *Model) SortByChange() {
	sort.Slice(m.stockOrder, func(i, j int) bool {
		si, sj := m.stocks[m.stockOrder[i]], m.stocks[m.stockOrder[j]]
		if m.sortAscending {
			return si.Change < sj.Change
		}
		return si.Change > sj.Change
	})
	m.updateTableRows()
	m.updatePortfolioTableRows()
}

func (m *Model) updatePortfolioTableRows() {
	var rows []table.Row
	for _, symbol := range m.stockOrder {
		data := m.stocks[symbol]
		if data == nil || data.Quantity <= 0 {
			continue
		}

		displayName := data.Symbol
		if data.Name != "" {
			displayName = fmt.Sprintf("%s(%s)", data.Symbol, data.Name)
		}

		priceStr := "--"
		changeStr := "--"
		quantityStr := fmt.Sprintf("%.2f", data.Quantity)
		costStr := fmt.Sprintf("$%.2f", data.CostPrice)
		plStr := "--"

		if m.privacyMode {
			quantityStr = "****"
			costStr = "****"
		}

		if data.Error != "" {
			displayName = "❌ " + displayName
		} else if data.Price > 0 {
			priceStr = fmt.Sprintf("$%.2f", data.Price)

			// Calculate daily change display
			var priceChange float64
			if data.PrevClose > 0 {
				priceChange = data.Price - data.PrevClose
			} else {
				priceChange = data.Price * data.Change / (100 + data.Change)
			}

			if data.Change >= 0 {
				changeStr = greenStyle.Render(fmt.Sprintf("+$%.2f (+%.2f%%)", priceChange, data.Change))
			} else {
				changeStr = redStyle.Render(fmt.Sprintf("-$%.2f (%.2f%%)", -priceChange, data.Change))
			}
		}

		// Portfolio calculations (Total P/L = Realized + Unrealized)
		unrealizedPnL := 0.0
		if data.Price > 0 {
			unrealizedPnL = (data.Price - data.CostPrice) * data.Quantity
		}
		totalPnL := data.RealizedPnL + unrealizedPnL
		plPercent := 0.0
		cost := data.CostPrice * data.Quantity
		if cost > 0 {
			plPercent = (totalPnL / cost) * 100
		}

		visPl := ""
		if m.privacyMode {
			visPl = "****"
		} else {
			absPl := totalPnL
			if totalPnL < 0 {
				absPl = -totalPnL
			}
			visPl = fmt.Sprintf("$%.2f", absPl)
		}

		absPlPercent := plPercent
		if plPercent < 0 {
			absPlPercent = -plPercent
		}

		if totalPnL >= 0 {
			plStr = greenStyle.Render(fmt.Sprintf("+%s (+%.2f%%)", visPl, absPlPercent))
		} else {
			plStr = redStyle.Render(fmt.Sprintf("-%s (-%.2f%%)", visPl, absPlPercent))
		}

		rows = append(rows, table.NewRow(table.RowData{
			"symbol":   displayName,
			"price":    priceStr,
			"change":   changeStr,
			"quantity": quantityStr,
			"cost":     costStr,
			"pl":       plStr,
		}))
	}
	m.portfolioTable = m.portfolioTable.WithRows(rows)
}

func (m Model) selectedPortfolioSymbol() (string, bool) {
	selectedRow := m.portfolioTable.HighlightedRow()
	if selectedRow.Data == nil {
		return "", false
	}

	raw, ok := selectedRow.Data["symbol"].(string)
	if !ok {
		return "", false
	}

	symbol := parseSymbolFromTable(raw)
	if symbol == "" {
		return "", false
	}

	return symbol, true
}

func (m *Model) startPositionAction(action int) tea.Cmd {
	symbol, ok := m.selectedPortfolioSymbol()
	if !ok {
		m.statusMessage = "No holding selected"
		return nil
	}

	m.positionAction = action
	m.positionSymbol = symbol
	m.positionStep = 0
	m.positionQuantity = 0

	if action != positionActionDelete {
		m.positionInput.SetValue("")
		m.positionInput.Focus()
		return textinput.Blink
	}

	return nil
}

func (m *Model) resetPositionAction() {
	m.positionAction = positionActionNone
	m.positionSymbol = ""
	m.positionStep = 0
	m.positionQuantity = 0
	m.positionInput.SetValue("")
	m.positionInput.Blur()
}

func (m *Model) handlePositionActionKey(msg tea.KeyMsg) tea.Cmd {
	if msg.Type == tea.KeyEsc {
		m.resetPositionAction()
		m.statusMessage = "Operation canceled"
		return nil
	}

	if msg.Type == tea.KeyEnter {
		m.confirmPositionAction()
		return nil
	}

	if m.positionAction == positionActionDelete {
		return nil
	}

	var cmd tea.Cmd
	m.positionInput, cmd = m.positionInput.Update(msg)
	return cmd
}

func (m *Model) confirmPositionAction() {
	if m.positionAction == positionActionDelete {
		m.deleteSelectedHolding()
		return
	}

	inputValue := strings.TrimSpace(m.positionInput.Value())
	if inputValue == "" {
		m.statusMessage = "Please enter a number"
		return
	}

	parsedValue, err := strconv.ParseFloat(inputValue, 64)
	if err != nil {
		m.statusMessage = "Invalid number format"
		return
	}

	if m.positionStep == 0 {
		if parsedValue <= 0 {
			m.statusMessage = "Quantity must be greater than 0"
			return
		}

		if m.positionAction == positionActionSell {
			holding := m.cfg.GetHolding(m.positionSymbol)
			if holding == nil {
				m.statusMessage = fmt.Sprintf("Holding %s not found", m.positionSymbol)
				m.resetPositionAction()
				return
			}
			if parsedValue > holding.Quantity+1e-9 {
				m.statusMessage = "Sell quantity exceeds current holding"
				return
			}
		}

		m.positionQuantity = parsedValue
		m.positionStep = 1
		m.positionInput.SetValue("")
		return
	}

	if parsedValue < 0 {
		m.statusMessage = "Price must be greater than or equal to 0"
		return
	}

	m.applyPositionChange(parsedValue)
	m.resetPositionAction()
}

func (m *Model) applyPositionChange(price float64) {
	if m.positionSymbol == "" {
		m.statusMessage = "No holding selected"
		return
	}

	previousHoldings := append([]config.Holding(nil), m.cfg.Holdings...)
	holding := m.cfg.GetHolding(m.positionSymbol)
	if holding == nil {
		m.statusMessage = fmt.Sprintf("Holding %s not found", m.positionSymbol)
		return
	}

	quantity := m.positionQuantity
	successMessage := ""

	switch m.positionAction {
	case positionActionBuy:
		newTotalQty := holding.Quantity + quantity
		newAvgCost := ((holding.Quantity * holding.CostPrice) + (quantity * price)) / newTotalQty
		holding.Quantity = newTotalQty
		holding.CostPrice = newAvgCost
		successMessage = fmt.Sprintf("Bought %s %s @ %.2f", formatQuantityValue(quantity), m.positionSymbol, price)
	case positionActionSell:
		if quantity > holding.Quantity+1e-9 {
			m.statusMessage = "Sell quantity exceeds current holding"
			return
		}

		avgCost := holding.CostPrice
		remainingQty := holding.Quantity - quantity
		realizedDelta := (price - avgCost) * quantity

		if remainingQty <= 1e-9 {
			m.cfg.RemoveHolding(m.positionSymbol)
		} else {
			holding.Quantity = remainingQty
			holding.CostPrice = avgCost
			holding.RealizedPnL += realizedDelta
		}

		successMessage = fmt.Sprintf(
			"Sold %s %s @ %.2f, Realized %s",
			formatQuantityValue(quantity),
			m.positionSymbol,
			price,
			formatSignedValue(realizedDelta),
		)
	case positionActionEdit:
		holding.Quantity = quantity
		holding.CostPrice = price
		successMessage = fmt.Sprintf("Updated %s: %s @ %.2f", m.positionSymbol, formatQuantityValue(quantity), price)
	default:
		return
	}

	if err := m.cfg.SaveTo(m.configPath); err != nil {
		m.cfg.Holdings = previousHoldings
		m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
		return
	}

	m.reloadRules()
	m.statusMessage = successMessage
}

func (m *Model) deleteSelectedHolding() {
	if m.positionSymbol == "" {
		m.statusMessage = "No holding selected"
		m.resetPositionAction()
		return
	}

	previousHoldings := append([]config.Holding(nil), m.cfg.Holdings...)
	if !m.cfg.RemoveHolding(m.positionSymbol) {
		m.statusMessage = fmt.Sprintf("Holding %s not found", m.positionSymbol)
		m.resetPositionAction()
		return
	}

	if err := m.cfg.SaveTo(m.configPath); err != nil {
		m.cfg.Holdings = previousHoldings
		m.statusMessage = fmt.Sprintf("Failed to save config: %v", err)
		m.resetPositionAction()
		return
	}

	m.reloadRules()
	m.statusMessage = fmt.Sprintf("Deleted holding %s", m.positionSymbol)
	m.resetPositionAction()
}

func (m Model) positionActionTitle() string {
	switch m.positionAction {
	case positionActionBuy:
		return fmt.Sprintf("Buy %s", m.positionSymbol)
	case positionActionSell:
		return fmt.Sprintf("Sell %s", m.positionSymbol)
	case positionActionEdit:
		return fmt.Sprintf("Edit %s", m.positionSymbol)
	case positionActionDelete:
		return fmt.Sprintf("Delete %s", m.positionSymbol)
	default:
		return ""
	}
}

func (m Model) positionActionPrompt() string {
	if m.positionStep == 0 {
		return "Quantity (> 0)"
	}

	switch m.positionAction {
	case positionActionBuy:
		return "Buy Price (>= 0)"
	case positionActionSell:
		return "Sell Price (>= 0)"
	case positionActionEdit:
		return "Cost Price (>= 0)"
	default:
		return ""
	}
}

func parseSymbolFromTable(raw string) string {
	parts := strings.Split(raw, "(")
	symbol := strings.TrimSpace(parts[0])
	if idx := strings.LastIndex(symbol, " "); idx != -1 {
		symbol = symbol[idx+1:]
	}
	return symbol
}

func formatQuantityValue(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func formatSignedValue(value float64) string {
	if value >= 0 {
		return fmt.Sprintf("+%.2f", value)
	}
	return fmt.Sprintf("%.2f", value)
}
