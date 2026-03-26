// Package beads implements the Beads CLI client for FactoryAI.
// This package wraps the beads CLI tool for work item management.
package beads

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client wraps the beads CLI
type Client struct {
	binaryPath string
	workingDir string
}

// NewClient creates a new beads CLI client
func NewClient(binaryPath, workingDir string) (*Client, error) {
	// If binaryPath is empty, try to find beads in PATH
	if binaryPath == "" {
		path, err := exec.LookPath("beads")
		if err != nil {
			return nil, fmt.Errorf("beads CLI not found in PATH: %w", err)
		}
		binaryPath = path
	}

	return &Client{
		binaryPath: binaryPath,
		workingDir: workingDir,
	}, nil
}

// Execute runs a beads command and returns the output
func (c *Client) Execute(args ...string) (string, error) {
	cmd := exec.Command(c.binaryPath, args...)
	if c.workingDir != "" {
		cmd.Dir = c.workingDir
	}

	// Disable pager for all commands
	cmd.Env = append(os.Environ(), "PAGER=cat", "GIT_PAGER=cat")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("beads %v: %w (output: %s)", args, err, string(output))
	}

	return string(output), nil
}

// ExecuteJSON runs a beads command and parses JSON output
func (c *Client) ExecuteJSON(args ...string) (map[string]interface{}, error) {
	output, err := c.Execute(args...)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return result, nil
}

// Create creates a new bead
func (c *Client) Create(beadType, title string) (*Bead, error) {
	args := []string{"create", "--type", string(beadType), "--title", title}
	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("creating bead: %w", err)
	}

	// Parse the bead ID from output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Created bead:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return c.Get(strings.TrimSpace(parts[1]))
			}
		}
	}

	return nil, fmt.Errorf("could not parse bead ID from output: %s", output)
}

// Get retrieves a bead by ID
func (c *Client) Get(id string) (*Bead, error) {
	args := []string{"get", id}
	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("getting bead: %w", err)
	}

	var bead Bead
	if err := json.Unmarshal([]byte(output), &bead); err != nil {
		return nil, fmt.Errorf("parsing bead: %w", err)
	}

	return &bead, nil
}

// Update updates a bead
func (c *Client) Update(id string, updates map[string]interface{}) error {
	args := []string{"update", id}

	for key, value := range updates {
		args = append(args, "--set", fmt.Sprintf("%s=%v", key, value))
	}

	_, err := c.Execute(args...)
	return err
}

// List lists beads with optional filter
func (c *Client) List(filter BeadFilter) ([]*Bead, error) {
	args := []string{"list"}

	if filter.Type != "" {
		args = append(args, "--type", string(filter.Type))
	}
	if filter.Status != "" {
		args = append(args, "--status", string(filter.Status))
	}
	if filter.Assignee != "" {
		args = append(args, "--assignee", filter.Assignee)
	}
	if filter.StationID != "" {
		args = append(args, "--station", filter.StationID)
	}
	if filter.Pinned != nil {
		if *filter.Pinned {
			args = append(args, "--pinned")
		} else {
			args = append(args, "--unpinned")
		}
	}

	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("listing beads: %w", err)
	}

	var beads []*Bead
	if err := json.Unmarshal([]byte(output), &beads); err != nil {
		return nil, fmt.Errorf("parsing beads: %w", err)
	}

	return beads, nil
}

// Delete deletes a bead
func (c *Client) Delete(id string) error {
	_, err := c.Execute("delete", id)
	return err
}

// Close marks a bead as done
func (c *Client) Close(id string) error {
	_, err := c.Execute("close", id)
	return err
}

// Ready gets ready (pending) work
func (c *Client) Ready() ([]*Bead, error) {
	output, err := c.Execute("ready")
	if err != nil {
		return nil, fmt.Errorf("getting ready beads: %w", err)
	}

	var beads []*Bead
	if err := json.Unmarshal([]byte(output), &beads); err != nil {
		return nil, fmt.Errorf("parsing beads: %w", err)
	}

	return beads, nil
}

