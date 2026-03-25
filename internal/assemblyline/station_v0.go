// Package assemblyline provides v0.x station implementation for backward compatibility.
package assemblyline

import (
	"context"
	"fmt"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/inspector"
	"github.com/uttufy/FactoryAI/internal/job"
	"github.com/uttufy/FactoryAI/internal/worker"
)

// StationV0 represents a v0.x station (for backward compatibility)
type StationV0 struct {
	config    config.StationConfig
	worker    *worker.Worker
	inspector *inspector.Inspector
	lineName  string
}

// NewV0Station creates a new v0.x station
func NewV0Station(cfg config.StationConfig, agent agents.Agent, lineName string) *StationV0 {
	w := worker.New(cfg, agent)

	var insp *inspector.Inspector
	if cfg.Inspector != nil && cfg.Inspector.Enabled {
		insp = inspector.New(agent, *cfg.Inspector)
	}

	return &StationV0{
		config:    cfg,
		worker:    w,
		inspector: insp,
		lineName:  lineName,
	}
}

func (s *StationV0) Name() string {
	return s.config.Name
}

func (s *StationV0) Run(ctx context.Context, task, context string, eventsChan chan<- events.Event) (job.StationResult, error) {
	maxAttempts := 1
	if s.inspector != nil && s.config.Inspector != nil {
		maxAttempts = s.config.Inspector.MaxRetries + 1
	}

	var result job.StationResult
	var lastOutput string

	for attempt := 0; attempt < maxAttempts; attempt++ {
		eventsChan <- events.StationStarted(s.lineName, s.config.Name)

		result = s.worker.Run(ctx, task, context)
		if result.Error != nil {
			eventsChan <- events.StationFailed(s.lineName, s.config.Name, result.Duration, result.Error, attempt)
			return result, result.Error
		}

		lastOutput = result.Output

		if s.inspector == nil {
			result.Passed = true
			result.RetriesUsed = attempt
			eventsChan <- events.StationDone(s.lineName, s.config.Name, result.Duration, result.Output, attempt)
			return result, nil
		}

		eventsChan <- events.StationInspecting(s.lineName, s.config.Name)

		passed, reasoning, err := s.inspector.Inspect(ctx, task, result.Output)
		if err != nil {
			eventsChan <- events.StationFailed(s.lineName, s.config.Name, result.Duration, err, attempt)
			return result, err
		}

		result.Passed = passed
		result.Reasoning = reasoning
		result.RetriesUsed = attempt

		if passed {
			eventsChan <- events.StationDone(s.lineName, s.config.Name, result.Duration, result.Output, attempt)
			return result, nil
		}

		context = fmt.Sprintf("%s\n\nPrevious output:\n%s\n\nInspector feedback (attempt %d/%d): %s",
			context, lastOutput, attempt+1, maxAttempts, reasoning)
	}

	eventsChan <- events.StationFailed(s.lineName, s.config.Name, result.Duration,
		fmt.Errorf("inspection failed after %d attempts: %s", maxAttempts, result.Reasoning), maxAttempts-1)

	return result, fmt.Errorf("inspection failed after %d attempts: %s", maxAttempts, result.Reasoning)
}
