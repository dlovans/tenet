package tenet

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResolveVar(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"name": {Type: "string", Value: "John"},
			"age":  {Type: "number", Value: float64(30)},
		},
	}

	engine := NewEngine(schema)

	tests := []struct {
		path     string
		expected any
	}{
		{"name", "John"},
		{"age", float64(30)},
		{"unknown", nil},
	}

	for _, tt := range tests {
		result := engine.getVar(tt.path)
		if result != tt.expected {
			t.Errorf("getVar(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestResolveVarNested(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"user": {
				Type: "object",
				Value: map[string]any{
					"address": map[string]any{
						"city": "Stockholm",
					},
				},
			},
		},
	}

	engine := NewEngine(schema)

	result := engine.getVar("user.address.city")
	if result != "Stockholm" {
		t.Errorf("getVar(user.address.city) = %v, want Stockholm", result)
	}

	// Test missing path
	result = engine.getVar("user.address.country")
	if result != nil {
		t.Errorf("getVar(user.address.country) = %v, want nil", result)
	}
}

func TestOperatorEquality(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"status": {Type: "string", Value: "active"},
		},
	}

	engine := NewEngine(schema)

	// Test equality with var lookup
	expr := map[string]any{
		"==": []any{
			map[string]any{"var": "status"},
			"active",
		},
	}

	result := engine.resolve(expr)
	if result != true {
		t.Errorf("resolve(status == active) = %v, want true", result)
	}

	// Test inequality
	expr = map[string]any{
		"!=": []any{
			map[string]any{"var": "status"},
			"inactive",
		},
	}

	result = engine.resolve(expr)
	if result != true {
		t.Errorf("resolve(status != inactive) = %v, want true", result)
	}
}

func TestOperatorArithmetic(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"price":    {Type: "number", Value: float64(100)},
			"quantity": {Type: "number", Value: float64(5)},
		},
	}

	engine := NewEngine(schema)

	// Test multiplication
	expr := map[string]any{
		"*": []any{
			map[string]any{"var": "price"},
			map[string]any{"var": "quantity"},
		},
	}

	result := engine.resolve(expr)
	if result != float64(500) {
		t.Errorf("resolve(price * quantity) = %v, want 500", result)
	}
}

func TestOperatorNilSafe(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"revenue": {Type: "number", Value: nil}, // Nil value
		},
	}

	engine := NewEngine(schema)

	// Comparison with nil should return false, not panic
	expr := map[string]any{
		">": []any{
			map[string]any{"var": "revenue"},
			float64(1000),
		},
	}

	result := engine.resolve(expr)
	if result != false {
		t.Errorf("resolve(nil > 1000) = %v, want false", result)
	}
}

func TestOperatorIf(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"location": {Type: "string", Value: "SE"},
		},
	}

	engine := NewEngine(schema)

	// Test if-then-else
	expr := map[string]any{
		"if": []any{
			map[string]any{
				"==": []any{
					map[string]any{"var": "location"},
					"SE",
				},
			},
			0.25,
			0.20,
		},
	}

	result := engine.resolve(expr)
	if result != 0.25 {
		t.Errorf("resolve(if location == SE then 0.25 else 0.20) = %v, want 0.25", result)
	}
}

func TestOperatorDate(t *testing.T) {
	schema := &Schema{
		Definitions: map[string]*Definition{
			"deadline": {Type: "date", Value: "2025-12-31"},
			"today":    {Type: "date", Value: "2025-01-15"},
		},
	}

	engine := NewEngine(schema)

	// Test before
	expr := map[string]any{
		"before": []any{
			map[string]any{"var": "today"},
			map[string]any{"var": "deadline"},
		},
	}

	result := engine.resolve(expr)
	if result != true {
		t.Errorf("resolve(today before deadline) = %v, want true", result)
	}
}

func TestRunBasic(t *testing.T) {
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"level": {"type": "select", "value": "High", "options": ["Low", "High"]}
		},
		"logic_tree": [
			{
				"id": "rule_1",
				"when": {"==": [{"var": "level"}, "High"]},
				"then": {
					"set": {"alert": true}
				}
			}
		]
	}`

	result, err := Run(input, time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var schema Schema
	if err := json.Unmarshal([]byte(result), &schema); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check that alert was set
	alertDef, ok := schema.Definitions["alert"]
	if !ok {
		t.Fatal("Expected 'alert' definition to be created")
	}
	if alertDef.Value != true {
		t.Errorf("Expected alert = true, got %v", alertDef.Value)
	}

	// Check status is READY (no required fields missing)
	if schema.Status != StatusReady {
		t.Errorf("Expected status READY, got %v", schema.Status)
	}
}

func TestRunWithMissingRequired(t *testing.T) {
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"email": {"type": "string", "required": true}
		}
	}`

	result, err := Run(input, time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var schema Schema
	if err := json.Unmarshal([]byte(result), &schema); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check status is INCOMPLETE (required field missing)
	if schema.Status != StatusIncomplete {
		t.Errorf("Expected status INCOMPLETE, got %v", schema.Status)
	}

	// Check there's an error for the missing field
	if len(schema.Errors) == 0 {
		t.Error("Expected errors for missing required field")
	}
}

func TestVerify(t *testing.T) {
	oldJson := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"level": {"type": "select", "value": "High", "options": ["Low", "High"]}
		},
		"logic_tree": [
			{
				"id": "rule_1",
				"when": {"==": [{"var": "level"}, "High"]},
				"then": {"set": {"alert": true}}
			}
		]
	}`

	// Run the old JSON to get the expected new state
	newJson, err := Run(oldJson, time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify should pass
	valid, err := Verify(newJson, oldJson)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Error("Expected verification to pass")
	}
}
