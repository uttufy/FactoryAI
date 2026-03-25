// Package assembly implements the Final Assembly - merge queue with conflict detection.
package assembly

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uttufy/FactoryAI/internal/beads"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/store"
)

// MergeStatus represents the status of a merge request
type MergeStatus string

const (
	MergePending    MergeStatus = "pending"
	MergeChecking   MergeStatus = "checking"    // Checking for conflicts
	MergeReady      MergeStatus = "ready"       // No conflicts
	MergeConflicted MergeStatus = "conflicted"
	MergeMerging    MergeStatus = "merging"
	MergeComplete   MergeStatus = "complete"
	MergeFailed     MergeStatus = "failed"
)

// MergeRequest represents a merge request
type MergeRequest struct {
	ID          string       `json:"id"`
	BeadID      string       `json:"bead_id"`
	StationID   string       `json:"station_id"`
	Branch      string       `json:"branch"`
	Status      MergeStatus  `json:"status"`
	Priority    int          `json:"priority"`
	Conflicts   []string     `json:"conflicts,omitempty"`
	SubmittedAt time.Time    `json:"submitted_at"`
	MergedAt    *time.Time   `json:"merged_at,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// Assembly is the merge queue manager
type Assembly struct {
	projectPath string
	events      *events.EventBus
	store       *store.Store
	client      *beads.Client
	queue       []*MergeRequest
	mu          sync.RWMutex
}

// NewAssembly creates a new assembly manager
func NewAssembly(projectPath string, events *events.EventBus, store *store.Store, client *beads.Client) *Assembly {
	a := &Assembly{
		projectPath: projectPath,
		events:      events,
		store:       store,
		client:      client,
		queue:       make([]*MergeRequest, 0),
	}

	// Subscribe to events - commented out for now
	// events.Subscribe(events.EventMergeReady, a.handleMergeReady)

	return a
}

// Submit adds a merge request to the queue
func (a *Assembly) Submit(ctx context.Context, beadID, stationID, branch string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mr := &MergeRequest{
		ID:          uuid.New().String(),
		BeadID:      beadID,
		StationID:   stationID,
		Branch:      branch,
		Status:      MergePending,
		Priority:    0,
		SubmittedAt: time.Now(),
	}

	a.queue = append(a.queue, mr)

	a.events.Emit(events.EventMergeReady, "assembly", mr.ID, map[string]interface{}{
		"bead_id":    beadID,
		"station_id": stationID,
		"branch":     branch,
	})

	return nil
}

// CheckConflicts pre-checks for merge conflicts
func (a *Assembly) CheckConflicts(ctx context.Context, mrID string) ([]string, error) {
	a.mu.RLock()
	mr, err := a.getMergeRequest(mrID)
	a.mu.RUnlock()

	if err != nil {
		return nil, err
	}

	// Run git merge check
	cmd := exec.Command("git", "merge", "--no-commit", "--no-ff", mr.Branch)
	cmd.Dir = a.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a conflict error
		outputStr := string(output)
		if strings.Contains(outputStr, "CONFLICT") {
			// Parse conflicting files
			conflicts := a.parseConflicts(outputStr)
			return conflicts, nil
		}
		return nil, fmt.Errorf("merge check failed: %w", err)
	}

	// Abort the test merge
	_ = exec.Command("git", "merge", "--abort").Run()

	return []string{}, nil
}

// parseConflicts parses git conflict output
func (a *Assembly) parseConflicts(output string) []string {
	var conflicts []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "CONFLICT (content):") {
			parts := strings.Split(line, ":")
			if len(parts) >= 3 {
				conflictFile := strings.TrimSpace(parts[2])
				conflicts = append(conflicts, conflictFile)
			}
		}
	}

	return conflicts
}

// CanMergeParallel checks if MRs can be merged concurrently (no file overlap)
func (a *Assembly) CanMergeParallel(ctx context.Context, mrIDs []string) (bool, error) {
	conflictMaps := make([]map[string]bool, len(mrIDs))

	for i, mrID := range mrIDs {
		conflicts, err := a.CheckConflicts(ctx, mrID)
		if err != nil {
			return false, err
		}

		conflictMaps[i] = make(map[string]bool)
		for _, f := range conflicts {
			conflictMaps[i][f] = true
		}
	}

	// Check for overlapping files
	for i := 0; i < len(conflictMaps); i++ {
		for j := i + 1; j < len(conflictMaps); j++ {
			for file := range conflictMaps[i] {
				if conflictMaps[j][file] {
					return false, nil // Overlapping conflict found
				}
			}
		}
	}

	return true, nil
}

// ProcessQueue processes all ready merge requests in parallel
func (a *Assembly) ProcessQueue(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Group by priority and check for parallel merge capability
	var ready []*MergeRequest
	for _, mr := range a.queue {
		if mr.Status == MergeReady {
			ready = append(ready, mr)
		}
	}

	if len(ready) == 0 {
		return nil
	}

	// For now, process sequentially
	// In future, use CanMergeParallel to group compatible merges
	for _, mr := range ready {
		if err := a.Merge(ctx, mr.ID); err != nil {
			return fmt.Errorf("merging %s: %w", mr.ID, err)
		}
	}

	return nil
}

// Merge performs the actual merge
func (a *Assembly) Merge(ctx context.Context, mrID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mr, err := a.getMergeRequest(mrID)
	if err != nil {
		return err
	}

	mr.Status = MergeMerging
	a.events.Emit(events.EventMergeStarted, "assembly", mr.ID, nil)

	// Perform the merge
	cmd := exec.Command("git", "merge", "--no-ff", mr.Branch)
	cmd.Dir = a.projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		mr.Status = MergeFailed
		mr.Error = string(output)
		now := time.Now()
		mr.MergedAt = &now

		a.events.Emit(events.EventMergeConflict, "assembly", mr.ID, map[string]interface{}{
			"error": string(output),
		})

		return fmt.Errorf("merge failed: %w", err)
	}

	now := time.Now()
	mr.Status = MergeComplete
	mr.MergedAt = &now

	a.events.Emit(events.EventMergeCompleted, "assembly", mr.ID, nil)

	// Remove from queue
	a.removeFromQueue(mrID)

	return nil
}

// GetQueue returns the current merge queue
func (a *Assembly) GetQueue(ctx context.Context) []*MergeRequest {
	a.mu.RLock()
	defer a.mu.RUnlock()

	queue := make([]*MergeRequest, len(a.queue))
	copy(queue, a.queue)

	return queue
}

// Escalate marks an MR for human attention
func (a *Assembly) Escalate(ctx context.Context, mrID, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mr, err := a.getMergeRequest(mrID)
	if err != nil {
		return err
	}

	mr.Status = MergeConflicted
	mr.Error = reason

	a.events.Emit(events.EventMergeConflict, "assembly", mr.ID, map[string]interface{}{
		"reason": reason,
	})

	return nil
}

// Start begins listening for merge events
func (a *Assembly) Start(ctx context.Context) error {
	// Start queue processor
	go a.queueProcessor(ctx)

	return nil
}

// queueProcessor processes the merge queue
func (a *Assembly) queueProcessor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = a.ProcessQueue(ctx)
		}
	}
}

// handleMergeReady handles merge ready events
func (a *Assembly) handleMergeReady(evt events.Event) {
	mrID, ok := evt.Payload["mr_id"].(string)
	if !ok {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Update status to checking
	for _, mr := range a.queue {
		if mr.ID == mrID {
			mr.Status = MergeChecking
			break
		}
	}

	// Check for conflicts
	go func() {
		conflicts, err := a.CheckConflicts(context.Background(), mrID)
		a.mu.Lock()
		defer a.mu.Unlock()

		for _, mr := range a.queue {
			if mr.ID == mrID {
				if err != nil {
					mr.Status = MergeFailed
					mr.Error = err.Error()
				} else if len(conflicts) > 0 {
					mr.Status = MergeConflicted
					mr.Conflicts = conflicts
				} else {
					mr.Status = MergeReady
				}
				break
			}
		}
	}()
}

// getMergeRequest gets a merge request by ID
func (a *Assembly) getMergeRequest(id string) (*MergeRequest, error) {
	for _, mr := range a.queue {
		if mr.ID == id {
			return mr, nil
		}
	}
	return nil, fmt.Errorf("merge request not found: %s", id)
}

// removeFromQueue removes a merge request from the queue
func (a *Assembly) removeFromQueue(id string) {
	for i, mr := range a.queue {
		if mr.ID == id {
			a.queue = append(a.queue[:i], a.queue[i+1:]...)
			return
		}
	}
}

// GetPendingCount returns the number of pending merges
func (a *Assembly) GetPendingCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	count := 0
	for _, mr := range a.queue {
		if mr.Status == MergePending || mr.Status == MergeReady {
			count++
		}
	}

	return count
}

// GetConflictedCount returns the number of conflicted merges
func (a *Assembly) GetConflictedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	count := 0
	for _, mr := range a.queue {
		if mr.Status == MergeConflicted {
			count++
		}
	}

	return count
}

// PrioritizeMerge updates the priority of a merge request
func (a *Assembly) PrioritizeMerge(ctx context.Context, mrID string, priority int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, mr := range a.queue {
		if mr.ID == mrID {
			mr.Priority = priority
			return nil
		}
	}

	return fmt.Errorf("merge request not found: %s", mrID)
}

// AbortMerge aborts an in-progress merge
func (a *Assembly) AbortMerge(ctx context.Context, mrID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	mr, err := a.getMergeRequest(mrID)
	if err != nil {
		return err
	}

	if mr.Status != MergeMerging {
		return fmt.Errorf("merge not in progress: %s", mrID)
	}

	// Abort git merge
	cmd := exec.Command("git", "merge", "--abort")
	cmd.Dir = a.projectPath

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aborting merge: %w", err)
	}

	mr.Status = MergePending

	return nil
}
