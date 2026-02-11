package tui

import (
	"fmt"
	"strings"
)

// ViewDashboard renders the main dashboard view
func (m Model) ViewDashboard() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("📊 Stock Ping")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Market status
	b.WriteString("\n\n")

	// Table
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Status bar
	statusParts := []string{
		fmt.Sprintf("Interval: %ds", m.cfg.Interval),
		fmt.Sprintf("Stocks: %d", len(m.stocks)),
	}
	if !m.lastRefresh.IsZero() {
		statusParts = append(statusParts, fmt.Sprintf("Last: %s", m.lastRefresh.Format("15:04:05")))
	}
	if m.statusMessage != "" {
		statusParts = append(statusParts, m.statusMessage)
	}
	b.WriteString(statusBarStyle.Render(strings.Join(statusParts, " • ")))
	b.WriteString("\n\n")

	// Help (Contextual)
	b.WriteString(mutedStyle.Render(m.help.View(m.keys)))

	return b.String()
}
