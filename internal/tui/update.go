package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/uttufy/FactoryAI/internal/events"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case events.Event:
		m = m.handleEvent(msg)
		if msg.Type == events.EvtDone {
			m.done = true
			m.finalOutput = msg.Output
		}
		return m, waitForEvent(m.events)
	}

	return m, nil
}

func (m Model) handleEvent(evt events.Event) Model {
	for i := range m.lines {
		if m.lines[i].Name == evt.LineName {
			for j := range m.lines[i].Stations {
				if m.lines[i].Stations[j].Name == evt.StationName {
					m.updateStation(&m.lines[i].Stations[j], evt)
					m.updateLineStatus(&m.lines[i])
					break
				}
			}
			break
		}
	}
	return m
}

func (m Model) updateStation(station *StationView, evt events.Event) {
	switch evt.Type {
	case events.EvtStationStarted:
		station.Status = StatusRunning
	case events.EvtStationInspecting:
		station.Status = StatusInspecting
	case events.EvtStationDone:
		station.Status = StatusDone
		station.Duration = evt.Duration
		station.Output = evt.Output
		station.Retries = evt.Retries
	case events.EvtStationFailed:
		station.Status = StatusFailed
		station.Duration = evt.Duration
		station.Retries = evt.Retries
	}
}

func (m Model) updateLineStatus(line *LineView) {
	allDone := true
	anyFailed := false
	anyRunning := false

	for _, s := range line.Stations {
		switch s.Status {
		case StatusFailed:
			anyFailed = true
		case StatusRunning, StatusInspecting:
			anyRunning = true
			allDone = false
		case StatusPending:
			allDone = false
		}
	}

	switch {
	case anyFailed:
		line.Status = StatusFailed
	case allDone:
		line.Status = StatusDone
	case anyRunning:
		line.Status = StatusRunning
	default:
		line.Status = StatusPending
	}
}
