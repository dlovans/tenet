package tenet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const outputDir = "testdata/output"

// writeOutput writes the result JSON to the output directory for inspection.
func writeOutput(t *testing.T, name, result string) {
	t.Helper()
	path := filepath.Join(outputDir, name+".json")
	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		t.Logf("Warning: could not write output file: %v", err)
	}
}

// TestReactiveTransformation demonstrates the core VM behavior:
// When definition values change, Run() re-evaluates the logic tree
// and returns modified JSON with triggered rules applied.
func TestReactiveTransformation(t *testing.T) {
	effectiveDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)

	t.Run("employed with good credit becomes approved", func(t *testing.T) {
		// Initial state: employed, credit_score=720, loan=250k, income=75k
		input := createLoanSchema("employed", 720, 75000, 250000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "01_employed_good_credit", result)
		schema := parseResult(t, result)

		// Should be APPROVED because:
		// - credit_score >= 700 ✓
		// - employment_status in [employed, self_employed] ✓
		// - debt_to_income_ratio <= 0.43 ✓
		assertDefinitionValue(t, schema, "approval_status", "approved")
		assertDefinitionValue(t, schema, "risk_level", "low")
		assertEqual(t, schema.Status, StatusReady)

		// Derived values should be computed
		assertDefinitionExists(t, schema, "debt_to_income_ratio")
		assertDefinitionExists(t, schema, "max_loan_eligible")

		// max_loan_eligible = income * 4 = 300000
		assertDefinitionValue(t, schema, "max_loan_eligible", float64(300000))
	})

	t.Run("user changes employment to unemployed - triggers denial", func(t *testing.T) {
		// User updated employment_status from "employed" to "unemployed"
		input := createLoanSchema("unemployed", 720, 75000, 250000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "02_unemployed_denial", result)
		schema := parseResult(t, result)

		// Should be DENIED because unemployed triggers rule_unemployed_denial
		assertDefinitionValue(t, schema, "approval_status", "denied")
		assertDefinitionValue(t, schema, "risk_level", "high")

		// Should have error with law reference
		assertHasErrorWithLawRef(t, schema, "Lending Standards Act §4.2")
	})

	t.Run("user changes credit score to 580 - triggers review", func(t *testing.T) {
		// User updated credit_score from 720 to 580
		input := createLoanSchema("employed", 580, 75000, 250000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "03_low_credit_review", result)
		schema := parseResult(t, result)

		// Should be REVIEW_REQUIRED because credit < 650
		assertDefinitionValue(t, schema, "approval_status", "review_required")
		assertDefinitionValue(t, schema, "risk_level", "high")
		assertDefinitionValue(t, schema, "additional_docs_required", true)

		// Attestation should be required now
		assertDefinitionRequired(t, schema, "income_verification", true)

		// Should have error with law reference
		assertHasErrorWithLawRef(t, schema, "Consumer Credit Reg §12.1")

		// Status should be INCOMPLETE because attestation is now required but not confirmed
		assertEqual(t, schema.Status, StatusIncomplete)
	})

	t.Run("self-employed requires additional docs", func(t *testing.T) {
		input := createLoanSchema("self_employed", 750, 100000, 200000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "04_self_employed", result)
		schema := parseResult(t, result)

		// Should still be approved (good credit, valid employment)
		assertDefinitionValue(t, schema, "approval_status", "approved")

		// But should require additional docs for self-employed
		assertDefinitionValue(t, schema, "additional_docs_required", true)
		assertDefinitionRequired(t, schema, "income_verification", true)
	})

	t.Run("high loan amount triggers DTI warning", func(t *testing.T) {
		// Loan amount that creates DTI > 0.43
		// DTI = loan / (income * 30)
		// With income=50000 and loan=1000000:
		// DTI = 1000000 / (50000 * 30) = 1000000 / 1500000 = 0.667 > 0.43
		input := createLoanSchema("employed", 750, 50000, 1000000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "05_high_dti_warning", result)
		schema := parseResult(t, result)

		// DTI rule should have fired
		assertDefinitionValue(t, schema, "risk_level", "medium")
		assertHasErrorWithLawRef(t, schema, "Responsible Lending Code §8.3")

		// Should NOT be approved because DTI > 0.43 fails the approval condition
		// approval_status stays "pending" (never set to approved)
		assertDefinitionValue(t, schema, "approval_status", "pending")
	})
}

