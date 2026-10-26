// Package domain defines the core entities, value objects, and port interfaces
// for the SpecTest engine. It contains pure Go types with zero external dependencies.
package domain

// Suite is the root schema of `spectest.yaml` (schema version 1).
//
// `Version`, `Servers`, and `APIs` are derived from CLI input and OpenAPI files;
// the remaining four blocks form the mutable suite content that is hashed by the
// log and may be edited by users.
type Suite struct {
	Version   int              `yaml:"version" json:"version"`
	Input     InputMode        `yaml:"input" json:"input"`
	Servers   []string         `yaml:"servers" json:"servers"`
	APIs      []string         `yaml:"apis" json:"apis"`
	Analysis  string           `yaml:"analysis,omitempty" json:"analysis,omitempty"`
	Seed      map[string]Seed  `yaml:"seed,omitempty" json:"seed,omitempty"`
	Setup     map[string]Setup `yaml:"setup,omitempty" json:"setup,omitempty"`
	Scenarios []Scenario       `yaml:"scenarios" json:"scenarios"`
}

// Seed holds global default variables available to every setup and scenario.
type Seed struct {
	Vars map[string]any `yaml:"vars,omitempty" json:"vars,omitempty"`
}

// Setup is a reusable sequence of steps that prepares state for scenarios.
// Use lists other setups that must run first (a dependency set, not an order).
type Setup struct {
	Use   []string       `yaml:"use,omitempty" json:"use,omitempty"`
	Vars  map[string]any `yaml:"vars,omitempty" json:"vars,omitempty"`
	Steps []Step         `yaml:"steps" json:"steps"`
}

// Scenario is an executable business flow. ID is stable and kebab-case.
// Locked protects the scenario from AI changes; Skip carries a non-empty reason
// when the runner should not execute the scenario.
type Scenario struct {
	ID     string   `yaml:"id" json:"id"`
	Name   string   `yaml:"name" json:"name"`
	Setup  []string `yaml:"setup,omitempty" json:"setup,omitempty"`
	Steps  []Step   `yaml:"steps" json:"steps"`
	Locked bool     `yaml:"locked,omitempty" json:"locked,omitempty"`
	Skip   string   `yaml:"skip,omitempty" json:"skip,omitempty"`
}

// Step is a single HTTP call. Call contains the method, server placeholder,
// path, and optional inline query string. Save maps variable names to
// `body.<dot-path>` or `header.<name>` extraction expressions.
type Step struct {
	Call    string            `yaml:"call" json:"call"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body    any               `yaml:"body,omitempty" json:"body,omitempty"`
	Expect  int               `yaml:"expect" json:"expect"`
	Save    map[string]string `yaml:"save,omitempty" json:"save,omitempty"`
}

// MutableSuite is the subset of Suite that participates in the suite hash and
// may be changed by AI or users. Version, Servers, and APIs are derived fields.
type MutableSuite struct {
	Analysis  string           `yaml:"analysis,omitempty" json:"analysis,omitempty"`
	Seed      map[string]Seed  `yaml:"seed,omitempty" json:"seed,omitempty"`
	Setup     map[string]Setup `yaml:"setup,omitempty" json:"setup,omitempty"`
	Scenarios []Scenario       `yaml:"scenarios" json:"scenarios"`
}

// Mutable returns the four mutable blocks of the suite.
func (s *Suite) Mutable() MutableSuite {
	return MutableSuite{
		Analysis:  s.Analysis,
		Seed:      s.Seed,
		Setup:     s.Setup,
		Scenarios: s.Scenarios,
	}
}
