package condition

import (
	"testing"

	"github.com/NikitaCOEUR/dirvana/internal/config"
)

func TestParse_NilWhen(t *testing.T) {
	_, err := Parse(nil)
	if err == nil {
		t.Error("Expected error for nil When, got nil")
	}
}

func TestParse_EmptyWhen(t *testing.T) {
	when := &config.When{}
	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for empty When, got nil")
	}
	if err.Error() != "when block must specify at least one condition" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestParse_SingleFileCondition(t *testing.T) {
	when := &config.When{
		File: "test.txt",
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return FileCondition
	fileCond, ok := cond.(FileCondition)
	if !ok {
		t.Fatalf("Expected FileCondition, got %T", cond)
	}

	if fileCond.Path != "test.txt" {
		t.Errorf("Expected path 'test.txt', got '%s'", fileCond.Path)
	}
}

func TestParse_SingleVarCondition(t *testing.T) {
	when := &config.When{
		Var: "MY_VAR",
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	varCond, ok := cond.(VarCondition)
	if !ok {
		t.Fatalf("Expected VarCondition, got %T", cond)
	}

	if varCond.Name != "MY_VAR" {
		t.Errorf("Expected name 'MY_VAR', got '%s'", varCond.Name)
	}
}

func TestParse_SingleDirCondition(t *testing.T) {
	when := &config.When{
		Dir: "mydir",
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	dirCond, ok := cond.(DirCondition)
	if !ok {
		t.Fatalf("Expected DirCondition, got %T", cond)
	}

	if dirCond.Path != "mydir" {
		t.Errorf("Expected path 'mydir', got '%s'", dirCond.Path)
	}
}

func TestParse_SingleCommandCondition(t *testing.T) {
	when := &config.When{
		Command: "docker",
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cmdCond, ok := cond.(CommandCondition)
	if !ok {
		t.Fatalf("Expected CommandCondition, got %T", cond)
	}

	if cmdCond.Name != "docker" {
		t.Errorf("Expected name 'docker', got '%s'", cmdCond.Name)
	}
}

func TestParse_MultipleAtomicConditions(t *testing.T) {
	// Multiple atomic conditions should be wrapped in AllCondition
	when := &config.When{
		File: "test.txt",
		Var:  "MY_VAR",
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	allCond, ok := cond.(AllCondition)
	if !ok {
		t.Fatalf("Expected AllCondition for multiple atomic conditions, got %T", cond)
	}

	if len(allCond.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(allCond.Conditions))
	}

	// Check that we have FileCondition and VarCondition
	hasFile := false
	hasVar := false
	for _, c := range allCond.Conditions {
		switch c.(type) {
		case FileCondition:
			hasFile = true
		case VarCondition:
			hasVar = true
		}
	}

	if !hasFile || !hasVar {
		t.Error("Expected FileCondition and VarCondition in AllCondition")
	}
}

func TestParse_AllConditions(t *testing.T) {
	when := &config.When{
		All: []config.When{
			{File: "test.txt"},
			{Var: "MY_VAR"},
		},
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	allCond, ok := cond.(AllCondition)
	if !ok {
		t.Fatalf("Expected AllCondition, got %T", cond)
	}

	if len(allCond.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(allCond.Conditions))
	}
}

func TestParse_AnyConditions(t *testing.T) {
	when := &config.When{
		Any: []config.When{
			{File: "test1.txt"},
			{File: "test2.txt"},
		},
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	anyCond, ok := cond.(AnyCondition)
	if !ok {
		t.Fatalf("Expected AnyCondition, got %T", cond)
	}

	if len(anyCond.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(anyCond.Conditions))
	}
}

func TestParse_NestedConditions(t *testing.T) {
	// Nested: all [ var, any [file1, file2] ]
	when := &config.When{
		All: []config.When{
			{Var: "MY_VAR"},
			{
				Any: []config.When{
					{File: "test1.txt"},
					{File: "test2.txt"},
				},
			},
		},
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	allCond, ok := cond.(AllCondition)
	if !ok {
		t.Fatalf("Expected AllCondition, got %T", cond)
	}

	if len(allCond.Conditions) != 2 {
		t.Errorf("Expected 2 conditions in AllCondition, got %d", len(allCond.Conditions))
	}

	// Second condition should be AnyCondition
	_, ok = allCond.Conditions[1].(AnyCondition)
	if !ok {
		t.Errorf("Expected second condition to be AnyCondition, got %T", allCond.Conditions[1])
	}
}

func TestParse_MixedAtomicAndComposite(t *testing.T) {
	// Cannot mix atomic and composite at the same level
	when := &config.When{
		File: "test.txt",
		All: []config.When{
			{Var: "MY_VAR"},
		},
	}

	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for mixing atomic and composite conditions, got nil")
	}
}

func TestParse_BothAllAndAny(t *testing.T) {
	// Cannot have both 'all' and 'any' at the same level
	when := &config.When{
		All: []config.When{
			{File: "test.txt"},
		},
		Any: []config.When{
			{Var: "MY_VAR"},
		},
	}

	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for having both 'all' and 'any', got nil")
	}
}

func TestParse_EmptyAllConditions(t *testing.T) {
	when := &config.When{
		All: []config.When{},
	}

	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for empty 'all' array, got nil")
	}
}

func TestParse_EmptyAnyConditions(t *testing.T) {
	when := &config.When{
		Any: []config.When{},
	}

	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for empty 'any' array, got nil")
	}
}

func TestParse_InvalidNestedCondition(t *testing.T) {
	// Invalid nested condition (empty When in All)
	when := &config.When{
		All: []config.When{
			{File: "test.txt"},
			{}, // Empty condition - invalid
		},
	}

	_, err := Parse(when)
	if err == nil {
		t.Error("Expected error for invalid nested condition, got nil")
	}
}

func TestParse_ComplexRealWorldExample(t *testing.T) {
	// Real-world example: KUBECONFIG exists and file exists
	when := &config.When{
		All: []config.When{
			{Var: "KUBECONFIG"},
			{File: "$KUBECONFIG"},
		},
	}

	cond, err := Parse(when)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	allCond, ok := cond.(AllCondition)
	if !ok {
		t.Fatalf("Expected AllCondition, got %T", cond)
	}

	if len(allCond.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(allCond.Conditions))
	}

	// Verify first is VarCondition
	varCond, ok := allCond.Conditions[0].(VarCondition)
	if !ok {
		t.Errorf("Expected first condition to be VarCondition, got %T", allCond.Conditions[0])
	} else if varCond.Name != "KUBECONFIG" {
		t.Errorf("Expected var name 'KUBECONFIG', got '%s'", varCond.Name)
	}

	// Verify second is FileCondition
	fileCond, ok := allCond.Conditions[1].(FileCondition)
	if !ok {
		t.Errorf("Expected second condition to be FileCondition, got %T", allCond.Conditions[1])
	} else if fileCond.Path != "$KUBECONFIG" {
		t.Errorf("Expected file path '$KUBECONFIG', got '%s'", fileCond.Path)
	}
}
