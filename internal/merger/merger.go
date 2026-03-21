package merger

import (
	"context"
	"fmt"
	"strings"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/job"
)

type Merger interface {
	Merge(ctx context.Context, task string, results []job.LineResult) (string, error)
}

type ConcatMerger struct {
	separator string
}

func NewConcatMerger(separator string) *ConcatMerger {
	return &ConcatMerger{separator: separator}
}

func (m *ConcatMerger) Merge(ctx context.Context, task string, results []job.LineResult) (string, error) {
	outputs := make([]string, 0, len(results))
	for _, r := range results {
		if r.Error == nil && r.Output != "" {
			outputs = append(outputs, fmt.Sprintf("=== %s ===\n%s", r.LineName, r.Output))
		}
	}
	return strings.Join(outputs, m.separator), nil
}

type FirstMerger struct{}

func NewFirstMerger() *FirstMerger {
	return &FirstMerger{}
}

func (m *FirstMerger) Merge(ctx context.Context, task string, results []job.LineResult) (string, error) {
	for _, r := range results {
		if r.Error == nil && r.Output != "" {
			return r.Output, nil
		}
	}
	return "", fmt.Errorf("no successful lines to merge")
}

type ClaudeMerger struct {
	agent  agents.Agent
	prompt string
}

func NewClaudeMerger(agent agents.Agent, prompt string) *ClaudeMerger {
	return &ClaudeMerger{
		agent:  agent,
		prompt: prompt,
	}
}

func (m *ClaudeMerger) Merge(ctx context.Context, task string, results []job.LineResult) (string, error) {
	var contextBuilder strings.Builder
	for i, r := range results {
		if r.Error == nil {
			contextBuilder.WriteString(fmt.Sprintf("Line %d (%s):\n%s\n\n", i, r.LineName, r.Output))
		} else {
			contextBuilder.WriteString(fmt.Sprintf("Line %d (%s): FAILED - %s\n\n", i, r.LineName, r.Error))
		}
	}

	prompt := strings.ReplaceAll(m.prompt, "{task}", task)
	prompt = strings.ReplaceAll(prompt, "{context}", contextBuilder.String())

	for i, r := range results {
		placeholder := fmt.Sprintf("{line_%d_output}", i)
		prompt = strings.ReplaceAll(prompt, placeholder, r.Output)
	}

	resp, err := m.agent.Run(ctx, agents.Request{
		SystemPrompt: "You are a merger agent. Combine multiple inputs into a coherent final output.",
		Task:         prompt,
	})
	if err != nil {
		return "", err
	}

	return resp.Output, nil
}

func NewMerger(cfg config.MergerConfig, agent agents.Agent) (Merger, error) {
	switch cfg.Type {
	case "concat":
		sep := cfg.Separator
		if sep == "" {
			sep = "\n\n---\n\n"
		}
		return NewConcatMerger(sep), nil
	case "first":
		return NewFirstMerger(), nil
	case "claude":
		if cfg.Prompt == "" {
			return nil, fmt.Errorf("claude merger requires a prompt")
		}
		return NewClaudeMerger(agent, cfg.Prompt), nil
	default:
		return nil, fmt.Errorf("unknown merger type: %s", cfg.Type)
	}
}
