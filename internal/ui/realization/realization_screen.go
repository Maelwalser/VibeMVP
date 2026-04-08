package realization

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vibe-menu/internal/manifest"
	"github.com/vibe-menu/internal/realize/orchestrator"
	"github.com/vibe-menu/internal/ui/core"
)

// spinnerFrames is a matrix-style 12-frame braille spinner.
var spinnerFrames = core.MatrixSpinnerFrames[:]

// ── message types ─────────────────────────────────────────────────────────────

type realizeLogMsg string
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
				Foreground(lipgloss.Color(core.ClrYellow)).Bold(true)

	styleRealizeAppName = lipgloss.NewStyle().
				Foreground(lipgloss.Color(core.ClrFg)).Bold(true)

	styleRealizeStatus = lipgloss.NewStyle().
				Foreground(lipgloss.Color(core.ClrComment))

	styleProgressFill  = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrYellow))
	styleProgressEmpty = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFgDim))
	styleProgressLabel = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrFg))
	styleProgressDone  = lipgloss.NewStyle().Foreground(lipgloss.Color(core.ClrGreen))

	styleLogStart = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrCyan))

	styleLogDone = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrGreen))

	styleLogVerify = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrYellow))

	styleLogError = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrRed))

	styleLogSkip = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrComment))

	styleLogWave = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrMagenta))

	styleLogInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrFgDim))

	styleRealizeDone = lipgloss.NewStyle().
				Foreground(lipgloss.Color(core.ClrGreen)).Bold(true)

	styleRealizeErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(core.ClrRed)).Bold(true)
)

// ── Screen ───────────────────────────────────────────────────────────────────

// Screen is shown in the content area while the realize agent runs.
type Screen struct {
	appName    string
	logs       []logEntry
	done       bool
	err        error
	frame      int
	logCh      chan string
	cancelFn   context.CancelFunc
	wantsQuit  bool
	totalTasks int
	doneTasks  int
	startedAt  time.Time
	elapsed    time.Duration
}

func NewScreen() Screen {
	return Screen{}
}