// TestDerivedStateComputation tests that derived values are correctly computed
// from input definitions using JSON-logic expressions.
func TestDerivedStateComputation(t *testing.T) {
	effectiveDate := time.Now()

	t.Run("max_loan_eligible computed from income", func(t *testing.T) {
		tests := []struct {
			income   float64
			expected float64
		}{
			{50000, 200000},  // 50k * 4
			{75000, 300000},  // 75k * 4
			{100000, 400000}, // 100k * 4
			{150000, 600000}, // 150k * 4
		}

		for _, tt := range tests {
			input := createLoanSchema("employed", 750, tt.income, 100000)
			result, err := Run(input, effectiveDate)
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			schema := parseResult(t, result)
			assertDefinitionValue(t, schema, "max_loan_eligible", tt.expected)
		}
	})

	t.Run("debt_to_income_ratio computed correctly", func(t *testing.T) {
		// DTI = loan / (income * 30)
		input := createLoanSchema("employed", 750, 60000, 180000)
		// Expected: 180000 / (60000 * 30) = 180000 / 1800000 = 0.1

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		schema := parseResult(t, result)

		// Get the computed DTI
		dtiDef := schema.Definitions["debt_to_income_ratio"]
		if dtiDef == nil {
			t.Fatal("debt_to_income_ratio not found")
		}

		dti, ok := dtiDef.Value.(float64)
		if !ok {
			t.Fatalf("debt_to_income_ratio is not a number: %v", dtiDef.Value)
		}

		// Allow small floating point error
		expected := 0.1
		if dti < expected-0.001 || dti > expected+0.001 {
			t.Errorf("debt_to_income_ratio = %v, want ~%v", dti, expected)
		}
	})
}

// TestUIModification tests that rules can modify UI metadata on definitions.
func TestUIModification(t *testing.T) {
	effectiveDate := time.Now()

	t.Run("low credit makes attestation visible and required", func(t *testing.T) {
		input := createLoanSchema("employed", 580, 75000, 250000)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		schema := parseResult(t, result)

		// income_verification should now be visible and required
		attestation := schema.Definitions["income_verification"]
		if attestation == nil {
			t.Fatal("income_verification not found")
		}

		if attestation.Visible == nil || !*attestation.Visible {
			t.Error("income_verification should be visible")
		}
		if !attestation.Required {
			t.Error("income_verification should be required")
		}
	})
}

// TestErrorAccumulation tests that errors are accumulated (not fail-fast).
func TestErrorAccumulation(t *testing.T) {
	effectiveDate := time.Now()

	t.Run("multiple rule violations accumulate errors", func(t *testing.T) {
		// Create a schema that triggers multiple rules
		// low credit (< 650) AND unemployed (worst case)
		input := createLoanSchemaFull("unemployed", 580, 75000, 250000, false)

		result, err := Run(input, effectiveDate)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		writeOutput(t, "06_multiple_violations", result)
		schema := parseResult(t, result)

		// Should have errors from BOTH rules
		assertHasErrorWithLawRef(t, schema, "Lending Standards Act §4.2") // unemployed
		assertHasErrorWithLawRef(t, schema, "Consumer Credit Reg §12.1")  // low credit

		// Multiple errors accumulated
		if len(schema.Errors) < 2 {
			t.Errorf("Expected at least 2 errors, got %d", len(schema.Errors))
		}
	})
}

// === Helper Functions ===

func createLoanSchema(employment string, creditScore, income, loanAmount float64) string {
	return createLoanSchemaFull(employment, creditScore, income, loanAmount, false)
}

