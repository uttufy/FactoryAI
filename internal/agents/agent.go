package agents

import "context"

type Request struct {
	SystemPrompt string
	Task         string
	Context      string
}

type Response struct {
	Output      string
	AgentName   string
	DurationSec float64
}

type Agent interface {
	Name() string
	Run(ctx context.Context, req Request) (Response, error)
}
