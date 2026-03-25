// Package tmux provides tmux session management for FactoryAI stations.
package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Session represents a tmux session
type Session struct {
	Name       string `json:"name"`
	Window     int    `json:"window"`
	Pane       int    `json:"pane"`
	WorkingDir string `json:"working_dir"`
	Command    string `json:"command,omitempty"`
	Pid        int    `json:"pid,omitempty"`
}

// Manager manages tmux sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewManager creates a new tmux manager
func NewManager() (*Manager, error) {
	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}

	return &Manager{
		sessions: make(map[string]*Session),
	}, nil
}

// runTmuxCommand executes a tmux command and returns the output
func (m *Manager) runTmuxCommand(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %v failed: %w (stderr: %s)", args, err, stderr.String())
	}

	return stdout.String(), nil
}

// CreateSession creates a new tmux session
func (m *Manager) CreateSession(name, workDir string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create session with named window
	if _, err := m.runTmuxCommand("new-session", "-d", "-s", name, "-c", workDir); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	session := &Session{
		Name:       name,
		Window:     0,
		Pane:       0,
		WorkingDir: workDir,
	}

	m.sessions[name] = session
	return session, nil
}

// SendKeys sends keystrokes to a session
func (m *Manager) SendKeys(session string, keys string) error {
	_, err := m.runTmuxCommand("send-keys", "-t", session, keys, "C-m")
	return err
}

// SendKeysToPane sends keystrokes to a specific pane
func (m *Manager) SendKeysToPane(session string, window, pane int, keys string) error {
	target := fmt.Sprintf("%s:%d.%d", session, window, pane)
	_, err := m.runTmuxCommand("send-keys", "-t", target, keys, "C-m")
	return err
}

// CaptureOutput captures the current pane output
func (m *Manager) CaptureOutput(session string) (string, error) {
	// Capture pane output (last 100 lines by default)
	output, err := m.runTmuxCommand("capture-pane", "-p", "-t", session)
	if err != nil {
		return "", fmt.Errorf("capturing output: %w", err)
	}
	return output, nil
}

// CapturePaneOutput captures output from a specific pane
func (m *Manager) CapturePaneOutput(session string, window, pane int) (string, error) {
	target := fmt.Sprintf("%s:%d.%d", session, window, pane)
	output, err := m.runTmuxCommand("capture-pane", "-p", "-t", target)
	if err != nil {
		return "", fmt.Errorf("capturing pane output: %w", err)
	}
	return output, nil
}

// KillSession kills a tmux session
func (m *Manager) KillSession(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := m.runTmuxCommand("kill-session", "-t", name); err != nil {
		return fmt.Errorf("killing session: %w", err)
	}

	delete(m.sessions, name)
	return nil
}

// ListSessions lists all tmux sessions
func (m *Manager) ListSessions() ([]*Session, error) {
	output, err := m.runTmuxCommand("list-sessions", "-F", "#{session_name}")
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var sessions []*Session

	for _, line := range lines {
		if line == "" {
			continue
		}
		sessions = append(sessions, &Session{Name: line})
	}

	return sessions, nil
}

// HasSession checks if a session exists
func (m *Manager) HasSession(name string) bool {
	sessions, err := m.ListSessions()
	if err != nil {
		return false
	}

	for _, s := range sessions {
		if s.Name == name {
			return true
		}
	}
	return false
}

// RenameSession renames a session
func (m *Manager) RenameSession(oldName, newName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := m.runTmuxCommand("rename-session", "-t", oldName, newName); err != nil {
		return fmt.Errorf("renaming session: %w", err)
	}

	if session, ok := m.sessions[oldName]; ok {
		session.Name = newName
		m.sessions[newName] = session
		delete(m.sessions, oldName)
	}

	return nil
}

// SplitPane splits a pane
func (m *Manager) SplitPane(session string, window, pane int, horizontal bool) (int, error) {
	target := fmt.Sprintf("%s:%d.%d", session, window, pane)

	var splitFlag string
	if horizontal {
		splitFlag = "-h"
	} else {
		splitFlag = "-v"
	}

	// Split the pane
	if _, err := m.runTmuxCommand("split-window", splitFlag, "-t", target); err != nil {
		return 0, fmt.Errorf("splitting pane: %w", err)
	}

	// Get the new pane number
	output, err := m.runTmuxCommand("list-panes", "-t", fmt.Sprintf("%s:%d", session, window), "-F", "#{pane_index}")
	if err != nil {
		return 0, fmt.Errorf("listing panes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	newPane := 0
	for _, line := range lines {
		if num, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && num > newPane {
			newPane = num
		}
	}

	return newPane, nil
}

// NewWindow creates a new window in a session
func (m *Manager) NewWindow(session string, name string) (int, error) {
	output, err := m.runTmuxCommand("new-window", "-t", session, "-n", name)
	if err != nil {
		return 0, fmt.Errorf("creating window: %w", err)
	}

	// Parse the window index from output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		parts := strings.Split(lines[0], "@")
		if len(parts) > 0 {
			windowNum := strings.TrimPrefix(parts[0], session+":")
			if num, err := strconv.Atoi(windowNum); err == nil {
				return num, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse window index from output")
}

// GetSessionInfo gets detailed information about a session
func (m *Manager) GetSessionInfo(name string) (*Session, error) {
	if !m.HasSession(name) {
		return nil, fmt.Errorf("session %s does not exist", name)
	}

	session := &Session{Name: name}

	// Get session working directory
	output, err := m.runTmuxCommand("display-message", "-p", "-t", name, "#{pane_current_path}")
	if err == nil {
		session.WorkingDir = strings.TrimSpace(output)
	}

	// Get current window and pane
	output, err = m.runTmuxCommand("display-message", "-p", "-t", name, "#{window_index}.#{pane_index}")
	if err == nil {
		parts := strings.Split(strings.TrimSpace(output), ".")
		if len(parts) == 2 {
			if window, err := strconv.Atoi(parts[0]); err == nil {
				session.Window = window
			}
			if pane, err := strconv.Atoi(parts[1]); err == nil {
				session.Pane = pane
			}
		}
	}

	return session, nil
}

// AttachSession attaches to a session (for interactive use)
func (m *Manager) AttachSession(name string) error {
	// This should typically not be used programmatically
	// as it will take over the terminal
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	return cmd.Run()
}
