package job

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID            string
	Task          string
	BlueprintName string
	CreatedAt     time.Time
	StartedAt     time.Time
}

func NewJob(task, blueprintName string) *Job {
	return &Job{
		ID:            uuid.New().String()[:8],
		Task:          task,
		BlueprintName: blueprintName,
		CreatedAt:     time.Now(),
	}
}

type StationResult struct {
	StationName  string
	Output       string
	Duration     time.Duration
	Passed       bool
	RetriesUsed  int
	Error        error
	Reasoning    string
}

type LineResult struct {
	LineName string
	Stations []StationResult
	Output   string
	Error    error
	Duration time.Duration
}

type JobResult struct {
	JobID         string
	LineResults   []LineResult
	FinalOutput   string
	TotalDuration time.Duration
	Error         error
}
