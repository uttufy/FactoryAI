package worker

import (
	"context"
	"strings"
	"time"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/job"
)

type Worker struct {
	config config.StationConfig
	agent  agents.Agent
}

func New(cfg config.StationConfig, agent agents.Agent) *Worker {
	return &Worker{
		config: cfg,
		agent:  agent,
	}
}

func (w *Worker) Run(ctx context.Context, task, context string) job.StationResult {
	start := time.Now()

	prompt := w.renderPrompt(task, context)

	resp, err := w.agent.Run(ctx, agents.Request{
		SystemPrompt: w.buildSystemPrompt(),
		Task:         prompt,
	})

	result := job.StationResult{
		StationName: w.config.Name,
		Duration:    time.Since(start),
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Output = resp.Output
	return result
}

func (w *Worker) renderPrompt(task, context string) string {
	prompt := w.config.Prompt

	prompt = strings.ReplaceAll(prompt, "{task}", task)
	prompt = strings.ReplaceAll(prompt, "{context}", context)
	prompt = strings.ReplaceAll(prompt, "{role}", w.config.Role)

	return prompt
}

func (w *Worker) buildSystemPrompt() string {
	if w.config.Role != "" {
		return "You are a " + w.config.Role + "."
	}
	return ""
}
