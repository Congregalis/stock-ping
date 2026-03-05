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
	var totalValue, totalCost, totalRealized, totalUnrealized float64
	for _, symbol := range m.stockOrder {
		data := m.stocks[symbol]
		if data == nil || data.Quantity <= 0 {
			continue
		}

		if data.Price > 0 {
			totalValue += data.Price * data.Quantity
			totalUnrealized += (data.Price - data.CostPrice) * data.Quantity
		}

		totalCost += data.CostPrice * data.Quantity
		totalRealized += data.RealizedPnL
	}
	totalPL := totalRealized + totalUnrealized

	// Summary Cards
	if m.holdingsCount > 0 {
		visValue := fmt.Sprintf("$%.2f", totalValue)
		visCost := fmt.Sprintf("$%.2f", totalCost)
		visRealized := fmt.Sprintf("$%.2f", absFloat(totalRealized))
		visUnrealized := fmt.Sprintf("$%.2f", absFloat(totalUnrealized))

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
			visRealized = "****"
			visUnrealized = "****"
			visPL = "****"
		}

		plPercent := 0.0
		if totalCost > 0 {
			plPercent = (totalPL / totalCost) * 100
		}
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

		realizedStyle := greenStyle
		realizedContent := fmt.Sprintf("+%s", visRealized)
		if totalRealized < 0 {
			realizedStyle = redStyle
			realizedContent = fmt.Sprintf("-%s", visRealized)
		}

		unrealizedStyle := greenStyle
		unrealizedContent := fmt.Sprintf("+%s", visUnrealized)
		if totalUnrealized < 0 {
			unrealizedStyle = redStyle
			unrealizedContent = fmt.Sprintf("-%s", visUnrealized)
		}

		realizedCard := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				summaryLabelStyle.Render("REALIZED P/L"),
				realizedStyle.Render(realizedContent),
			),
		)

		unrealizedCard := cardStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				summaryLabelStyle.Render("UNREALIZED P/L"),
				unrealizedStyle.Render(unrealizedContent),
			),
		)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, valueCard, costCard, realizedCard, unrealizedCard, plCard))
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
	b.WriteString("\n")

	if m.positionAction != positionActionNone {
		b.WriteString("\n")
		b.WriteString(cardStyle.Render(m.renderPositionActionView()))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Help with updated keys hint
	helpView := m.help.View(m.keys)
	b.WriteString(mutedStyle.Render(helpView))

	return b.String()
}

func (m Model) renderPositionActionView() string {
	if m.positionAction == positionActionDelete {
		return lipgloss.JoinVertical(lipgloss.Left,
			summaryLabelStyle.Render(m.positionActionTitle()),
			"Press Enter to confirm delete",
			"Press Esc to cancel",
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		summaryLabelStyle.Render(m.positionActionTitle()),
		summaryLabelStyle.Render(m.positionActionPrompt()),
		m.positionInput.View(),
		mutedStyle.Render("Enter: confirm • Esc: cancel"),
	)
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
