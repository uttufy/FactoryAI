package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var b strings.Builder

	title := m.styles.title.Render(fmt.Sprintf("🏭 %s", m.factoryName))
	if m.jobID != "" {
		title += m.styles.duration.Render(fmt.Sprintf("  [Job: %s]", m.jobID))
	}
	b.WriteString(title)
	b.WriteString("\n\n")

	lineViews := make([]string, len(m.lines))
	for i, line := range m.lines {
		lineViews[i] = m.renderLine(line)
	}

	linesRow := lipgloss.JoinHorizontal(lipgloss.Top, lineViews...)
	b.WriteString(linesRow)
	b.WriteString("\n\n")

	if m.done && m.finalOutput != "" {
		outputHeader := m.styles.title.Render("Final Output:")
		b.WriteString(outputHeader)
		b.WriteString("\n")
		output := m.styles.output.Render(m.finalOutput)
		b.WriteString(output)
		b.WriteString("\n\n")
	}

	help := m.styles.help.Render("[q] Quit")
	b.WriteString(help)

	return b.String()
}

func (m Model) renderLine(line LineView) string {
	var b strings.Builder

	headerStyle := m.styles.title
	if line.Status == StatusDone {
		headerStyle = headerStyle.Foreground(lipgloss.Color("10"))
	} else if line.Status == StatusFailed {
		headerStyle = headerStyle.Foreground(lipgloss.Color("9"))
	} else if line.Status == StatusRunning {
		headerStyle = headerStyle.Foreground(lipgloss.Color("11"))
	}

	b.WriteString(headerStyle.Render(line.Name))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 20))
	b.WriteString("\n")

	for _, station := range line.Stations {
		icon := statusIcons[station.Status]
		style := m.styles.status[station.Status]

		duration := ""
		if station.Duration > 0 {
			duration = fmt.Sprintf("%.1fs", station.Duration.Seconds())
		}

		retries := ""
		if station.Retries > 0 {
			retries = fmt.Sprintf(" (x%d)", station.Retries+1)
		}

		row := fmt.Sprintf(" %s %s%s %s",
			style.Render(icon),
			station.Name,
			retries,
			m.styles.duration.Render(duration),
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	return m.styles.lineBox.Render(b.String())
}
