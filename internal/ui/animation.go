package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// AnimFrame is the current global animation frame index (0 or 1).
// It is toggled by the root model's ticker and read by render helpers
// to produce pulse / breathing effects on the active selection.
var AnimFrame int

// uiTickMsg is sent on each animation tick.
type uiTickMsg struct{}

// uiTick returns a command that sends uiTickMsg after the animation interval.
func uiTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return uiTickMsg{}
	})
}

// headerDecoFrames are the two-frame scanline decorations shown in the header bar.
// They alternate on each tick to create a subtle scanning animation.
var headerDecoFrames = [2]string{"░▒▓", "▓▒░"}

// modeSpinFrames are the two-frame decorators flanking the mode badge.
var modeSpinFrames = [2][2]string{
	{"◀", "▶"},
	{"◁", "▷"},
}
