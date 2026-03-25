// Package workflow implements TOML-based Formula parsing for FactoryAI.
// Formulas are recipes that cook into Protomolecules (SOP templates).
package workflow

import (
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/uttufy/FactoryAI/internal/beads"
)

// FormulaStep defines a step in a TOML formula
type FormulaStep struct {
	Name         string            `toml:"name"`
	Description  string            `toml:"description,omitempty"`
	Assignee     string            `toml:"assignee,omitempty"`
	Dependencies []string          `toml:"dependencies,omitempty"`
	Acceptance   string            `toml:"acceptance,omitempty"`
	Gate         string            `toml:"gate,omitempty"`
	Timeout      int               `toml:"timeout,omitempty"`
	MaxRetries   int               `toml:"max_retries,omitempty"`
	Variables    map[string]string `toml:"variables,omitempty"`
}

// Formula is a TOML recipe that cooks into a Protomolecule
type Formula struct {
	Name        string            `toml:"name"`
	Description string            `toml:"description,omitempty"`
	Variables   map[string]string `toml:"variables,omitempty"`
	Steps       []FormulaStep     `toml:"steps"`
}

// LoadFormula loads a formula from a TOML file
func LoadFormula(path string) (*Formula, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading formula file: %w", err)
	}

	return ParseFormula(string(data))
}

// ParseFormula parses a formula from string
func ParseFormula(data string) (*Formula, error) {
	var formula Formula
	if err := toml.Unmarshal([]byte(data), &formula); err != nil {
		return nil, fmt.Errorf("parsing TOML: %w", err)
	}

	return &formula, nil
}

// Cook converts a formula to a protomolecule
func (f *Formula) Cook() (*Protomolecule, error) {
	return f.CookWithVars(nil)
}

// CookWithVars converts a formula to a protomolecule with variable substitution
func (f *Formula) CookWithVars(vars map[string]string) (*Protomolecule, error) {
	// Merge formula variables with provided variables
	mergedVars := make(map[string]string)
	for k, v := range f.Variables {
		mergedVars[k] = v
	}
	for k, v := range vars {
		mergedVars[k] = v
	}

	// Convert formula steps to workflow steps
	steps := make([]*beads.Step, len(f.Steps))
	for i, fs := range f.Steps {
		step := &beads.Step{
			ID:           "", // Will be assigned by DAGEngine
			Name:         fs.Name,
			Description:  fs.Description,
			Assignee:     fs.Assignee,
			Dependencies: fs.Dependencies,
			Acceptance:   fs.Acceptance,
			Gate:         fs.Gate,
			Timeout:      fs.Timeout,
			MaxRetries:   fs.MaxRetries,
		}

		// Substitute variables in description
		step.Description = substituteVariables(step.Description, mergedVars)

		steps[i] = step
	}

	protomolecule := &Protomolecule{
		ID:          "", // Will be assigned when saved
		Name:        f.Name,
		Description: substituteVariables(f.Description, mergedVars),
		Steps:       steps,
		CreatedAt:   time.Now(),
	}

	return protomolecule, nil
}

// substituteVariables replaces {var} placeholders with values
func substituteVariables(text string, vars map[string]string) string {
	if text == "" {
		return text
	}

	result := text
	for key, value := range vars {
		placeholder := fmt.Sprintf("{%s}", key)
		result = replaceAll(result, placeholder, value)
	}

	return result
}

// replaceAll replaces all occurrences of old with new in s
func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}

	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

// indexOf finds the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SaveFormula saves a formula to a TOML file
func SaveFormula(formula *Formula, path string) error {
	data, err := toml.Marshal(formula)
	if err != nil {
		return fmt.Errorf("marshaling TOML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing formula file: %w", err)
	}

	return nil
}

// Protomolecule is a reusable SOP template
type Protomolecule struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Steps       []*beads.Step `json:"steps"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Instantiate creates a new SOP from this template
func (p *Protomolecule) Instantiate(vars map[string]string) (*SOP, error) {
	return p.InstantiateAsWisp(vars, false)
}

// InstantiateAsWisp creates an ephemeral SOP from this template
func (p *Protomolecule) InstantiateAsWisp(vars map[string]string, asWisp bool) (*SOP, error) {
	// Create a copy of steps with variable substitution
	steps := make([]*Step, len(p.Steps))
	for i, ps := range p.Steps {
		step := &Step{
			ID:           "", // Will be assigned by DAGEngine
			Name:         ps.Name,
			Description:  substituteVariables(ps.Description, vars),
			Assignee:     ps.Assignee,
			Dependencies: ps.Dependencies,
			Acceptance:   substituteVariables(ps.Acceptance, vars),
			Gate:         ps.Gate,
			Timeout:      ps.Timeout,
			MaxRetries:   ps.MaxRetries,
		}
		steps[i] = step
	}

	sop := &SOP{
		ID:          "", // Will be assigned by DAGEngine
		Name:        substituteVariables(p.Name, vars),
		Description: substituteVariables(p.Description, vars),
		Steps:       steps,
		Status:      SOPPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsWisp:      asWisp,
	}

	return sop, nil
}

// Validate validates a formula for correctness
func (f *Formula) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("formula must have a name")
	}

	if len(f.Steps) == 0 {
		return fmt.Errorf("formula must have at least one step")
	}

	stepNames := make(map[string]bool)
	for i, step := range f.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: must have a name", i)
		}

		if stepNames[step.Name] {
			return fmt.Errorf("step %d: duplicate step name '%s'", i, step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies exist
		for _, dep := range step.Dependencies {
			if !stepNames[dep] {
				return fmt.Errorf("step %d: dependency '%s' not found (must be defined before this step)", i, dep)
			}
		}
	}

	return nil
}
