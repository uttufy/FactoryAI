package agents

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

	// Create pipes for stdout/stderr to stream in real-time while capturing
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Response{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Response{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return Response{}, fmt.Errorf("failed to start claude command: %w", err)
	}

	// Stream output in real-time while capturing
	var stdoutBuf, stderrBuf bytes.Buffer
	done := make(chan struct{})

	go func() {
		defer close(done)
		// Stream stdout
		go func() {
			multiWriter := io.MultiWriter(&stdoutBuf, os.Stdout)
			io.Copy(multiWriter, stdoutPipe)
		}()
		// Stream stderr
		multiWriter := io.MultiWriter(&stderrBuf, os.Stderr)
		io.Copy(multiWriter, stderrPipe)
	}()

	// Wait for command to complete
	err = cmd.Wait()
	<-done

	duration := time.Since(start)

	output := stdoutBuf.String()
	if stderrBuf.Len() > 0 && output == "" {
		output = stderrBuf.String()
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
