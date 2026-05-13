package tui

import "github.com/charmbracelet/lipgloss"

// Theme colors.
const (
	ColorBg       = "#1a1b26"
	ColorSurface  = "#24283b"
	ColorBorder   = "#414868"
	ColorText     = "#c0caf5"
	ColorMuted    = "#565f89"
	ColorPrimary  = "#7aa2f7"
	ColorSuccess  = "#9ece6a"
	ColorWarning  = "#e0af68"
	ColorError    = "#f7768e"
	ColorSelected = "#bb9af7"
)

var (
	// App styles.
	AppStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(ColorBg)).
		Foreground(lipgloss.Color(ColorText)).
		Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(0, 1).
			Width(80)

	StatusBarStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(0, 1).
			Width(80)

	// List styles.
	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginBottom(1)

	ListItemSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorSurface)).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(0, 1).
				MarginBottom(1)

	ListTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	ListURLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Italic(true)

	ListBucketStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true)

	// Form styles.
	FormStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Width(60)

	FormLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			MarginTop(1)

	FormInputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorBorder)).
			Padding(0, 1).
			Width(56)

	FormInputFocusedStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(0, 1).
				Width(56)

	// Help text.
	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Padding(1, 2)

	// Status bar variants.
	StatusInfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorText))
	StatusSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	StatusErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	StatusWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))

	// Degraded indicator.
	DegradedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Bold(true)

	// Dashboard styles.
	DashBucketStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			MarginTop(1)

	DashCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true)

	DashItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	// Triage styles.
	TriageOptionStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Margin(0, 1)

	TriageSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color(ColorSurface)).
				Border(lipgloss.ThickBorder(), true).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(0, 1).
				Margin(0, 1)

	// Config styles.
	ConfigKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			Width(20)

	ConfigValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorText))

	// Navigation tabs.
	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color(ColorMuted))

	TabActiveStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			Underline(true)
)

// DegradedIndicator returns the styled degraded indicator text.
func DegradedIndicator() string {
	return DegradedStyle.Render("⚠ DEGRADED")
}
