package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-mvp/internal/manifest"
	"github.com/vibe-mvp/internal/realize/orchestrator"
)

// spinnerFrames is a braille dot animation.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ── message types ─────────────────────────────────────────────────────────────

type realizeLogMsg  string
type realizeDoneMsg struct{ err error }
type realizeTickMsg struct{}

// ── log entry ─────────────────────────────────────────────────────────────────

type logKind int

const (
	logInfo logKind = iota
	logStart
	logVerify
	logDone
	logError
	logSkip
	logWave
)

type logEntry struct {
	text string
	kind logKind
}

func classifyLog(text string) logKind {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "done (") || strings.Contains(lower, "complete"):
		return logDone
	case strings.Contains(lower, "starting:"):
		return logStart
	case strings.Contains(lower, "verify:"):
		return logVerify
	case strings.Contains(lower, "wave "):
		return logWave
	case strings.Contains(lower, "skip"):
		return logSkip
	case strings.Contains(lower, "error") || strings.Contains(lower, "warning") || strings.Contains(lower, "failed"):
		return logError
	default:
		return logInfo
	}
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleRealizeSpinner = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrBlue)).Bold(true)

	styleRealizeAppName = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrFg)).Bold(true)

	styleRealizeStatus = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrComment))

	styleLogStart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrCyan))

	styleLogDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrGreen))

	styleLogVerify = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrYellow))

	styleLogError = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed))

	styleLogSkip = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrComment))

	styleLogWave = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrMagenta))

	styleLogInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrFgDim))

	styleRealizeDone = lipgloss.NewStyle().
				Foreground(lipgloss.Color(clrGreen)).Bold(true)

	styleRealizeErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(clrRed)).Bold(true)
)

// ── RealizationScreen ─────────────────────────────────────────────────────────

// RealizationScreen is shown in the content area while the realize agent runs.
type RealizationScreen struct {
	appName   string
	logs      []logEntry
	done      bool
	err       error
	frame     int
	logCh     chan string
	cancelFn  context.CancelFunc
	wantsQuit bool
}

func newRealizationScreen() RealizationScreen {
	return RealizationScreen{}
}

// Start initializes the screen and launches the orchestrator in a goroutine.
// manifestPath is the path to the saved manifest.json.
func (s RealizationScreen) Start(manifestPath string, mf *manifest.Manifest) (RealizationScreen, tea.Cmd) {
	logCh := make(chan string, 512)
	ctx, cancel := context.WithCancel(context.Background())

	s.logCh     = logCh
	s.cancelFn  = cancel
	s.appName   = mf.Realize.AppName
	s.done      = false
	s.err       = nil
	s.logs      = nil
	s.frame     = 0
	s.wantsQuit = false

	opts := mf.Realize

	runCmd := func() tea.Msg {
		cfg := orchestrator.Config{
			ManifestPath: manifestPath,
			OutputDir:    opts.OutputDir,
			SkillsDir:    ".vibemvp/skills",
			MaxRetries:   3,
			Parallelism:  opts.Concurrency,
			DryRun:       opts.DryRun,
			Verbose:      false,
			LogFunc: func(line string) {
				select {
				case logCh <- line:
				default: // drop if buffer is full
				}
			},
		}
		err := orchestrator.New(cfg).Run(ctx)
		close(logCh)
		return realizeDoneMsg{err: err}
	}

	tickCmd := tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return realizeTickMsg{}
	})

	return s, tea.Batch(runCmd, tickCmd, s.waitForLog())
}

func (s RealizationScreen) waitForLog() tea.Cmd {
	if s.logCh == nil {
		return nil
	}
	return func() tea.Msg {
		line, ok := <-s.logCh
		if !ok {
			return nil // channel closed; realizeDoneMsg arrives separately
		}
		return realizeLogMsg(line)
	}
}

// WantsQuit reports whether the user pressed q after completion.
func (s RealizationScreen) WantsQuit() bool { return s.wantsQuit }

// ── Update ────────────────────────────────────────────────────────────────────

func (s RealizationScreen) Update(msg tea.Msg) (RealizationScreen, tea.Cmd) {
	switch m := msg.(type) {

	case realizeTickMsg:
		s.frame = (s.frame + 1) % len(spinnerFrames)
		if !s.done {
			return s, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return realizeTickMsg{}
			})
		}

	case realizeLogMsg:
		text := string(m)
		s.logs = append(s.logs, logEntry{text: text, kind: classifyLog(text)})
		return s, s.waitForLog()

	case realizeDoneMsg:
		s.done = true
		s.err = m.err
		if m.err != nil {
			s.logs = append(s.logs, logEntry{
				text: fmt.Sprintf("error: %v", m.err),
				kind: logError,
			})
		} else {
			s.logs = append(s.logs, logEntry{
				text: "realization complete",
				kind: logDone,
			})
		}

	case tea.KeyMsg:
		switch m.String() {
		case "q":
			if s.done {
				s.wantsQuit = true
			}
		case "esc":
			if !s.done && s.cancelFn != nil {
				s.cancelFn()
			}
		}
	}

	return s, nil
}

// Mode satisfies the sub-editor interface — always NORMAL during realization.
func (s RealizationScreen) Mode() Mode { return ModeNormal }

// HintLine returns the bottom hint bar text.
func (s RealizationScreen) HintLine() string {
	if s.done {
		return hintBar("q", "quit")
	}
	return hintBar("Esc", "cancel")
}

// ── View ──────────────────────────────────────────────────────────────────────

func (s RealizationScreen) View(w, h int) string {
	// Status line: spinner + app name + state
	var statusLine string
	if s.done {
		if s.err != nil {
			statusLine = styleRealizeErr.Render("✗") + "  " +
				styleRealizeAppName.Render(s.appName) + "  " +
				styleRealizeStatus.Render("failed")
		} else {
			statusLine = styleRealizeDone.Render("✓") + "  " +
				styleRealizeAppName.Render(s.appName) + "  " +
				styleRealizeStatus.Render("complete")
		}
	} else {
		spin := styleRealizeSpinner.Render(spinnerFrames[s.frame])
		statusLine = spin + "  " +
			styleRealizeAppName.Render(s.appName) + "  " +
			styleRealizeStatus.Render("realizing…")
	}

	lines := []string{
		"  " + statusLine,
		"",
	}

	// Log area: fill remaining height with the most recent entries.
	// Reserve 2 lines for the header above.
	logH := h - 2
	if logH < 0 {
		logH = 0
	}

	start := 0
	if len(s.logs) > logH {
		start = len(s.logs) - logH
	}

	maxTextW := w - 5 // 4 chars line-number + 1 space
	if maxTextW < 10 {
		maxTextW = 10
	}

	for i, entry := range s.logs[start:] {
		lineNo := start + i + 1
		num := StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))

		t := entry.text
		if len(t) > maxTextW {
			t = t[:maxTextW-1] + "…"
		}

		var colored string
		switch entry.kind {
		case logStart:
			colored = styleLogStart.Render(t)
		case logDone:
			colored = styleLogDone.Render(t)
		case logVerify:
			colored = styleLogVerify.Render(t)
		case logError:
			colored = styleLogError.Render(t)
		case logSkip:
			colored = styleLogSkip.Render(t)
		case logWave:
			colored = styleLogWave.Render(t)
		default:
			colored = styleLogInfo.Render(t)
		}

		lines = append(lines, num+colored)
	}

	return fillTildes(lines, h)
}
