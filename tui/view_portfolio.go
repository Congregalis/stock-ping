package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ViewPortfolio renders the portfolio holdings view
func (m Model) ViewPortfolio() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("📈 Portfolio")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Calculate totals
	var totalValue, totalCost, totalPL float64
	for _, symbol := range m.stockOrder {
		data := m.stocks[symbol]
		if data == nil || data.Quantity <= 0 {
			continue
		}
		value := data.Price * data.Quantity
		cost := data.CostPrice * data.Quantity
		totalValue += value
		totalCost += cost
		totalPL += value - cost
	}

	// Summary Cards
	if totalCost > 0 {
		visValue := fmt.Sprintf("$%.2f", totalValue)
		visCost := fmt.Sprintf("$%.2f", totalCost)

		absPL := totalPL
		plStyle := greenStyle
		if totalPL < 0 {
			absPL = -totalPL
			plStyle = redStyle
		}
		visPL := fmt.Sprintf("$%.2f", absPL)

		if m.privacyMode {
			visValue = "****"
			visCost = "****"
			visPL = "****"
		}

		plPercent := (totalPL / totalCost) * 100
		if plPercent < 0 {
			plPercent = -plPercent
		}

		// Cards
		valueCard := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				summaryLabelStyle.Render("TOTAL VALUE"),
				summaryValueStyle.Render(visValue),
			),
		)

		costCard := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				summaryLabelStyle.Render("TOTAL COST"),
				summaryValueStyle.Render(visCost),
			),
		)

		plCardContent := fmt.Sprintf("+%s (+%.2f%%)", visPL, plPercent)
		if totalPL < 0 {
			plCardContent = fmt.Sprintf("-%s (-%.2f%%)", visPL, plPercent)
		}

		plCard := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				summaryLabelStyle.Render("TOTAL P/L"),
				plStyle.Render(plCardContent),
			),
		)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, valueCard, costCard, plCard))
		b.WriteString("\n\n")
	}

	// Portfolio table
	b.WriteString(m.portfolioTable.View())
	b.WriteString("\n\n")

	// Status bar
	statusParts := []string{
		fmt.Sprintf("Interval: %ds", m.cfg.Interval),
		fmt.Sprintf("Holdings: %d", m.holdingsCount),
	}
	if !m.lastRefresh.IsZero() {
		statusParts = append(statusParts, fmt.Sprintf("Last: %s", m.lastRefresh.Format("15:04:05")))
	}
	if m.statusMessage != "" {
		statusParts = append(statusParts, m.statusMessage)
	}

	b.WriteString(statusBarStyle.Render(strings.Join(statusParts, " • ")))
	b.WriteString("\n\n")

	// Help with updated keys hint
	helpView := m.help.View(m.keys)
	b.WriteString(mutedStyle.Render(helpView))

	return b.String()
}