func createLoanSchemaFull(employment string, creditScore, income, loanAmount float64, attestationConfirmed bool) string {
	schema := map[string]any{
		"protocol":   "Tenet_v1.0",
		"schema_id":  "loan_application",
		"version":    "2025.01.16",
		"valid_from": "2025-01-01",
		"definitions": map[string]any{
			"applicant_income": map[string]any{
				"type":     "number",
				"value":    income,
				"required": true,
			},
			"loan_amount": map[string]any{
				"type":     "number",
				"value":    loanAmount,
				"required": true,
			},
			"employment_status": map[string]any{
				"type":     "select",
				"value":    employment,
				"options":  []string{"employed", "self_employed", "unemployed", "retired"},
				"required": true,
			},
			"credit_score": map[string]any{
				"type":     "number",
				"value":    creditScore,
				"required": true,
			},
			"approval_status": map[string]any{
				"type":    "select",
				"options": []string{"pending", "approved", "denied", "review_required"},
				"value":   "pending",
			},
			"risk_level": map[string]any{
				"type":    "select",
				"options": []string{"low", "medium", "high"},
			},
			"additional_docs_required": map[string]any{
				"type":  "boolean",
				"value": false,
			},
			"income_verification": map[string]any{
				"type":     "attestation",
				"label":    "I confirm the income information is accurate.",
				"required": false,
				"value":    attestationConfirmed,
			},
		},
		"logic_tree": []any{
			map[string]any{
				"id":      "rule_unemployed_denial",
				"law_ref": "Lending Standards Act §4.2",
				"when":    map[string]any{"==": []any{map[string]any{"var": "employment_status"}, "unemployed"}},
				"then": map[string]any{
					"set":       map[string]any{"approval_status": "denied", "risk_level": "high"},
					"error_msg": "Unemployed applicants do not meet minimum employment requirements per Lending Standards Act §4.2.",
				},
			},
			map[string]any{
				"id":      "rule_low_credit_review",
				"law_ref": "Consumer Credit Reg §12.1",
				"when":    map[string]any{"<": []any{map[string]any{"var": "credit_score"}, 650}},
				"then": map[string]any{
					"set":       map[string]any{"approval_status": "review_required", "risk_level": "high", "additional_docs_required": true},
					"ui_modify": map[string]any{"income_verification": map[string]any{"visible": true, "required": true}},
					"error_msg": "Credit score below threshold requires manual review per Consumer Credit Reg §12.1.",
				},
			},
			map[string]any{
				"id":      "rule_high_dti_warning",
				"law_ref": "Responsible Lending Code §8.3",
				"when":    map[string]any{">": []any{map[string]any{"var": "debt_to_income_ratio"}, 0.43}},
				"then": map[string]any{
					"set":       map[string]any{"risk_level": "medium"},
					"error_msg": "Debt-to-income ratio exceeds 43% guideline per Responsible Lending Code §8.3.",
				},
			},
			map[string]any{
				"id":   "rule_self_employed_docs",
				"when": map[string]any{"==": []any{map[string]any{"var": "employment_status"}, "self_employed"}},
				"then": map[string]any{
					"set":       map[string]any{"additional_docs_required": true},
					"ui_modify": map[string]any{"income_verification": map[string]any{"visible": true, "required": true}},
				},
			},
			map[string]any{
				"id": "rule_good_credit_approval",
				"when": map[string]any{
					"and": []any{
						map[string]any{">=": []any{map[string]any{"var": "credit_score"}, 700}},
						map[string]any{"in": []any{map[string]any{"var": "employment_status"}, []any{"employed", "self_employed"}}},
						map[string]any{"<=": []any{map[string]any{"var": "debt_to_income_ratio"}, 0.43}},
					},
				},
				"then": map[string]any{
					"set": map[string]any{"approval_status": "approved", "risk_level": "low"},
				},
			},
		},
		"state_model": map[string]any{
			"inputs": []string{"applicant_income", "loan_amount"},
			"derived": map[string]any{
				"debt_to_income_ratio": map[string]any{
					"eval": map[string]any{
						"if": []any{
							map[string]any{">": []any{map[string]any{"var": "applicant_income"}, 0}},
							map[string]any{"/": []any{
								map[string]any{"var": "loan_amount"},
								map[string]any{"*": []any{map[string]any{"var": "applicant_income"}, 30}},
							}},
							0,
						},
					},
				},
				"max_loan_eligible": map[string]any{
					"eval": map[string]any{
						"*": []any{map[string]any{"var": "applicant_income"}, 4},
					},
				},
			},
		},
	}

	bytes, _ := json.Marshal(schema)
	return string(bytes)
}

func parseResult(t *testing.T, result string) *Schema {
	t.Helper()
	var schema Schema
	if err := json.Unmarshal([]byte(result), &schema); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}
	return &schema
}

func assertDefinitionValue(t *testing.T, schema *Schema, id string, expected any) {
	t.Helper()
	def, ok := schema.Definitions[id]
	if !ok {
		t.Errorf("Definition '%s' not found", id)
		return
	}
	if def.Value != expected {
		t.Errorf("Definition '%s' = %v, want %v", id, def.Value, expected)
	}
}

func assertDefinitionExists(t *testing.T, schema *Schema, id string) {
	t.Helper()
	if _, ok := schema.Definitions[id]; !ok {
		t.Errorf("Definition '%s' should exist", id)
	}
}

func assertDefinitionRequired(t *testing.T, schema *Schema, id string, expected bool) {
	t.Helper()
	def, ok := schema.Definitions[id]
	if !ok {
		t.Errorf("Definition '%s' not found", id)
		return
	}
	if def.Required != expected {
		t.Errorf("Definition '%s'.Required = %v, want %v", id, def.Required, expected)
	}
}

func assertHasErrorWithLawRef(t *testing.T, schema *Schema, lawRef string) {
	t.Helper()
	for _, err := range schema.Errors {
		if err.LawRef == lawRef {
			return
		}
	}
	t.Errorf("Expected error with law_ref '%s', but not found in %v", lawRef, schema.Errors)
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
