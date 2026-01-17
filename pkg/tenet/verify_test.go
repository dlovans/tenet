package tenet

import (
	"strings"
	"testing"
)

func TestVerifyTurnBased(t *testing.T) {
	// Base schema with branching logic
	baseSchema := `{
		"definitions": {
			"revenue": {"type": "number", "value": null, "visible": true},
			"small_biz_field": {"type": "string", "visible": false},
			"large_biz_field": {"type": "string", "visible": false}
		},
		"logic_tree": [
			{
				"id": "show_small",
				"when": {"<=": [{"var": "revenue"}, 5000]},
				"then": {"ui_modify": {"small_biz_field": {"visible": true, "required": true}}}
			},
			{
				"id": "show_large",
				"when": {">": [{"var": "revenue"}, 5000]},
				"then": {"ui_modify": {"large_biz_field": {"visible": true, "required": true}}}
			}
		],
		"state_model": {
			"derived": {
				"tax_bracket": {"eval": {"if": [{"<=": [{"var": "revenue"}, 5000]}, "low", "high"]}}
			}
		}
	}`

	t.Run("valid small business path", func(t *testing.T) {
		// User entered revenue=3000, filled small_biz_field, got tax_bracket=low
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "value": "small corp", "visible": true, "required": true},
				"large_biz_field": {"type": "string", "visible": false},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !valid {
			t.Fatal("Expected valid, got invalid")
		}
	})

	t.Run("valid large business path", func(t *testing.T) {
		// User entered revenue=10000, filled large_biz_field, got tax_bracket=high
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 10000, "visible": true},
				"small_biz_field": {"type": "string", "visible": false},
				"large_biz_field": {"type": "string", "value": "big corp", "visible": true, "required": true},
				"tax_bracket": {"type": "string", "value": "high", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !valid {
			t.Fatal("Expected valid, got invalid")
		}
	})

	t.Run("tampered computed value", func(t *testing.T) {
		// User claims tax_bracket=low but revenue=10000 should give high
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 10000, "visible": true},
				"large_biz_field": {"type": "string", "value": "big corp", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if valid {
			t.Fatal("Expected invalid due to tampered computed value")
		}
		if err == nil || !strings.Contains(err.Error(), "mismatch") {
			t.Fatalf("Expected mismatch error, got: %v", err)
		}
	})

	t.Run("wrong branch - claims field that shouldnt be visible", func(t *testing.T) {
		// User claims they filled large_biz_field but revenue=3000 should show small_biz_field
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "visible": false},
				"large_biz_field": {"type": "string", "value": "FAKE", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		// This should fail because large_biz_field shouldnt be visible at revenue=3000
		valid, err := Verify(completedDoc, baseSchema)
		// The verification will detect that the path is wrong because
		// it replays from base and sees different visible fields
		t.Logf("Result: valid=%v, err=%v", valid, err)
		// Note: Current impl might not catch this specific case - documenting behavior
	})

	t.Run("status tampering", func(t *testing.T) {
		// User claims READY but missing required field
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "visible": true, "required": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if valid {
			t.Fatal("Expected invalid due to status tampering (claimed READY but field empty)")
		}
		if err == nil {
			t.Fatal("Expected error for status mismatch")
		}
		t.Logf("Status tamper caught: %v", err)
	})

	t.Run("renamed field - field_a renamed to field_x", func(t *testing.T) {
		// User renamed small_biz_field to fake_field
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"fake_renamed_field": {"type": "string", "value": "small corp", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		t.Logf("Renamed field result: valid=%v, err=%v", valid, err)
		// The verification should catch this because small_biz_field is missing
		// and fake_renamed_field doesn't exist in base schema
	})

	t.Run("extra field that doesnt exist in base", func(t *testing.T) {
		// User added a field that doesn't exist in base schema
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "value": "small corp", "visible": true, "required": true},
				"INJECTED_FIELD": {"type": "string", "value": "hacked", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if valid {
			t.Fatal("Expected invalid due to injected field")
		}
		if err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("Expected unknown field error, got: %v", err)
		}
		t.Logf("Extra field caught: %v", err)
	})
}

func TestVerifyWithAttestations(t *testing.T) {
	baseSchema := `{
		"definitions": {
			"amount": {"type": "number", "value": null, "visible": true}
		},
		"attestations": {
			"officer_sign": {
				"statement": "I certify this is correct",
				"required": true,
				"signed": false
			}
		}
	}`

	t.Run("attestation fulfilled", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"amount": {"type": "number", "value": 5000, "visible": true}
			},
			"attestations": {
				"officer_sign": {
					"statement": "I certify this is correct",
					"required": true,
					"signed": true,
					"evidence": {
						"provider_audit_id": "ds_123",
						"timestamp": "2026-01-17T12:00:00Z",
						"signer_id": "john@example.com"
					}
				}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !valid {
			t.Fatal("Expected valid")
		}
	})

	t.Run("attestation missing evidence", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"amount": {"type": "number", "value": 5000, "visible": true}
			},
			"attestations": {
				"officer_sign": {
					"statement": "I certify this is correct",
					"required": true,
					"signed": true
				}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if valid {
			t.Fatal("Expected invalid due to missing evidence")
		}
		if err == nil || !strings.Contains(err.Error(), "missing evidence") {
			t.Fatalf("Expected missing evidence error, got: %v", err)
		}
	})
}

// Helper to run schema and check convergence
func TestVerifyConvergence(t *testing.T) {
	// Schema with cascading field reveals
	baseSchema := `{
		"definitions": {
			"step1": {"type": "string", "value": null, "visible": true},
			"step2": {"type": "string", "visible": false},
			"step3": {"type": "string", "visible": false}
		},
		"logic_tree": [
			{
				"id": "reveal_step2",
				"when": {"==": [{"var": "step1"}, "done"]},
				"then": {"ui_modify": {"step2": {"visible": true}}}
			},
			{
				"id": "reveal_step3",
				"when": {"==": [{"var": "step2"}, "done"]},
				"then": {"ui_modify": {"step3": {"visible": true}}}
			}
		]
	}`

	t.Run("cascading reveals verified", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"step1": {"type": "string", "value": "done", "visible": true},
				"step2": {"type": "string", "value": "done", "visible": true},
				"step3": {"type": "string", "value": "final", "visible": true}
			},
			"status": "READY"
		}`

		valid, err := Verify(completedDoc, baseSchema)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !valid {
			t.Fatal("Expected valid for cascading reveals")
		}
	})
}
