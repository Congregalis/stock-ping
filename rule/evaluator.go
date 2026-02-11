package rule

import (
	"fmt"

	"github.com/congregalis/stock-ping/config"
	"github.com/congregalis/stock-ping/stock"
)

// TriggerResult represents the result of a rule evaluation
type TriggerResult struct {
	Rule    *config.Rule
	Quote   *stock.Quote
	Reasons []string // List of trigger reasons
}

// Triggered returns true if any conditions were triggered
func (t *TriggerResult) Triggered() bool {
	return len(t.Reasons) > 0
}

// FormatNotification returns a formatted notification message
func (t *TriggerResult) FormatNotification() (title, body string) {
	displayName := t.Rule.Symbol
	if t.Rule.Name != "" {
		displayName = fmt.Sprintf("%s (%s)", t.Rule.Symbol, t.Rule.Name)
	}

	title = fmt.Sprintf("📊 %s 触发提醒", displayName)

	changeSign := ""
	if t.Quote.PercentChange >= 0 {
		changeSign = "+"
	}

	body = fmt.Sprintf("价格: $%.2f (%s%.2f%%)\n",
		t.Quote.CurrentPrice, changeSign, t.Quote.PercentChange)

	for _, reason := range t.Reasons {
		body += fmt.Sprintf("⚠️ %s\n", reason)
	}

	return title, body
}

// Evaluator evaluates monitoring rules against stock quotes
type Evaluator struct{}

// NewEvaluator creates a new rule evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate checks if a quote triggers any conditions in the rule
func (e *Evaluator) Evaluate(rule *config.Rule, quote *stock.Quote) *TriggerResult {
	result := &TriggerResult{
		Rule:    rule,
		Quote:   quote,
		Reasons: []string{},
	}

	// Check price above threshold
	if rule.PriceAbove != nil && quote.CurrentPrice > *rule.PriceAbove {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("价格 $%.2f 超过 $%.2f", quote.CurrentPrice, *rule.PriceAbove))
	}

	// Check price below threshold
	if rule.PriceBelow != nil && quote.CurrentPrice < *rule.PriceBelow {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("价格 $%.2f 低于 $%.2f", quote.CurrentPrice, *rule.PriceBelow))
	}

	// Check percent change above threshold (positive)
	if rule.ChangeAbove != nil && quote.PercentChange > *rule.ChangeAbove {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("涨幅 %.2f%% 超过 %.2f%%", quote.PercentChange, *rule.ChangeAbove))
	}

	// Check percent change below threshold (negative)
	if rule.ChangeBelow != nil && quote.PercentChange < *rule.ChangeBelow {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("跌幅 %.2f%% 超过 %.2f%%", quote.PercentChange, *rule.ChangeBelow))
	}

	return result
}
