package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/guptarohit/asciigraph"
)

// ViewTrend renders the historical trend view
func (m Model) ViewTrend() string {
	var b strings.Builder

	// Title / Header
	title := titleStyle.Render(fmt.Sprintf("📈 Trend: %s", m.selectedSymbol))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Content
	if m.trendLoading {
		b.WriteString("Loading trend data...\n")
	} else if m.trendError != nil {
		b.WriteString(redStyle.Render(fmt.Sprintf("Error loading data: %v", m.trendError)))
		b.WriteString("\n\n")
	} else if m.trendData != nil {
		// Display basic info
		data, ok := m.stocks[m.selectedSymbol]
		if ok {
			info := fmt.Sprintf("Price: $%.2f • Change: %.2f%%", data.Price, data.Change)
			if data.Change >= 0 {
				b.WriteString(greenStyle.Render(info))
			} else {
				b.WriteString(redStyle.Render(info))
			}
			b.WriteString("\n\n")
		}

		// Render Chart using asciigraph
		// Auto-size height based on window, but keep some bounds
		chartHeight := m.height - 12
		if chartHeight < 10 {
			chartHeight = 10
		}

		width := m.width - 15
		if width < 10 {
			width = 10
		}

		// Configure chart
		graph := asciigraph.Plot(
			m.trendData.C,
			asciigraph.Height(chartHeight),
			asciigraph.Width(width),
			asciigraph.Precision(2),
			asciigraph.SeriesColors(
				asciigraph.Blue,
			),
		)

		b.WriteString(graph)
		b.WriteString("\n")

		// Add X-Axis Labels (manual)
		if len(m.trendData.T) > 0 {
			firstTime := time.Unix(m.trendData.T[0], 0)
			lastTime := time.Unix(m.trendData.T[len(m.trendData.T)-1], 0)

			startLabel := firstTime.Format("2006-01-02")
			endLabel := lastTime.Format("2006-01-02")

			// We need to account for the Y-axis label width that asciigraph adds.
			// It's dynamic, but usually around 8-9 chars.
			// Ideally we'd calculate it, but fixed padding is a "good enough" approximation for now.
			paddingLeft := 9

			// Ensure we don't overflow
			availableSpace := width
			if len(startLabel)+len(endLabel) < availableSpace {
				// Create a spacer string
				spaceWidth := availableSpace - len(startLabel) - len(endLabel)
				spacer := strings.Repeat(" ", spaceWidth)

				// Apply padding to shift the whole line to the right to clear Y-axis labels
				axisPadding := strings.Repeat(" ", paddingLeft)

				b.WriteString(mutedStyle.Render(axisPadding + startLabel + spacer + endLabel))
			}
		}
		b.WriteString("\n\n")
	}

	// Footer / Help
	b.WriteString(mutedStyle.Render("Press [Esc] to return"))

	return b.String()
}
