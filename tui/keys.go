package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines keybindings
type keyMap struct {
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
	Sort    key.Binding

	// Navigation
	Select    key.Binding // Enter
	Back      key.Binding // Esc, Backspace
	Portfolio key.Binding // p - switch to portfolio view
	Dashboard key.Binding // d - switch to dashboard view
	Privacy   key.Binding // P - toggle privacy/share mode
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Refresh, k.Sort, k.Portfolio, k.Dashboard, k.Privacy, k.Select, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Refresh, k.Sort, k.Portfolio, k.Dashboard, k.Privacy, k.Select, k.Back, k.Quit, k.Help}}
}

var defaultKeyMap = keyMap{
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("h", "help"),
		key.WithHelp("h", "help"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort order"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "details"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Portfolio: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "portfolio"),
	),
	Dashboard: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "dashboard"),
	),
	Privacy: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "privacy mode"),
	),
}
