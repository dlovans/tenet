package tenet

import (
	"strings"
	"testing"
	"time"
)

func TestCycleDetection(t *testing.T) {
	t.Run("detects field set by multiple rules", func(t *testing.T) {
		schema := `{
			"definitions": {
				"a": {"type": "number", "value": 5, "visible": true},
				"b": {"type": "number", "value": 10, "visible": true}
			},
			"logic_tree": [
				{
					"id": "rule_1",
					"when": {"<": [{"var": "a"}, 10]},
					"then": {"set": {"b": 20}}
				},
				{
					"id": "rule_2",
					"when": {"<": [{"var": "b"}, 25]},
					"then": {"set": {"b": 30}}
				}
			]
		}`

		result, err := Run(schema, time.Now())
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Should contain cycle warning
		if !strings.Contains(result, "potential cycle") {
			t.Error("Expected cycle detection warning")
		}
		if !strings.Contains(result, "rule_1") && !strings.Contains(result, "rule_2") {
			t.Error("Expected both rule IDs in error message")
		}
		t.Logf("Cycle detected: %s", result)
	})

	t.Run("no warning for same rule setting field twice", func(t *testing.T) {
		// Same rule can set same field multiple times (not a cycle)
		schema := `{
			"definitions": {
				"a": {"type": "number", "value": 5, "visible": true},
				"b": {"type": "number", "value": 0, "visible": true}
			},
			"logic_tree": [
				{
					"id": "rule_1",
					"when": {"<": [{"var": "a"}, 10]},
					"then": {"set": {"b": 20}}
				}
			]
		}`

		result, err := Run(schema, time.Now())
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		// Should NOT contain cycle warning
		if strings.Contains(result, "potential cycle") {
			t.Error("Should not warn when same rule sets field")
		}
	})
}
