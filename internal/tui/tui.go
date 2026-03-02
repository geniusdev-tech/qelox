// Package tui — interface TUI do qelox usando Bubble Tea.
// Teclas: S=start, X=stop, R=restart, Q=quit
package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zeus/qelox/internal/client"
	"github.com/zeus/qelox/internal/monitor"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FF41")). // Matrix Green
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#00FF41")).
			Padding(0, 2).
			MarginBottom(1)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0ea5e9")). // Cyan
			Padding(1, 2)

	styleLabelKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("#38bdf8")).Width(14) // Sky
	styleVal       = lipgloss.NewStyle().Foreground(lipgloss.Color("#f8fafc"))
	styleNeonGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ade80")).Bold(true).Italic(true)
	styleNeonRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f43f5e")).Bold(true).Italic(true)
	styleNeonAmber = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true)
	styleNeonCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22d3ee")).Bold(true)
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
	styleKeyBind   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF41")).
			Bold(true).
			Padding(0, 1)

	progressStyle = lipgloss.NewStyle().MarginLeft(2)
)

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	client  *client.Client
	metrics monitor.Metrics
	err     string
	width   int
	height  int
	cpuProg progress.Model
	ramProg progress.Model
}

type tickMsg time.Time
type metricsMsg monitor.Metrics
type errMsg string

func NewModel() Model {
	cp := progress.New(progress.WithDefaultGradient())
	rp := progress.New(progress.WithGradient("#22d3ee", "#a855f7"))
	return Model{
		client:  client.New(),
		cpuProg: cp,
		ramProg: rp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tick(), m.fetchMetrics())
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.SendRaw("metrics")
		if err != nil {
			return errMsg(err.Error())
		}
		var met monitor.Metrics
		json.Unmarshal(resp.Payload, &met)
		return metricsMsg(met)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		return m, tea.Batch(tick(), m.fetchMetrics())
	case metricsMsg:
		m.metrics = monitor.Metrics(msg)
		m.err = ""
	case errMsg:
		m.err = string(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		case "s", "S":
			return m, func() tea.Msg { m.client.Send("start"); return tickMsg(time.Now()) }
		case "x", "X":
			return m, func() tea.Msg { m.client.Send("stop"); return tickMsg(time.Now()) }
		case "r", "R":
			return m, func() tea.Msg { m.client.Send("restart"); return tickMsg(time.Now()) }
		}
	}
	return m, nil
}

func (m Model) View() string {
	met := m.metrics
	w := m.width
	if w < 60 {
		w = 60
	}

	header := styleTitle.Width(w - 6).Render("QELO-X  ORCHESTRATOR [v1.0.0]")

	stateStyle := styleNeonGreen
	if met.NodeState != "RUNNING" {
		stateStyle = styleNeonRed
	}

	row := func(label, val string) string {
		return fmt.Sprintf("  %s  %s", styleLabelKey.Render(label), styleVal.Render(val))
	}

	frozen := ""
	if met.Frozen {
		frozen = "  " + styleNeonRed.Render("⚠ FROZEN")
	}

	m.cpuProg.Width = w - 24
	m.ramProg.Width = w - 24

	body := styleBox.Width(w - 6).Render(lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("  %s  %s", styleLabelKey.Render("NODE STATUS"), stateStyle.Render(met.NodeState)),
		row("SYNC STATUS", strings.ToUpper(met.SyncStatus)),
		row("PEER COUNT", fmt.Sprintf("%d connected", met.PeerCount)),
		row("BLOCK HEIGHT", fmt.Sprintf("#%d", met.BlockHeight)+frozen),
		"\n",
		row("CPU USAGE", fmt.Sprintf("%.1f%%", met.CPUPercent)),
		progressStyle.Render(m.cpuProg.ViewAs(met.CPUPercent/100)),
		"\n",
		row("RAM USAGE", fmt.Sprintf("%.1f MB", float64(met.RAMBytes)/1024/1024)),
		progressStyle.Render(m.ramProg.ViewAs(met.RAMPercent/100)),
		"\n",
		row("DAEMON UPTIME", met.Uptime),
	))

	hints := fmt.Sprintf("  %s START  %s STOP  %s RESTART  %s QUIT",
		styleKeyBind.Render("[S]"),
		styleKeyBind.Render("[X]"),
		styleKeyBind.Render("[R]"),
		styleKeyBind.Render("[Q]"))

	errLine := ""
	if m.err != "" {
		errLine = "\n  " + styleNeonRed.Render("⚠ "+m.err)
	}

	footer := styleMuted.Render("  status: terminal active · system monitoring enabled")

	return fmt.Sprintf("\n%s\n\n%s\n\n%s%s\n\n%s\n",
		header, body, hints, errLine, footer)
}

// Run inicializa e executa o programa Bubble Tea.
func Run() error {
	m := NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