// CreateWisp creates an ephemeral wisp bead
func (c *Client) CreateWisp(beadType, title string) (*Bead, error) {
	args := []string{"create", "--wisp", "--type", string(beadType), "--title", title}
	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("creating wisp: %w", err)
	}

	// Parse the bead ID from output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Created bead:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return c.Get(strings.TrimSpace(parts[1]))
			}
		}
	}

	return nil, fmt.Errorf("could not parse bead ID from output: %s", output)
}

// Burn burns (deletes) a wisp bead
func (c *Client) Burn(id string) error {
	_, err := c.Execute("burn", id)
	return err
}

// Squash squashes a wisp into a summary bead
func (c *Client) Squash(id, summary string) error {
	_, err := c.Execute("squash", id, "--summary", summary)
	return err
}

// CreateEpic creates a new epic
func (c *Client) CreateEpic(title string) (*Epic, error) {
	args := []string{"epic", "create", "--title", title}
	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("creating epic: %w", err)
	}

	var epic Epic
	if err := json.Unmarshal([]byte(output), &epic); err != nil {
		return nil, fmt.Errorf("parsing epic: %w", err)
	}

	return &epic, nil
}

// AddChild adds a child bead to an epic
func (c *Client) AddChild(epicID, childID string) error {
	_, err := c.Execute("epic", "add-child", epicID, childID)
	return err
}

// GetChildren gets all children of an epic
func (c *Client) GetChildren(epicID string) ([]*Bead, error) {
	output, err := c.Execute("epic", "children", epicID)
	if err != nil {
		return nil, fmt.Errorf("getting epic children: %w", err)
	}

	var beads []*Bead
	if err := json.Unmarshal([]byte(output), &beads); err != nil {
		return nil, fmt.Errorf("parsing beads: %w", err)
	}

	return beads, nil
}

// CreateMolecule creates a new molecule (SOP template)
func (c *Client) CreateMolecule(name string, steps []*Step) (*Molecule, error) {
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("marshaling steps: %w", err)
	}

	args := []string{"molecule", "create", "--name", name, "--steps", string(stepsJSON)}
	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("creating molecule: %w", err)
	}

	var molecule Molecule
	if err := json.Unmarshal([]byte(output), &molecule); err != nil {
		return nil, fmt.Errorf("parsing molecule: %w", err)
	}

	return &molecule, nil
}

// InstantiateMolecule instantiates a molecule template
func (c *Client) InstantiateMolecule(protoID string, vars map[string]string) (*Molecule, error) {
	args := []string{"molecule", "instantiate", protoID}

	for key, value := range vars {
		args = append(args, "--var", fmt.Sprintf("%s=%s", key, value))
	}

	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("instantiating molecule: %w", err)
	}

	var molecule Molecule
	if err := json.Unmarshal([]byte(output), &molecule); err != nil {
		return nil, fmt.Errorf("parsing molecule: %w", err)
	}

	return &molecule, nil
}

// AttachTraveler attaches a traveler to a station
func (c *Client) AttachTraveler(stationID, beadID string, opts ...AttachOption) error {
	traveler := &Traveler{
		StationID: stationID,
		BeadID:    beadID,
	}

	for _, opt := range opts {
		opt(traveler)
	}

	args := []string{"traveler", "attach", "--station", stationID, "--bead", beadID}

	if traveler.Priority > 0 {
		args = append(args, "--priority", fmt.Sprintf("%d", traveler.Priority))
	}
	if traveler.Deferred {
		args = append(args, "--defer")
	}
	if traveler.Restart {
		args = append(args, "--restart")
	}

	_, err := c.Execute(args...)
	return err
}

// GetTraveler gets the traveler for a station
func (c *Client) GetTraveler(stationID string) (*Traveler, error) {
	output, err := c.Execute("traveler", "get", "--station", stationID)
	if err != nil {
		return nil, fmt.Errorf("getting traveler: %w", err)
	}

	var traveler Traveler
	if err := json.Unmarshal([]byte(output), &traveler); err != nil {
		return nil, fmt.Errorf("parsing traveler: %w", err)
	}

	return &traveler, nil
}

// ClearTraveler clears the traveler from a station
func (c *Client) ClearTraveler(stationID string) error {
	_, err := c.Execute("traveler", "clear", "--station", stationID)
	return err
}

