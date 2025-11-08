package condition

import (
	"fmt"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

// Parse converts a config.When struct into a Condition interface
// Returns an error if the When struct is invalid (e.g., no conditions specified)
func Parse(when *config.When) (Condition, error) {
	if when == nil {
		return nil, fmt.Errorf("when is nil")
	}

	atomicCount := countAtomicConditions(when)
	compositeCount := countCompositeConditions(when)

	// Validate
	if err := validateConditions(atomicCount, compositeCount, when); err != nil {
		return nil, err
	}

	// Parse composite conditions first
	if len(when.All) > 0 {
		return parseAll(when.All)
	}
	if len(when.Any) > 0 {
		return parseAny(when.Any)
	}

	// Parse atomic conditions
	return parseAtomicConditions(when, atomicCount)
}

// countAtomicConditions counts the number of atomic conditions in a When struct
func countAtomicConditions(when *config.When) int {
	count := 0
	if when.File != "" {
		count++
	}
	if when.Var != "" {
		count++
	}
	if when.Dir != "" {
		count++
	}
	if when.Command != "" {
		count++
	}
	return count
}

// countCompositeConditions counts the number of composite conditions in a When struct
func countCompositeConditions(when *config.When) int {
	count := 0
	if len(when.All) > 0 {
		count++
	}
	if len(when.Any) > 0 {
		count++
	}
	return count
}

// validateConditions validates the condition structure
func validateConditions(atomicCount, compositeCount int, when *config.When) error {
	// Must have at least one condition
	if atomicCount == 0 && compositeCount == 0 {
		return fmt.Errorf("when block must specify at least one condition")
	}

	// Cannot mix atomic and composite conditions at the same level
	if atomicCount > 0 && compositeCount > 0 {
		return fmt.Errorf("cannot mix atomic conditions (file, var, dir, command) with composite conditions (all, any) at the same level")
	}

	// Cannot have both 'all' and 'any' at the same level
	if len(when.All) > 0 && len(when.Any) > 0 {
		return fmt.Errorf("cannot have both 'all' and 'any' at the same level")
	}

	return nil
}

// parseAtomicConditions parses atomic conditions from a When struct
func parseAtomicConditions(when *config.When, atomicCount int) (Condition, error) {
	// If multiple atomic conditions, wrap them in AllCondition
	if atomicCount > 1 {
		conditions := collectAtomicConditions(when)
		return AllCondition{Conditions: conditions}, nil
	}

	// Single atomic condition
	if when.File != "" {
		return FileCondition{Path: when.File}, nil
	}
	if when.Var != "" {
		return VarCondition{Name: when.Var}, nil
	}
	if when.Dir != "" {
		return DirCondition{Path: when.Dir}, nil
	}
	if when.Command != "" {
		return CommandCondition{Name: when.Command}, nil
	}

	// Should never reach here due to earlier validation
	return nil, fmt.Errorf("no valid condition found")
}

// collectAtomicConditions collects all atomic conditions into a slice
func collectAtomicConditions(when *config.When) []Condition {
	var conditions []Condition

	if when.File != "" {
		conditions = append(conditions, FileCondition{Path: when.File})
	}
	if when.Var != "" {
		conditions = append(conditions, VarCondition{Name: when.Var})
	}
	if when.Dir != "" {
		conditions = append(conditions, DirCondition{Path: when.Dir})
	}
	if when.Command != "" {
		conditions = append(conditions, CommandCondition{Name: when.Command})
	}

	return conditions
}

// parseAll parses an array of When structs into an AllCondition
func parseAll(whens []config.When) (Condition, error) {
	if len(whens) == 0 {
		return nil, fmt.Errorf("all: must contain at least one condition")
	}

	var conditions []Condition

	for i, when := range whens {
		cond, err := Parse(&when)
		if err != nil {
			return nil, fmt.Errorf("all[%d]: %w", i, err)
		}
		conditions = append(conditions, cond)
	}

	return AllCondition{Conditions: conditions}, nil
}

// parseAny parses an array of When structs into an AnyCondition
func parseAny(whens []config.When) (Condition, error) {
	if len(whens) == 0 {
		return nil, fmt.Errorf("any: must contain at least one condition")
	}

	var conditions []Condition

	for i, when := range whens {
		cond, err := Parse(&when)
		if err != nil {
			return nil, fmt.Errorf("any[%d]: %w", i, err)
		}
		conditions = append(conditions, cond)
	}

	return AnyCondition{Conditions: conditions}, nil
}
