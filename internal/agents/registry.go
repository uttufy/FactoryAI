package agents

import "fmt"

func NewAgent(agentType, binaryPath string) (Agent, error) {
	switch agentType {
	case "claude":
		return NewClaudeAgent(binaryPath)
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}