// Start initializes the screen and launches the orchestrator in a goroutine.
// manifestPath is the path to the saved manifest.json.
func (s Screen) Start(manifestPath string, mf *manifest.Manifest) (Screen, tea.Cmd) {
	logCh := make(chan string, 512)
	ctx, cancel := context.WithCancel(context.Background())

	s.logCh = logCh
	s.cancelFn = cancel
	s.appName = mf.Realize.AppName
	s.startedAt = time.Now()
	s.done = false
	s.err = nil
	s.logs = nil
	s.frame = 0
	s.wantsQuit = false

	opts := mf.Realize

	runCmd := func() tea.Msg {
		cfg := orchestrator.Config{
			ManifestPath: manifestPath,
			OutputDir:    opts.OutputDir,
			SkillsDir:    ".vibemenu/skills",
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

func (s Screen) waitForLog() tea.Cmd {
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
func (s Screen) WantsQuit() bool { return s.wantsQuit }

// ── Update ────────────────────────────────────────────────────────────────────

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch m := msg.(type) {

	case realizeTickMsg:
		s.frame = (s.frame + 1) % len(spinnerFrames)
		if !s.done {
			s.elapsed = time.Since(s.startedAt)
			return s, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return realizeTickMsg{}
			})
		}

	case realizeLogMsg:
		text := string(m)
		s.logs = append(s.logs, logEntry{text: text, kind: classifyLog(text)})
		s.trackProgress(text)
		return s, s.waitForLog()

	case realizeDoneMsg:
		s.done = true
		s.elapsed = time.Since(s.startedAt)
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
func (s Screen) Mode() core.Mode { return core.ModeNormal }

// HintLine returns the bottom hint bar text.
func (s Screen) HintLine() string {
	if s.done {
		return core.HintBar("q", "quit")
	}
	return core.HintBar("Esc", "cancel")
}

// trackProgress extracts task count and completion events from log lines.
func (s *Screen) trackProgress(text string) {
	// "realize: starting N tasks across M wave(s)" → total
	if s.totalTasks == 0 && strings.HasPrefix(text, "realize: starting ") {
		var n int
		if _, err := fmt.Sscanf(text, "realize: starting %d", &n); err == nil && n > 0 {
			s.totalTasks = n
		}
		return
	}
	// "[taskID] done (N files)" or "[taskID] skipping (already completed)"
	if strings.Contains(text, "] done (") ||
		(strings.Contains(text, "] skipping") && strings.Contains(text, "already completed")) {
		s.doneTasks++
	}
}

// renderProgressBar renders a filled/empty block progress bar with a task counter.
// When not complete, a sine-wave animation is shown at the leading edge of the
// filled portion, giving the impression that progress is flowing into the bar.
func renderProgressBar(done, total, w, frame int) string {
	label := fmt.Sprintf(" %d / %d tasks", done, total)
	barW := w - len([]rune(label)) - 6 // 2 padding + "[" + "]" + 1 space
	if barW < 4 {
		barW = 4
	}

	filled := 0
	if total > 0 {
		filled = done * barW / total
	}
	if filled > barW {
		filled = barW
	}

	complete := done >= total
	fillStyle := styleProgressFill
	if complete {
		fillStyle = styleProgressDone
	}

	var bar string
	if complete {
		bar = fillStyle.Render(strings.Repeat("█", barW))
	} else {
		wave := core.SineWaveFrames[frame%16]
		waveW := len([]rune(wave)) // 8 chars
		// Ensure the wave fits within the empty region.
		emptyW := barW - filled
		if waveW > emptyW {
			waveW = emptyW
			wave = string([]rune(wave)[:waveW])
		}
		bar = fillStyle.Render(strings.Repeat("█", filled)) +
			styleProgressFill.Render(wave) +
			styleProgressEmpty.Render(strings.Repeat("░", emptyW-waveW))
	}

	return "  " +
		styleProgressEmpty.Render("[") +
		bar +
		styleProgressEmpty.Render("]") +
		styleProgressLabel.Render(label)
}

// formatElapsed returns a human-readable mm:ss or hh:mm:ss string.
func formatElapsed(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// ── View ──────────────────────────────────────────────────────────────────────

// realizeHeaderBar renders a single-line bordered title bar for the realize screen.
func realizeHeaderBar(appName string, done bool, err error, w int) string {
	var stateTag string
	switch {
	case done && err != nil:
		stateTag = styleRealizeErr.Render(" ✗ FAILED ")
	case done:
		stateTag = styleRealizeDone.Render(" ✓ DONE ")
	default:
		stateTag = styleRealizeSpinner.Render(" ▶ REALIZE ")
	}

	appTag := styleRealizeAppName.Render(" " + appName + " ")

	innerW := w - 2 // subtract left + right border char
	if innerW < 10 {
		innerW = 10
	}

	stateW := len([]rune(stateTag)) // approximate; lipgloss.Width is accurate
	appW := len([]rune(appTag))
	_ = stateW
	_ = appW

	stateRendW := lipgloss.Width(stateTag)
	appRendW := lipgloss.Width(appTag)
	dashCount := innerW - stateRendW - appRendW - 2
	if dashCount < 1 {
		dashCount = 1
	}

	dashes := styleRealizeStatus.Render(strings.Repeat("─", dashCount))
	bar := styleRealizeStatus.Render("╭") +
		stateTag +
		dashes +
		appTag +
		styleRealizeStatus.Render("╮")
	return bar
}

// logPrefix returns a short colored prefix and icon for the given log kind.
func logPrefix(k logKind) string {
	switch k {
	case logStart:
		return styleLogStart.Render("›")
	case logDone:
		return styleLogDone.Render("✓")
	case logVerify:
		return styleLogVerify.Render("⚑")
	case logError:
		return styleLogError.Render("✗")
	case logSkip:
		return styleLogSkip.Render("∅")
	case logWave:
		return styleLogWave.Render("≋")
	default:
		return styleLogInfo.Render("·")
	}
}

func (s Screen) View(w, h int) string {
	lines := make([]string, 0, h)

	// Row 0: decorated header bar.
	lines = append(lines, realizeHeaderBar(s.appName, s.done, s.err, w))

	// Row 1: spinner + app name + state — the live status row.
	timer := styleRealizeStatus.Render("  " + formatElapsed(s.elapsed))

	var statusLine string
	if s.done {
		if s.err != nil {
			statusLine = styleRealizeErr.Render("  ✗") + "  " +
				styleRealizeAppName.Render(s.appName) + "  " +
				styleRealizeStatus.Render("failed") + timer
		} else {
			statusLine = styleRealizeDone.Render("  ✓") + "  " +
				styleRealizeAppName.Render(s.appName) + "  " +
				styleRealizeStatus.Render("complete") + timer
		}
	} else {
		spin := styleRealizeSpinner.Render(spinnerFrames[s.frame%len(spinnerFrames)])
		statusLine = "  " + spin + "  " +
			styleRealizeAppName.Render(s.appName) + "  " +
			styleRealizeStatus.Render("realizing…") + timer
	}
	lines = append(lines, statusLine)

	// Progress bar — shown once the orchestrator reports total task count.
	if s.totalTasks > 0 {
		lines = append(lines, renderProgressBar(s.doneTasks, s.totalTasks, w, s.frame))
	}

	lines = append(lines, "")

	// Log area: fill remaining height with the most recent entries.
	// Reserve lines for header + status + blank (+ progress bar when visible).
	reserved := 3
	if s.totalTasks > 0 {
		reserved = 4
	}
	logH := h - reserved
	if logH < 0 {
		logH = 0
	}

	start := 0
	if len(s.logs) > logH {
		start = len(s.logs) - logH
	}

	maxTextW := w - 7 // 4 chars line-number + 1 prefix icon + 2 spaces
	if maxTextW < 10 {
		maxTextW = 10
	}

	for i, entry := range s.logs[start:] {
		lineNo := start + i + 1
		num := core.StyleLineNum.Render(fmt.Sprintf("%3d ", lineNo))
		prefix := logPrefix(entry.kind) + " "

		t := entry.text
		if len([]rune(t)) > maxTextW {
			runes := []rune(t)
			t = string(runes[:maxTextW-1]) + "…"
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

		lines = append(lines, num+prefix+colored)
	}

	return core.FillTildes(lines, h)
}
