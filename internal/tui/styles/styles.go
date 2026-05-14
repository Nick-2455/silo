package styles

import "github.com/charmbracelet/lipgloss"

// Silo warm earth-tone palette.
var (
	ColorBase     = lipgloss.Color("#1e1e1e")
	ColorSurface  = lipgloss.Color("#2a2a2a")
	ColorOverlay  = lipgloss.Color("#6b6b6b")
	ColorText     = lipgloss.Color("#e8e6e3")
	ColorSubtext  = lipgloss.Color("#9a9a9a")
	ColorAccent   = lipgloss.Color("#d4a574") // warm amber
	ColorGreen    = lipgloss.Color("#8fbc8f") // sage
	ColorPeach    = lipgloss.Color("#d4a574") // amber
	ColorRed      = lipgloss.Color("#cd5c5c") // indian red
	ColorBlue     = lipgloss.Color("#6b8e9b") // steel blue
	ColorMauve    = lipgloss.Color("#c4a882") // warm tan
	ColorYellow   = lipgloss.Color("#d4c5a9") // sand
	ColorTeal     = lipgloss.Color("#7fb3b3") // muted teal
)

// Cursor is the prefix used for the currently focused item.
const Cursor = "▸ "

// Pre-built reusable styles.
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	HeadingStyle = lipgloss.NewStyle().
			Foreground(ColorMauve).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext)

	SubtextStyle = lipgloss.NewStyle().
			Foreground(ColorSubtext)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	UnselectedStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	FrameStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorAccent).
			Padding(1, 2)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorOverlay).
			Padding(0, 1)
)