// SendMail sends mail to a station
func (c *Client) SendMail(from, to, subject, body string) error {
	_, err := c.Execute("mail", "send", "--from", from, "--to", to, "--subject", subject, "--body", body)
	return err
}

// ReadMail reads mail for a station
func (c *Client) ReadMail(stationID string) ([]*Message, error) {
	output, err := c.Execute("mail", "read", "--station", stationID)
	if err != nil {
		return nil, fmt.Errorf("reading mail: %w", err)
	}

	var messages []*Message
	if err := json.Unmarshal([]byte(output), &messages); err != nil {
		return nil, fmt.Errorf("parsing messages: %w", err)
	}

	return messages, nil
}

// Route creates a client for a cross-rig routing
func (c *Client) Route(prefix string) (*Client, error) {
	// This is a placeholder for cross-rig routing functionality
	// In the full implementation, this would return a client configured
	// for a different beads rig (remote repository)
	return &Client{
		binaryPath: c.binaryPath,
		workingDir: c.workingDir,
	}, nil
}

// CreateBatch creates a new production batch
func (c *Client) CreateBatch(name string, jobIDs []string) (*Batch, error) {
	args := []string{"batch", "create", "--name", name}
	for _, id := range jobIDs {
		args = append(args, "--job", id)
	}

	output, err := c.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("creating batch: %w", err)
	}

	var batch Batch
	if err := json.Unmarshal([]byte(output), &batch); err != nil {
		return nil, fmt.Errorf("parsing batch: %w", err)
	}

	return &batch, nil
}

// GetBatch retrieves a batch by ID
func (c *Client) GetBatch(id string) (*Batch, error) {
	output, err := c.Execute("batch", "get", id)
	if err != nil {
		return nil, fmt.Errorf("getting batch: %w", err)
	}

	var batch Batch
	if err := json.Unmarshal([]byte(output), &batch); err != nil {
		return nil, fmt.Errorf("parsing batch: %w", err)
	}

	return &batch, nil
}

// ListBatches lists all batches
func (c *Client) ListBatches() ([]*Batch, error) {
	output, err := c.Execute("batch", "list")
	if err != nil {
		return nil, fmt.Errorf("listing batches: %w", err)
	}

	var batches []*Batch
	if err := json.Unmarshal([]byte(output), &batches); err != nil {
		return nil, fmt.Errorf("parsing batches: %w", err)
	}

	return batches, nil
}

// Init initializes beads in the working directory
// Runs: bd init --prefix <prefix>
func (c *Client) Init(prefix string) error {
	args := []string{"init", "--prefix", prefix}
	_, err := c.Execute(args...)
	if err != nil {
		return fmt.Errorf("initializing beads: %w", err)
	}
	return nil
}

// Doctor runs beads doctor to check system health
// Returns error if doctor finds critical errors
func (c *Client) Doctor() error {
	args := []string{"doctor"}
	cmd := exec.Command(c.binaryPath, args...)
	if c.workingDir != "" {
		cmd.Dir = c.workingDir
	}

	// Disable pager for all commands
	cmd.Env = append(os.Environ(), "PAGER=cat", "GIT_PAGER=cat")

	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Check for critical errors in output - look for error symbol with "Installation" error
	// which indicates beads is not initialized
	// Note: bd doctor returns exit code 1 for warnings too, so we check the actual content
	if strings.Contains(outputStr, "✖") {
		// Check for critical errors (like no .beads directory)
		if strings.Contains(outputStr, "No .beads/") || strings.Contains(outputStr, "Installation:") {
			return fmt.Errorf("beads doctor found errors:\n%s", outputStr)
		}
	}

	return nil
}

// InstallHooks installs recommended git hooks
// Runs: bd hooks install
func (c *Client) InstallHooks() error {
	args := []string{"hooks", "install"}
	_, err := c.Execute(args...)
	if err != nil {
		return fmt.Errorf("installing git hooks: %w", err)
	}
	return nil
}

// IsInitialized checks if beads is initialized in the working directory
func (c *Client) IsInitialized() bool {
	args := []string{"doctor"}
	output, err := c.Execute(args...)
	if err != nil {
		return false
	}

	// Check if .beads directory exists
	return !strings.Contains(output, "No .beads/ directory found")
}
