package factory

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/assemblyline"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/job"
	"github.com/uttufy/FactoryAI/internal/merger"
)

type Factory struct {
	blueprint *config.Blueprint
	agent     agents.Agent
	lines     []*assemblyline.AssemblyLine
	merger    merger.Merger
}

func New(blueprint *config.Blueprint, agent agents.Agent) (*Factory, error) {
	lines := make([]*assemblyline.AssemblyLine, len(blueprint.Factory.AssemblyLines))
	for i, lineCfg := range blueprint.Factory.AssemblyLines {
		lines[i] = assemblyline.New(lineCfg, agent)
	}

	var m merger.Merger
	var err error
	if len(lines) == 1 {
		m = merger.NewFirstMerger()
	} else {
		m, err = merger.NewMerger(blueprint.Factory.Merger, agent)
		if err != nil {
			return nil, err
		}
	}

	return &Factory{
		blueprint: blueprint,
		agent:     agent,
		lines:     lines,
		merger:    m,
	}, nil
}

func (f *Factory) Run(ctx context.Context, task string, eventsChan chan<- events.Event) (*job.JobResult, error) {
	start := time.Now()

	j := job.NewJob(task, f.blueprint.Factory.Name)

	results := make([]job.LineResult, len(f.lines))

	g, ctx := errgroup.WithContext(ctx)

	for i, line := range f.lines {
		i, line := i, line
		g.Go(func() error {
			result, err := line.Run(ctx, task, eventsChan)
			results[i] = result
			return err
		})
	}

	if err := g.Wait(); err != nil {
		eventsChan <- events.Done("")
		return &job.JobResult{
			JobID:         j.ID,
			LineResults:   results,
			TotalDuration: time.Since(start),
			Error:         err,
		}, err
	}

	eventsChan <- events.Merging()

	merged, err := f.merger.Merge(ctx, task, results)
	if err != nil {
		eventsChan <- events.Done("")
		return &job.JobResult{
			JobID:         j.ID,
			LineResults:   results,
			TotalDuration: time.Since(start),
			Error:         err,
		}, err
	}

	eventsChan <- events.Done(merged)

	return &job.JobResult{
		JobID:         j.ID,
		LineResults:   results,
		FinalOutput:   merged,
		TotalDuration: time.Since(start),
	}, nil
}

func (f *Factory) Blueprint() *config.Blueprint {
	return f.blueprint
}
