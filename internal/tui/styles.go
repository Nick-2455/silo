package tui

import "github.com/charmbracelet/lipgloss"

// All styles are plain text — no colors, no backgrounds, no borders.
// Layout is preserved for readability.

var (
	AppStyle = lipgloss.NewStyle().
		Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Width(80)

	StatusBarStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Width(80)

	// List styles.
	ListItemStyle = lipgloss.NewStyle().
		Padding(0, 1).
		MarginBottom(1)

	ListItemSelectedStyle = lipgloss.NewStyle().
		Padding(0, 1).
		MarginBottom(1)

	ListTitleStyle = lipgloss.NewStyle()

	ListURLStyle = lipgloss.NewStyle()

	ListBucketStyle = lipgloss.NewStyle()

	// Form styles.
	FormStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Width(60)

	FormLabelStyle = lipgloss.NewStyle().
		MarginTop(1)

	FormInputStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Width(56)

	FormInputFocusedStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Width(56)

	// Help text.
	HelpStyle = lipgloss.NewStyle().
		Padding(1, 2)

	// Status bar variants.
	StatusInfoStyle    = lipgloss.NewStyle()
	StatusSuccessStyle = lipgloss.NewStyle()
	StatusErrorStyle   = lipgloss.NewStyle()
	StatusWarnStyle    = lipgloss.NewStyle()

	// Degraded indicator.
	DegradedStyle = lipgloss.NewStyle()

	// Dashboard styles.
	DashBucketStyle = lipgloss.NewStyle().
		MarginTop(1)

	DashCountStyle = lipgloss.NewStyle()

	DashItemStyle = lipgloss.NewStyle().
		Padding(0, 2)

	// Triage styles.
	TriageOptionStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0, 1)

	TriageSelectedStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Margin(0, 1)

	// Config styles.
	ConfigKeyStyle = lipgloss.NewStyle().
		Width(20)

	ConfigValueStyle = lipgloss.NewStyle()

	// Navigation tabs.
	TabStyle = lipgloss.NewStyle().
		Padding(0, 2)

	TabActiveStyle = lipgloss.NewStyle().
		Padding(0, 2)
)

// DegradedIndicator returns the degraded indicator text.
func DegradedIndicator() string {
	return DegradedStyle.Render("[DEGRADED]")
}
