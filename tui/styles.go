package tui

import "github.com/charmbracelet/lipgloss"

// Color Palette (Catppuccin Mocha inspired)
const (
	ColorRosewater = "#f5e0dc"
	ColorFlamingo  = "#f2cdcd"
	ColorPink      = "#f5c2e7"
	ColorMauve     = "#cba6f7"
	ColorRed       = "#f38ba8"
	ColorMaroon    = "#eba0ac"
	ColorPeach     = "#fab387"
	ColorYellow    = "#f9e2af"
	ColorGreen     = "#a6e3a1"
	ColorTeal      = "#94e2d5"
	ColorSky       = "#89dceb"
	ColorSapphire  = "#74c7ec"
	ColorBlue      = "#89b4fa"
	ColorLavender  = "#b4befe"
	ColorText      = "#cdd6f4"
	ColorSubtext1  = "#bac2de"
	ColorSubtext0  = "#a6adc8"
	ColorOverlay2  = "#9399b2"
	ColorOverlay1  = "#7f849c"
	ColorOverlay0  = "#6c7086"
	ColorSurface2  = "#585b70"
	ColorSurface1  = "#45475a"
	ColorSurface0  = "#313244"
	ColorBase      = "#1e1e2e"
	ColorMantle    = "#181825"
	ColorCrust     = "#11111b"
)

// Styles
var (
	// App Title
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBase)).
			Background(lipgloss.Color(ColorMauve)).
			Bold(true).
			Padding(0, 1).
			MarginLeft(1)

	// Section Titles
	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorLavender)).
				Bold(true).
				MarginBottom(1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSubtext0))

	// Stock change colors
	greenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorRed))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOverlay0))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOverlay1))

	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBlue)).
			Bold(true)

	// Table Styles
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorLavender)).
				Bold(true).
				Align(lipgloss.Center)

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorText)).
				Background(lipgloss.Color(ColorSurface1)).
				Bold(true)

	// Card / Summary Styles
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorSurface2)).
			Padding(0, 1).
			MarginRight(1)

	summaryValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorText)).
				Bold(true)

	summaryLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorSubtext0))
)

const splashLogo = `
   ▄█████ ▄▄▄▄▄▄ ▄▄▄   ▄▄▄▄ ▄▄ ▄▄   █████▄ ▄▄ ▄▄  ▄▄  ▄▄▄▄ 
   ▀▀▀▄▄▄   ██  ██▀██ ██▀▀▀ ██▄█▀   ██▄▄█▀ ██ ███▄██ ██ ▄▄ 
   █████▀   ██  ▀███▀ ▀████ ██ ██   ██     ██ ██ ▀██ ▀███▀ 
`
