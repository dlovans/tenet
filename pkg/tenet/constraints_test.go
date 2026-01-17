package tenet

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMinMaxValidation tests that numeric constraints are enforced.
func TestMinMaxValidation(t *testing.T) {
	t.Run("value below minimum triggers error", func(t *testing.T) {
		min := float64(1000)
		max := float64(100000)

		schema := &Schema{
			Definitions: map[string]*Definition{
				"loan_amount": {
					Type:  "number",
					Value: float64(500), // Below min
					Min:   &min,
					Max:   &max,
				},
			},
		}

		engine := NewEngine(schema)
		engine.validateDefinitions()

		if len(engine.errors) == 0 {
			t.Error("Expected error for value below minimum")
		}

		// Check error message
		found := false
		for _, err := range engine.errors {
			if err.FieldID == "loan_amount" && containsString(err.Message, "below minimum") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'below minimum' error, got: %v", engine.errors)
		}
	})

	t.Run("value above maximum triggers error", func(t *testing.T) {
		min := float64(1000)
		max := float64(100000)

		schema := &Schema{
			Definitions: map[string]*Definition{
				"loan_amount": {
					Type:  "number",
					Value: float64(150000), // Above max
					Min:   &min,
					Max:   &max,
				},
			},
		}

		engine := NewEngine(schema)
		engine.validateDefinitions()

		if len(engine.errors) == 0 {
			t.Error("Expected error for value above maximum")
		}

		found := false
		for _, err := range engine.errors {
			if err.FieldID == "loan_amount" && containsString(err.Message, "exceeds maximum") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'exceeds maximum' error, got: %v", engine.errors)
		}
	})

	t.Run("value within range passes validation", func(t *testing.T) {
		min := float64(1000)
		max := float64(100000)

		schema := &Schema{
			Definitions: map[string]*Definition{
				"loan_amount": {
					Type:  "number",
					Value: float64(50000), // Within range
					Min:   &min,
					Max:   &max,
				},
			},
		}

		engine := NewEngine(schema)
		engine.validateDefinitions()

		if len(engine.errors) != 0 {
			t.Errorf("Expected no errors, got: %v", engine.errors)
		}
	})
}

// TestStringLengthValidation tests string length constraints.
func TestStringLengthValidation(t *testing.T) {
	t.Run("string too short triggers error", func(t *testing.T) {
		minLen := 5
		maxLen := 100

		schema := &Schema{
			Definitions: map[string]*Definition{
				"name": {
					Type:      "string",
					Value:     "AB", // Too short
					MinLength: &minLen,
					MaxLength: &maxLen,
				},
			},
		}

		engine := NewEngine(schema)
		engine.validateDefinitions()

		if len(engine.errors) == 0 {
			t.Error("Expected error for string too short")
		}
	})

	t.Run("string too long triggers error", func(t *testing.T) {
		maxLen := 10

		schema := &Schema{
			Definitions: map[string]*Definition{
				"code": {
					Type:      "string",
					Value:     "THIS_IS_WAY_TOO_LONG",
					MaxLength: &maxLen,
				},
			},
		}

		engine := NewEngine(schema)
		engine.validateDefinitions()

		if len(engine.errors) == 0 {
			t.Error("Expected error for string too long")
		}
	})
}

// TestDynamicConstraints tests that rules can modify min/max via ui_modify.
func TestDynamicConstraints(t *testing.T) {
	t.Run("rule increases max based on tier", func(t *testing.T) {
		input := `{
			"protocol": "Test_v1",
			"schema_id": "test",
			"definitions": {
				"tier": {"type": "select", "value": "premium", "options": ["basic", "premium"]},
				"max_amount": {"type": "number", "value": 50000, "max": 10000}
			},
			"logic_tree": [
				{
					"id": "rule_premium_limit",
					"when": {"==": [{"var": "tier"}, "premium"]},
					"then": {
						"ui_modify": {
							"max_amount": {"max": 100000}
						}
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

		// Check that max was updated to 100000
		maxAmountDef := schema.Definitions["max_amount"]
		if maxAmountDef == nil {
			t.Fatal("max_amount definition not found")
		}
		if maxAmountDef.Max == nil || *maxAmountDef.Max != 100000 {
			t.Errorf("Expected max=100000, got %v", maxAmountDef.Max)
		}

		// Value 50000 should now be valid (under the new max of 100000)
		hasMaxError := false
		for _, err := range schema.Errors {
			if err.FieldID == "max_amount" && containsString(err.Message, "exceeds maximum") {
				hasMaxError = true
				break
			}
		}
		if hasMaxError {
			t.Error("Should not have max error after constraint was raised")
		}
	})
}
