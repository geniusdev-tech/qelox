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
			Foreground(lipgloss.Color("#10b981")). // Emerald
			Background(lipgloss.Color("#022c22")).
			Padding(0, 2).
			MarginBottom(1)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#334155")). // Slate
			Padding(1, 2)

	styleBoxActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0ea5e9")). // Cyan
			Padding(1, 2)

	styleLabelKey  = lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5e9")).Width(15) // Cyan
	styleVal       = lipgloss.NewStyle().Foreground(lipgloss.Color("#f8fafc"))
	styleNeonGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Bold(true)
	styleNeonRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f43f5e")).Bold(true)
	styleNeonAmber = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24")).Bold(true)
	styleNeonCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22d3ee")).Bold(true)
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748b"))
	styleKeyBind   = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10b981")).
			Background(lipgloss.Color("#022c22")).
			Bold(true).
			Padding(0, 1)

	progressStyle = lipgloss.NewStyle().MarginLeft(2)
)

func fmtBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

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

	header := styleTitle.Width(w - 4).Render(" QELO-X NODE ORCHESTRATOR ")

	stateStyle := styleNeonGreen
	boxStyle := styleBoxActive
	if met.NodeState != "RUNNING" {
		stateStyle = styleNeonRed
		boxStyle = styleBox
	}

	row := func(label, val string) string {
		return fmt.Sprintf(" %s %s", styleLabelKey.Render(label), styleVal.Render(val))
	}

	frozen := ""
	if met.Frozen {
		frozen = " " + styleNeonRed.Render("⚠ FROZEN")
	}

	// Dynamic Widths for 2 Columns
	colWidth := (w - 10) / 2
	if colWidth < 40 {
		colWidth = 40
	}
	m.cpuProg.Width = colWidth - 20
	m.ramProg.Width = colWidth - 20

	peerStr := fmt.Sprintf("%d connected", met.PeerCount)
	if met.PeerCount == -1 {
		if met.GoQuaiTCPSockets > 0 {
			peerStr = fmt.Sprintf("~%d (sockets)", met.GoQuaiTCPSockets)
		} else {
			peerStr = "N/A"
		}
	}

	col1 := boxStyle.Width(colWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf(" %s %s", styleLabelKey.Render("NODE STATUS"), stateStyle.Render(met.NodeState)),
		row("SYNC STATUS", strings.ToUpper(met.SyncStatus)),
		row("PEER COUNT", peerStr),
		row("BLOCK HEIGHT", fmt.Sprintf("#%d", met.BlockHeight)+frozen),
		row("TCP SOCKETS", fmt.Sprintf("%d", met.GoQuaiTCPSockets)),
		row("ACTIVE THREADS", fmt.Sprintf("%d", met.GoQuaiThreads)),
		row("BLOCKS/MIN", fmt.Sprintf("%.1f", met.BlocksPerMinute)),
		row("UPTIME", met.Uptime),
		row("NETWORK ID", strings.ToUpper(met.NetworkID)),
		row("TARGET SLICE", met.SliceID),
	))

	col2 := styleBox.Width(colWidth).Render(lipgloss.JoinVertical(lipgloss.Left,
		row("SYSTEM LOAD", fmt.Sprintf("%.2f, %.2f, %.2f", met.LoadAvg1, met.LoadAvg5, met.LoadAvg15)),
		row("DATA DIR DISK", fmt.Sprintf("%s / %s (%.1f%%)", fmtBytes(met.DiskUsedBytes), fmtBytes(met.DiskUsedBytes+met.DiskFreeBytes), met.DiskUsedPct)),
		row("NET RECV/s", fmtBytes(met.NetRecvBytes)+"/s"),
		row("NET SENT/s", fmtBytes(met.NetSentBytes)+"/s"),
		"\n",
		row("CPU USAGE", fmt.Sprintf("%.1f%%", met.CPUPercent)),
		progressStyle.Render(m.cpuProg.ViewAs(met.CPUPercent/100)),
		"\n",
		row("RAM USAGE", fmt.Sprintf("%s (Total) / %s (Node)", fmtBytes(met.RAMBytes), fmtBytes(met.GoQuaiRAMBytes))),
		progressStyle.Render(m.ramProg.ViewAs(met.RAMPercent/100)),
	))

	body := lipgloss.JoinHorizontal(lipgloss.Top, col1, "  ", col2)

	hints := fmt.Sprintf("  %s START   %s STOP   %s RESTART   %s QUIT",
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
