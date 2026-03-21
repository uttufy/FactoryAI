package inspector

import (
	"context"
	"fmt"
	"strings"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
)

type Inspector struct {
	agent  agents.Agent
	config config.InspectorConfig
}

func New(agent agents.Agent, cfg config.InspectorConfig) *Inspector {
	return &Inspector{
		agent:  agent,
		config: cfg,
	}
}

func (i *Inspector) Inspect(ctx context.Context, task, output string) (passed bool, reasoning string, err error) {
	prompt := fmt.Sprintf(`You are a quality inspector. Evaluate whether the following output meets the criteria.

Original Task: %s

Output to Evaluate:
%s

Criteria: %s

Respond with either:
- PASS: [brief reasoning]
- FAIL: [brief reasoning explaining what needs to be fixed]

Your response:`, task, output, i.config.Criteria)

	resp, err := i.agent.Run(ctx, agents.Request{
		SystemPrompt: "You are a strict quality inspector. Be thorough but fair.",
		Task:         prompt,
	})
	if err != nil {
		return false, "", err
	}

	firstWord := strings.ToUpper(strings.Fields(resp.Output)[0])
	passed = firstWord == "PASS"

	reasoning = strings.TrimSpace(strings.TrimPrefix(resp.Output, firstWord))
	reasoning = strings.TrimPrefix(reasoning, ":")
	reasoning = strings.TrimSpace(reasoning)

	return passed, reasoning, nil
}
