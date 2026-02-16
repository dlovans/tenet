package tenet

import (
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
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "value": "small corp", "visible": true, "required": true},
				"large_biz_field": {"type": "string", "visible": false},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		if result.Error != "" {
			t.Fatalf("Verify error: %s", result.Error)
		}
		if !result.Valid {
			t.Fatalf("Expected valid, got issues: %+v", result.Issues)
		}
		if result.Status != StatusReady {
			t.Fatalf("Expected status READY, got %s", result.Status)
		}
		if result.Schema == nil {
			t.Fatal("Expected schema in result")
		}
	})

	t.Run("valid large business path", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 10000, "visible": true},
				"small_biz_field": {"type": "string", "visible": false},
				"large_biz_field": {"type": "string", "value": "big corp", "visible": true, "required": true},
				"tax_bracket": {"type": "string", "value": "high", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		if result.Error != "" {
			t.Fatalf("Verify error: %s", result.Error)
		}
		if !result.Valid {
			t.Fatalf("Expected valid, got issues: %+v", result.Issues)
		}
	})

	t.Run("tampered computed value", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 10000, "visible": true},
				"large_biz_field": {"type": "string", "value": "big corp", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		if result.Valid {
			t.Fatal("Expected invalid due to tampered computed value")
		}
		// Should have a computed_mismatch issue for tax_bracket
		found := false
		for _, issue := range result.Issues {
			if issue.Code == VerifyComputedMismatch && issue.FieldID == "tax_bracket" {
				found = true
				if issue.Expected != "high" {
					t.Fatalf("Expected 'high', got expected=%v", issue.Expected)
				}
				if issue.Claimed != "low" {
					t.Fatalf("Expected claimed 'low', got claimed=%v", issue.Claimed)
				}
			}
		}
		if !found {
			t.Fatalf("Expected computed_mismatch issue for tax_bracket, got: %+v", result.Issues)
		}
	})

	t.Run("wrong branch - claims field that shouldnt be visible", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "visible": false},
				"large_biz_field": {"type": "string", "value": "FAKE", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		t.Logf("Result: valid=%v, issues=%+v", result.Valid, result.Issues)
	})

	t.Run("status tampering", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "visible": true, "required": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		if result.Valid {
			t.Fatal("Expected invalid due to status tampering")
		}
		// Should have a status_mismatch issue
		found := false
		for _, issue := range result.Issues {
			if issue.Code == VerifyStatusMismatch {
				found = true
			}
		}
		if !found {
			t.Fatalf("Expected status_mismatch issue, got: %+v", result.Issues)
		}
	})

	t.Run("renamed field - field_a renamed to field_x", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"fake_renamed_field": {"type": "string", "value": "small corp", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		t.Logf("Renamed field result: valid=%v, issues=%+v", result.Valid, result.Issues)
	})

	t.Run("extra field that doesnt exist in base", func(t *testing.T) {
		completedDoc := `{
			"definitions": {
				"revenue": {"type": "number", "value": 3000, "visible": true},
				"small_biz_field": {"type": "string", "value": "small corp", "visible": true, "required": true},
				"INJECTED_FIELD": {"type": "string", "value": "hacked", "visible": true},
				"tax_bracket": {"type": "string", "value": "low", "readonly": true, "visible": true}
			},
			"status": "READY"
		}`

		result := Verify(completedDoc, baseSchema)
		if result.Valid {
			t.Fatal("Expected invalid due to injected field")
		}
		// Should have an unknown_field issue
		found := false
		for _, issue := range result.Issues {
			if issue.Code == VerifyUnknownField && issue.FieldID == "INJECTED_FIELD" {
				found = true
			}
		}
		if !found {
			t.Fatalf("Expected unknown_field issue for INJECTED_FIELD, got: %+v", result.Issues)
		}
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

		result := Verify(completedDoc, baseSchema)
		if result.Error != "" {
			t.Fatalf("Verify error: %s", result.Error)
		}
		if !result.Valid {
			t.Fatalf("Expected valid, got issues: %+v", result.Issues)
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

		result := Verify(completedDoc, baseSchema)
		if result.Valid {
			t.Fatal("Expected invalid due to missing evidence")
		}
		// Should have attestation evidence/timestamp issues
		foundEvidence := false
		for _, issue := range result.Issues {
			if issue.Code == VerifyAttestationNoEvidence && issue.FieldID == "officer_sign" {
				foundEvidence = true
			}
		}
		if !foundEvidence {
			t.Fatalf("Expected attestation_no_evidence issue, got: %+v", result.Issues)
		}
	})
}

func TestVerifyConvergence(t *testing.T) {
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

		result := Verify(completedDoc, baseSchema)
		if result.Error != "" {
			t.Fatalf("Verify error: %s", result.Error)
		}
		if !result.Valid {
			t.Fatalf("Expected valid, got issues: %+v", result.Issues)
		}
	})
}
