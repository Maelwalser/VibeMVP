package ui

import "github.com/charmbracelet/lipgloss"

// Editorial Dark palette — stark monochromatic with warm amber accent.
// Inspired by avant-garde print design and professional terminal interfaces.
const (
	clrBg      = "#0C0C0C" // near-black base
	clrBg2     = "#131313" // panel background
	clrCrust   = "#060606" // deepest black — statusline / header
	clrBgHL    = "#1C1C1C" // selection highlight
	clrBgHL2   = "#252525" // pulse-frame selection
	clrFg      = "#E0E0E0" // primary text (near-white)
	clrFgDim   = "#525252" // secondary / structural text
	clrBlue    = "#7AACCF" // slate blue — field keys
	clrCyan    = "#78B4B4" // teal — current line num, accents
	clrTeal    = "#5A9690" // muted teal
	clrGreen   = "#6A9E6A" // success
	clrYellow  = "#C8A05A" // warm amber — PRIMARY ACTIVE ACCENT
	clrRed     = "#A86060" // error / warning
	clrMagenta = "#8A78A8" // muted violet — secondary
	clrPink    = "#A87888" // subtle rose
	clrComment = "#282828" // borders (very dark)
	clrSel     = "#1C1C1C" // selection (same as bgHL)
	clrTabBg   = "#080808" // tab bar background
	clrViolet  = "#7A6EA0" // muted violet
	clrOrange  = "#B08058" // warm secondary
)

// Sharp rectangular border — brutalist / editorial aesthetic.
var sharpBorder = lipgloss.Border{
	Top:         "─",
	Bottom:      "─",
	Left:        "│",
	Right:       "│",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}

// heavyRoundBorder kept for compat — mapped to sharp style.
var heavyRoundBorder = sharpBorder

// activeBorder — heavy weight for the active / focused pane.
var activeBorder = lipgloss.Border{
	Top:         "━",
	Bottom:      "━",
	Left:        "┃",
	Right:       "┃",
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
}

// panelBorderActive kept for compat.
var panelBorderActive = activeBorder

var (
	// Mode indicator pills — high-contrast badges.
	StyleNormalMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleInsertMode = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCyan)).
			Foreground(lipgloss.Color(clrBg)).
			Bold(true).
			Padding(0, 1)

	StyleCommandMode = lipgloss.NewStyle().
				Background(lipgloss.Color(clrMagenta)).
				Foreground(lipgloss.Color(clrBg)).
				Bold(true).
				Padding(0, 1)

	StyleStatusLine = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFg))

	StyleStatusRight = lipgloss.NewStyle().
				Background(lipgloss.Color(clrCrust)).
				Foreground(lipgloss.Color(clrFgDim))

	// Tab bar — active tab uses warm amber, inactive uses near-black.
	StyleTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
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

	// Line numbers — dim for inactive, amber for current.
	StyleLineNum = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	StyleCurLineNum = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleCurLine = lipgloss.NewStyle().
			Background(lipgloss.Color(clrBgHL))

	StyleCurLinePulse = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBgHL2))

	// Empty-line marker (tilde · equivalent).
	StyleTilde = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	// Form field keys — slate blue when inactive, amber when active.
	StyleFieldKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrBlue))

	StyleFieldKeyActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleEquals = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	StyleFieldVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	StyleFieldValActive = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFg)).
				Bold(true)

	StyleSelectArrow = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFgDim))

	// Section headers.
	StyleSectionTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleSectionDesc = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFgDim)).
				Italic(true)

	// Application header bar.
	StyleHeaderBar = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFg))

	StyleHeaderTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleHeaderMod = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).
			Bold(true)

	// Command line.
	StyleCmdLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFg))

	// Status messages.
	StyleMsgOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrGreen)).
			Bold(true)

	StyleMsgErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).
			Bold(true)

	// Hint bar keys/descriptions.
	StyleHelpKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrYellow)).
			Bold(true)

	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	// Text area styling.
	StyleTextAreaLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrYellow)).
				Bold(true)

	StyleTextAreaBorder = lipgloss.NewStyle().
				BorderStyle(sharpBorder).
				BorderForeground(lipgloss.Color(clrComment))

	StyleCursor = lipgloss.NewStyle().
			Background(lipgloss.Color(clrYellow)).
			Foreground(lipgloss.Color(clrBg))

	// Modal / panel borders.
	StyleModalBorder = lipgloss.NewStyle().
				Border(sharpBorder).
				BorderForeground(lipgloss.Color(clrComment)).
				Background(lipgloss.Color(clrBg2)).
				Padding(0, 1)

	StylePanelActive = lipgloss.NewStyle().
				Border(activeBorder).
				BorderForeground(lipgloss.Color(clrYellow)).
				Background(lipgloss.Color(clrBg))

	StylePanelInactive = lipgloss.NewStyle().
				Border(sharpBorder).
				BorderForeground(lipgloss.Color(clrComment)).
				Background(lipgloss.Color(clrBg2))

	StyleDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	// Accent styles — all use the warm amber palette.
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

	StyleNeonTeal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrTeal)).
			Bold(true)

	StyleHeaderDeco = lipgloss.NewStyle().
			Background(lipgloss.Color(clrCrust)).
			Foreground(lipgloss.Color(clrFgDim))

	// Powerline / statusline segment styles.
	StyleStatusSegmentMode = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrFgDim))

	StyleStatusSegmentPos = lipgloss.NewStyle().
				Background(lipgloss.Color(clrBg2)).
				Foreground(lipgloss.Color(clrBlue))

	// ASCII art side panel styles.
	// The art is rendered dim so it provides architectural visual texture
	// without overpowering the form content on the left.
	StyleArtPanel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#424242"))

	StyleArtPanelAccent = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#565656"))

	StyleArtSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#303030"))
)

// activeCurLineStyle returns the highlighted-row style (constant, no pulse animation).
func activeCurLineStyle() lipgloss.Style {
	return StyleCurLine
}
