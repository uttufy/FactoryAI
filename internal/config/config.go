package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Blueprint struct {
	Factory FactoryConfig `yaml:"factory"`
}

type FactoryConfig struct {
	Name          string               `yaml:"name"`
	Description   string               `yaml:"description"`
	AssemblyLines []AssemblyLineConfig `yaml:"assembly_lines"`
	Merger        MergerConfig         `yaml:"merger"`
}

type AssemblyLineConfig struct {
	Name     string          `yaml:"name"`
	Stations []StationConfig `yaml:"stations"`
}

type StationConfig struct {
	Name      string           `yaml:"name"`
	Role      string           `yaml:"role"`
	Prompt    string           `yaml:"prompt"`
	Inspector *InspectorConfig `yaml:"inspector"`
}

type InspectorConfig struct {
	Enabled   bool   `yaml:"enabled"`
	MaxRetries int   `yaml:"max_retries"`
	Criteria  string `yaml:"criteria"`
}

type MergerConfig struct {
	Type      string `yaml:"type"`
	Separator string `yaml:"separator"`
	Prompt    string `yaml:"prompt"`
}

func LoadBlueprint(path string) (*Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bp Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}
