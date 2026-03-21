package tui

import (
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
)

type StationStatus int

const (
	StatusPending StationStatus = iota
	StatusRunning
	StatusInspecting
	StatusDone
	StatusFailed
)

type StationView struct {
	Name     string
	Status   StationStatus
	Duration time.Duration
	Output   string
	Retries  int
}

type LineView struct {
	Name     string
	Stations []StationView
	Status   StationStatus
	Output   string
}

type Model struct {
	factoryName string
	jobID       string
	lines       []LineView
	finalOutput string
	done        bool
	err         error

	events <-chan events.Event

	styles Styles
}

type Styles struct {
	title     lipgloss.Style
	lineBox   lipgloss.Style
	station   lipgloss.Style
	duration  lipgloss.Style
	output    lipgloss.Style
	help      lipgloss.Style
	status    map[StationStatus]lipgloss.Style
}

var statusIcons = map[StationStatus]string{
	StatusPending:    "○",
	StatusRunning:    "⠿",
	StatusInspecting: "🔍",
	StatusDone:       "✓",
	StatusFailed:     "✗",
}

func NewModel(blueprint *config.Blueprint, eventsChan <-chan events.Event) Model {
	lines := make([]LineView, len(blueprint.Factory.AssemblyLines))
	for i, lineCfg := range blueprint.Factory.AssemblyLines {
		stations := make([]StationView, len(lineCfg.Stations))
		for j, stationCfg := range lineCfg.Stations {
			stations[j] = StationView{
				Name:   stationCfg.Name,
				Status: StatusPending,
			}
		}
		lines[i] = LineView{
			Name:     lineCfg.Name,
			Stations: stations,
			Status:   StatusPending,
		}
	}

	return Model{
		factoryName: blueprint.Factory.Name,
		lines:       lines,
		events:      eventsChan,
		styles:      defaultStyles(),
	}
}

func defaultStyles() Styles {
	return Styles{
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			Padding(0, 1),
		lineBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1).
			Margin(0, 1),
		duration: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		output: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2).
			Margin(1, 0),
		help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		status: map[StationStatus]lipgloss.Style{
			StatusPending:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
			StatusRunning:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
			StatusInspecting: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
			StatusDone:       lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
			StatusFailed:     lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return waitForEvent(m.events)
}

func waitForEvent(events <-chan events.Event) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return nil
		}
		return <-events
	}
}
