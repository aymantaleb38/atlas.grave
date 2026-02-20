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
		// Calculate column widths based on total width
		pidW := 8
		sinW := 10
		burdenW := 10
		nameW := m.width - pidW - sinW - burdenW - 10
		if nameW < 10 { nameW = 10 }

		headStr := fmt.Sprintf("%-*s %-*s %-*s %-*s", pidW, "SOUL ID", nameW, "NAME", sinW, "SIN (CPU)", burdenW, "BURDEN")
		sb.WriteString(dimStyle.Render(headStr) + "\n")
		
		// Calculate display limit based on terminal height
		// Header(1) + Spacer(1) + Subheader(1) + List + Confirm/Help(2) + Border(2)
		displayHeight := m.height - 8
		if displayHeight < 5 { displayHeight = 5 }

		start := 0
		if m.cursor > displayHeight/2 {
			start = m.cursor - displayHeight/2
		}
		end := start + displayHeight
		if end > len(m.souls) {
			end = len(m.souls)
			start = end - displayHeight
			if start < 0 { start = 0 }
		}

		for i := start; i < end; i++ {
			s := m.souls[i]
			formatStr := fmt.Sprintf("%%-%dd %%-%d.%ds %%-%d.1f%%%% %%-%ds", pidW, nameW, nameW, sinW-1, burdenW)
			line := fmt.Sprintf(formatStr, s.PID, s.Name, s.CPU, formatBytes(s.Memory))
			
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

	// Apply screen border and force it to fill most of the space
	res := screenStyle.Width(m.width - 4).Height(m.height - 2).Render(sb.String())
	return res
}
