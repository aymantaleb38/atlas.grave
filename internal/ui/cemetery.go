package ui

import (
	"fmt"
	"strings"
	"time"

	"atlas.grave/internal/model"
	"atlas.grave/internal/system"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	amber  = lipgloss.Color("#FFB642")
	rusty  = lipgloss.Color("#5E4737")
	ncrRed = lipgloss.Color("#FF0000")
	onyx   = lipgloss.Color("#050505")

	screenStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(amber).
			Background(onyx)

	headerStyle = lipgloss.NewStyle().
			Foreground(onyx).
			Background(amber).
			Padding(0, 1).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(onyx).
			Background(amber).
			Bold(true)

	textStyle   = lipgloss.NewStyle().Foreground(amber)
	dimStyle    = lipgloss.NewStyle().Foreground(rusty)
	restlessStyle = lipgloss.NewStyle().Foreground(ncrRed).Bold(true)
)

type state int
const (
	stateBrowsing state = iota
	stateConfirming
)

type Model struct {
	reaper   *system.Reaper
	souls    []model.Soul
	cursor   int
	width    int
	height   int
	state    state
	reclaimed uint64
}

func NewModel(r *system.Reaper) Model {
	return Model{
		reaper: r,
		souls:  []model.Soul{},
	}
}

type scanMsg []model.Soul
type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.scanCmd(), tick())
}

func (m Model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		souls, _ := m.reaper.Scan()
		return scanMsg(souls)
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateConfirming {
			switch msg.String() {
			case "y", "Y":
				if len(m.souls) > 0 && m.cursor < len(m.souls) {
					soul := m.souls[m.cursor]
					m.reclaimed += soul.Memory
					m.reaper.Bury(soul.PID)
				}
				m.state = stateBrowsing
				return m, m.scanCmd()
			case "n", "N", "esc":
				m.state = stateBrowsing
				return m, nil
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.souls)-1 { m.cursor++ }
		case "enter", "b":
			if len(m.souls) > 0 {
				m.state = stateConfirming
			}
		}

	case scanMsg:
		m.souls = msg
		if m.cursor >= len(m.souls) && len(m.souls) > 0 {
			m.cursor = len(m.souls)-1
		}
	case tickMsg:
		if m.state == stateBrowsing {
			return m, tea.Batch(m.scanCmd(), tick())
		}
		return m, tick()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}
	return m, nil
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit { return fmt.Sprintf("%d B", b) }
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 { return "" }
	var sb strings.Builder

	header := headerStyle.Render(" ATLAS REAPER 3000 ") + " " + textStyle.Render(fmt.Sprintf("RECLAIMED: %s", formatBytes(m.reclaimed)))
	sb.WriteString(header + "\n\n")

	if len(m.souls) == 0 {
		sb.WriteString(dimStyle.Render("SCANNING FOR SOULS...") + "\n")
	} else {
		// Fixed column widths for stability
		sb.WriteString(dimStyle.Render(fmt.Sprintf("%-8s %-20s %-10s %-10s", "SOUL ID", "NAME", "SIN", "BURDEN")) + "\n")
		
		start := 0
		if m.cursor > 10 { start = m.cursor - 10 }
		end := start + 15
		if end > len(m.souls) { end = len(m.souls) }

		for i := start; i < end; i++ {
			s := m.souls[i]
			line := fmt.Sprintf("%-8d %-20.20s %-10.1f%% %-10s", s.PID, s.Name, s.CPU, formatBytes(s.Memory))
			
			style := textStyle
			if s.CPU > 50 || s.Memory > 1024*1024*500 {
				style = restlessStyle
			}

			if i == m.cursor {
				sb.WriteString(selectedStyle.Render("> "+line) + "\n")
			} else {
				sb.WriteString("  " + style.Render(line) + "\n")
			}
		}
	}

	if m.state == stateConfirming && len(m.souls) > 0 {
		target := m.souls[m.cursor]
		sb.WriteString("\n" + restlessStyle.Render(fmt.Sprintf("BURY SOUL %s (PID %d)? (y/n)", target.Name, target.PID)))
	} else {
		sb.WriteString("\n" + dimStyle.Render("J/K: NAVIGATE • ENTER: BURY • Q: EXIT"))
	}

	return screenStyle.Render(sb.String())
}
