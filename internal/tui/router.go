package tui

// Screen identifies which screen is active.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenList
	ScreenAdd
	ScreenTriage
	ScreenConfig
)

var screenNames = map[Screen]string{
	ScreenDashboard: "Dashboard",
	ScreenList:      "List",
	ScreenAdd:       "Add",
	ScreenTriage:    "Triage",
	ScreenConfig:    "Config",
}

func (s Screen) String() string {
	return screenNames[s]
}

// Route defines valid forward/backward navigation for a screen.
type Route struct {
	Forward  Screen
	Backward Screen
}

// routes maps each screen to its valid navigation targets.
var routes = map[Screen]Route{
	ScreenDashboard: {},
	ScreenList:      {Backward: ScreenDashboard},
	ScreenAdd:       {Backward: ScreenDashboard},
	ScreenTriage:    {Backward: ScreenDashboard},
	ScreenConfig:    {Backward: ScreenDashboard},
}

// setScreen navigates to the given screen, recording the current screen as previous.
func (m *Model) setScreen(s Screen) {
	m.Previous = m.Screen
	m.Screen = s
	m.Cursor = 0
}

// goBack navigates to the previous screen using the routes map.
// Returns false if no backward route exists.
func (m *Model) goBack() bool {
	route, ok := routes[m.Screen]
	if !ok || route.Backward == ScreenDashboard && m.Screen == ScreenDashboard {
		return false
	}
	if ok && route.Backward != 0 {
		m.Screen = route.Backward
		m.Cursor = 0
		return true
	}
	// Fallback: use stored previous
	if m.Previous != m.Screen && m.Previous != ScreenTriage && m.Previous != ScreenAdd {
		m.Screen = m.Previous
		m.Cursor = 0
		return true
	}
	m.Screen = ScreenDashboard
	m.Cursor = 0
	return true
}
