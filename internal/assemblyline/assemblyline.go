package assemblyline

import (
	"context"
	"time"

	"github.com/uttufy/FactoryAI/internal/agents"
	"github.com/uttufy/FactoryAI/internal/config"
	"github.com/uttufy/FactoryAI/internal/events"
	"github.com/uttufy/FactoryAI/internal/job"
)

type AssemblyLine struct {
	config   config.AssemblyLineConfig
	stations []*StationV0
}

func New(cfg config.AssemblyLineConfig, agent agents.Agent) *AssemblyLine {
	stations := make([]*StationV0, len(cfg.Stations))
	for i, stationCfg := range cfg.Stations {
		stations[i] = NewV0Station(stationCfg, agent, cfg.Name)
	}

	return &AssemblyLine{
		config:   cfg,
		stations: stations,
	}
}

func (al *AssemblyLine) Run(ctx context.Context, task string, eventsChan chan<- events.Event) (job.LineResult, error) {
	start := time.Now()

	result := job.LineResult{
		LineName: al.config.Name,
		Stations: make([]job.StationResult, 0, len(al.stations)),
	}

	var context string

	for _, s := range al.stations {
		stationResult, err := s.Run(ctx, task, context, eventsChan)
		result.Stations = append(result.Stations, stationResult)

		if err != nil {
			result.Error = err
			result.Duration = time.Since(start)
			return result, err
		}

		context = stationResult.Output
	}

	result.Output = context
	result.Duration = time.Since(start)

	return result, nil
}

func (al *AssemblyLine) Name() string {
	return al.config.Name
}

func (al *AssemblyLine) StationNames() []string {
	names := make([]string, len(al.stations))
	for i, s := range al.stations {
		names[i] = s.Name()
	}
	return names
}
