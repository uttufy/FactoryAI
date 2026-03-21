package agents

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ClaudeAgent struct {
	binaryPath string
}

func NewClaudeAgent(binaryPath string) (*ClaudeAgent, error) {
	if binaryPath == "" {
		binaryPath = "claude"
	}

	_, err := exec.LookPath(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("claude binary not found in PATH: %w", err)
	}

	return &ClaudeAgent{binaryPath: binaryPath}, nil
}

func (a *ClaudeAgent) Name() string {
	return "claude"
}

func (a *ClaudeAgent) Run(ctx context.Context, req Request) (Response, error) {
	start := time.Now()

	prompt := a.buildPrompt(req)

	cmd := exec.CommandContext(ctx, a.binaryPath, "-p", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	duration := time.Since(start)

	output := stdout.String()
	if stderr.Len() > 0 && output == "" {
		output = stderr.String()
	}

	if err != nil {
		return Response{}, fmt.Errorf("claude command failed: %w\nOutput: %s", err, output)
	}

	return Response{
		Output:      strings.TrimSpace(output),
		AgentName:   a.Name(),
		DurationSec: duration.Seconds(),
	}, nil
}

func (a *ClaudeAgent) buildPrompt(req Request) string {
	var b strings.Builder

	if req.SystemPrompt != "" {
		b.WriteString(req.SystemPrompt)
		b.WriteString("\n\n")
	}

	if req.Task != "" {
		b.WriteString("Task:\n")
		b.WriteString(req.Task)
		b.WriteString("\n\n")
	}

	if req.Context != "" {
		b.WriteString("Context:\n")
		b.WriteString(req.Context)
	}

	return b.String()
}

func GetBinaryPath() string {
	return os.Getenv("CLAUDE_BIN")
}
