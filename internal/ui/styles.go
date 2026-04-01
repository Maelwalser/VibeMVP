package ui

import "github.com/charmbracelet/lipgloss"

// Cyberpunk / neon palette — dark void with electric accents
const (
	clrBg      = "#0a0a0f" // void black
	clrBg2     = "#05050a" // deeper void
	clrBgHL    = "#12102a" // active selection — deep violet
	clrBgHL2   = "#1a0f35" // pulse-frame active selection
	clrFg      = "#e2f0f1" // near-white with slight cyan tint
	clrFgDim   = "#3a3f5a" // dim purple-gray
	clrBlue    = "#00b4ff" // electric blue
	clrCyan    = "#05d9e8" // neon cyan
	clrGreen   = "#05ffa1" // acid green
	clrYellow  = "#f5c542" // neon yellow
	clrRed     = "#ff2055" // hot red
	clrMagenta = "#ff2a6d" // neon magenta/pink
	clrComment = "#363b54" // dim purple-gray
	clrSel     = "#1a0a2e" // deep violet selection bg
	clrTabBg   = "#0d0d1a" // tab background
	clrViolet  = "#9b59ff" // electric violet
	clrOrange  = "#ff6e27" // neon orange
)

var (
	StyleNormalMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrCyan)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleInsertMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrGreen)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleCommandMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrMagenta)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleStatusLine = lipgloss.NewStyle().
		Background(lipgloss.Color(clrSel)).
		Foreground(lipgloss.Color(clrFg))

	StyleStatusRight = lipgloss.NewStyle().
		Background(lipgloss.Color(clrSel)).
		Foreground(lipgloss.Color(clrComment))

	StyleTabActive = lipgloss.NewStyle().
		Background(lipgloss.Color(clrViolet)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
		Background(lipgloss.Color(clrTabBg)).
		Foreground(lipgloss.Color(clrFgDim)).
		Padding(0, 1)

	StyleTabSep = lipgloss.NewStyle().
		Background(lipgloss.Color(clrTabBg)).
		Foreground(lipgloss.Color(clrComment))

	StyleTabBar = lipgloss.NewStyle().
		Background(lipgloss.Color(clrTabBg))

	StyleLineNum = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleCurLineNum = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleCurLine = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))

	StyleCurLinePulse = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL2))

	StyleTilde = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleFieldKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrBlue))

	StyleFieldKeyActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleEquals = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleFieldVal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))

	StyleFieldValActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Bold(true)

	StyleSelectArrow = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrViolet))

	StyleSectionTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleSectionDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment)).
		Italic(true)

	StyleHeaderBar = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBg2)).
		Foreground(lipgloss.Color(clrFg))

	StyleHeaderTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleHeaderMod = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrMagenta)).
		Bold(true)

	StyleCmdLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))

	StyleMsgOK = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrGreen)).
		Bold(true)

	StyleMsgErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrRed)).
		Bold(true)

	StyleHelpKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleTextAreaLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrViolet)).
		Bold(true)

	StyleTextAreaBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clrViolet))

	StyleCursor = lipgloss.NewStyle().
		Background(lipgloss.Color(clrCyan)).
		Foreground(lipgloss.Color(clrBg))

	StyleModalBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clrViolet)).
		Background(lipgloss.Color(clrBg2)).
		Padding(0, 1)

	// Cyberpunk accent styles used in headers and decorations
	StyleNeonMagenta = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrMagenta)).
		Bold(true)

	StyleNeonCyan = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan)).
		Bold(true)

	StyleNeonGreen = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrGreen)).
		Bold(true)

	StyleNeonViolet = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrViolet)).
		Bold(true)

	StyleNeonOrange = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrOrange)).
		Bold(true)

	StyleHeaderDeco = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBg2)).
		Foreground(lipgloss.Color(clrFgDim))
)

// activeCurLineStyle returns the appropriate highlighted-row style based on the
// current animation frame, producing a subtle breathing/pulse effect.
func activeCurLineStyle() lipgloss.Style {
	if AnimFrame == 1 {
		return StyleCurLinePulse
	}
	return StyleCurLine
}
