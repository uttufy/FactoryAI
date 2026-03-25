package tui

import (
	"github.com/charmbracelet/bubbletea"

	"github.com/uttufy/FactoryAI/internal/events"
)

// TODO: v1.0 TUI to be implemented
// For now, this is a placeholder for the v1.0 TUI that will show
// batch progress, station status, etc.

type Model struct {
	Done   bool
	Error  error
	events <-chan events.Event
}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return "v1.0 TUI not yet implemented"
}
