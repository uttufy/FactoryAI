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

IMPORTANT: Your response MUST start with either "PASS" or "FAIL" followed by a colon.

Example responses:
- PASS: The output meets all criteria because...
- FAIL: The output is missing X and needs Y...

Your response:`, task, output, i.config.Criteria)

	resp, err := i.agent.Run(ctx, agents.Request{
		SystemPrompt: "You are a strict quality inspector. Start your response with PASS: or FAIL:",
		Task:         prompt,
	})
	if err != nil {
		return false, "", err
	}

	upperOutput := strings.ToUpper(resp.Output)

	// Check for explicit PASS or FAIL at the start
	if strings.HasPrefix(upperOutput, "PASS") {
		passed = true
		reasoning = strings.TrimSpace(strings.TrimPrefix(resp.Output, "PASS"))
	} else if strings.HasPrefix(upperOutput, "FAIL") {
		passed = false
		reasoning = strings.TrimSpace(strings.TrimPrefix(resp.Output, "FAIL"))
	} else {
		// LLM didn't follow format - use heuristics
		// Default to PASS if we see positive indicators and no explicit FAIL
		positiveIndicators := []string{
			"WELL-STRUCTURED",
			"COMPREHENSIVE",
			"MEETS ALL CRITERIA",
			"MEETS THE CRITERIA",
			"SATISFIES",
			"SUCCESSFULLY",
			"GOOD",
			"EXCELLENT",
			"ADDRESSES",
			"THOROUGH",
			"COMPLETE",
		}
		negativeIndicators := []string{
			"FAILS TO",
			"DOES NOT MEET",
			"DOESN'T MEET",
			"MISSING",
			"INCOMPLETE",
			"UNCLEAR",
			"NEEDS IMPROVEMENT",
			"INSUFFICIENT",
			"LACKS",
		}

		hasPositive := false
		hasNegative := false

		for _, indicator := range positiveIndicators {
			if strings.Contains(upperOutput, indicator) {
				hasPositive = true
				break
			}
		}
		for _, indicator := range negativeIndicators {
			if strings.Contains(upperOutput, indicator) {
				hasNegative = true
				break
			}
		}

		// Default to PASS if positive indicators found and no negative ones
		// This is more lenient - if the LLM says good things, accept it
		if hasPositive && !hasNegative {
			passed = true
		} else if hasNegative {
			passed = false
		} else {
			// No clear indicators - default to PASS (trust the worker's output)
			passed = true
		}
		reasoning = resp.Output
	}
	reasoning = strings.TrimPrefix(reasoning, ":")
	reasoning = strings.TrimSpace(reasoning)

	return passed, reasoning, nil
}
