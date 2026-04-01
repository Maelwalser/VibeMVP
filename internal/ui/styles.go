package ui

import "github.com/charmbracelet/lipgloss"

// Tokyo Night color palette
const (
	clrBg       = "#1a1b26"
	clrBg2      = "#16161e"
	clrBgHL     = "#1e2030"
	clrFg       = "#c0caf5"
	clrFgDim    = "#545c7e"
	clrBlue     = "#7aa2f7"
	clrCyan     = "#7dcfff"
	clrGreen    = "#9ece6a"
	clrYellow   = "#e0af68"
	clrRed      = "#f7768e"
	clrMagenta  = "#bb9af7"
	clrComment  = "#565f89"
	clrSel      = "#283457"
	clrTabBg    = "#24283b"
)

var (
	StyleNormalMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBlue)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleInsertMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrGreen)).
		Foreground(lipgloss.Color(clrBg)).
		Bold(true).
		Padding(0, 1)

	StyleCommandMode = lipgloss.NewStyle().
		Background(lipgloss.Color(clrYellow)).
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
		Background(lipgloss.Color(clrBg)).
		Foreground(lipgloss.Color(clrFg)).
		Bold(true).
		Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
		Background(lipgloss.Color(clrTabBg)).
		Foreground(lipgloss.Color(clrComment)).
		Padding(0, 1)

	StyleTabBar = lipgloss.NewStyle().
		Background(lipgloss.Color(clrTabBg))

	StyleLineNum = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleCurLineNum = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrYellow)).
		Bold(true)

	StyleCurLine = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBgHL))

	StyleTilde = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment)).
		Bold(true)

	StyleFieldKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrCyan))

	StyleFieldKeyActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrBlue)).
		Bold(true)

	StyleEquals = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleFieldVal = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))

	StyleFieldValActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg)).
		Bold(true)

	StyleSelectArrow = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrYellow))

	StyleSectionTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrBlue)).
		Bold(true)

	StyleSectionDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment)).
		Italic(true)

	StyleHeaderBar = lipgloss.NewStyle().
		Background(lipgloss.Color(clrBg2)).
		Foreground(lipgloss.Color(clrFg))

	StyleHeaderTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrBlue)).
		Bold(true)

	StyleHeaderMod = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrRed)).
		Bold(true)

	StyleCmdLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrFg))

	StyleMsgOK = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrGreen))

	StyleMsgErr = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrRed))

	StyleHelpKey = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrYellow)).
		Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrComment))

	StyleTextAreaLabel = lipgloss.NewStyle().
		Foreground(lipgloss.Color(clrMagenta)).
		Bold(true)

	StyleTextAreaBorder = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clrBlue))

	StyleCursor = lipgloss.NewStyle().
		Background(lipgloss.Color(clrFg)).
		Foreground(lipgloss.Color(clrBg))

	StyleModalBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clrBlue)).
		Background(lipgloss.Color(clrBg2)).
		Padding(0, 1)
)
