package tenet

import (
	"encoding/json"
	"testing"
	"time"
)

// BenchmarkRun measures the throughput of the VM on a realistic schema.
func BenchmarkRun(b *testing.B) {
	effectiveDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)

	// Create a realistic loan application schema
	schema := createBenchmarkSchema()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(schema, effectiveDate)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRunParallel measures throughput with concurrent requests.
func BenchmarkRunParallel(b *testing.B) {
	effectiveDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	schema := createBenchmarkSchema()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Run(schema, effectiveDate)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLargeSchema tests performance with many definitions and rules.
func BenchmarkLargeSchema(b *testing.B) {
	effectiveDate := time.Now()
	schema := createLargeSchema(100, 50) // 100 definitions, 50 rules

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(schema, effectiveDate)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerify measures the cost of turn-based verification.
func BenchmarkVerify(b *testing.B) {
	effectiveDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	baseSchema := createBenchmarkSchema()

	// Run once to get a valid completed document
	completedDoc, err := Run(baseSchema, effectiveDate)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := Verify(completedDoc, baseSchema)
		if result.Error != "" {
			b.Fatal(result.Error)
		}
		if !result.Valid {
			b.Fatal("expected valid")
		}
	}
}

// BenchmarkVerifyParallel measures Verify throughput with concurrency.
func BenchmarkVerifyParallel(b *testing.B) {
	effectiveDate := time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)
	baseSchema := createBenchmarkSchema()
	completedDoc, err := Run(baseSchema, effectiveDate)
	if err != nil {
		b.Fatal(err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := Verify(completedDoc, baseSchema)
			if result.Error != "" {
				b.Fatal(result.Error)
			}
			if !result.Valid {
				b.Fatal("expected valid")
			}
		}
	})
}

func createBenchmarkSchema() string {
	schema := map[string]any{
		"protocol":   "Tenet_v1.0",
		"schema_id":  "loan_benchmark",
		"version":    "2025.01.16",
		"valid_from": "2025-01-01",
		"definitions": map[string]any{
			"applicant_income": map[string]any{
				"type":     "number",
				"value":    75000.0,
				"required": true,
			},
			"loan_amount": map[string]any{
				"type":     "number",
				"value":    250000.0,
				"required": true,
			},
			"employment_status": map[string]any{
				"type":     "select",
				"value":    "employed",
				"options":  []string{"employed", "self_employed", "unemployed", "retired"},
				"required": true,
			},
			"credit_score": map[string]any{
				"type":     "number",
				"value":    720.0,
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
		},
		"logic_tree": []any{
			map[string]any{
				"id":      "rule_unemployed_denial",
				"law_ref": "Lending Standards Act §4.2",
				"when":    map[string]any{"==": []any{map[string]any{"var": "employment_status"}, "unemployed"}},
				"then": map[string]any{
					"set":       map[string]any{"approval_status": "denied", "risk_level": "high"},
					"error_msg": "Unemployed applicants do not meet requirements.",
				},
			},
			map[string]any{
				"id":   "rule_good_credit",
				"when": map[string]any{">=": []any{map[string]any{"var": "credit_score"}, 700}},
				"then": map[string]any{
					"set": map[string]any{"approval_status": "approved", "risk_level": "low"},
				},
			},
			map[string]any{
				"id":   "rule_dti_warning",
				"when": map[string]any{">": []any{map[string]any{"var": "debt_to_income_ratio"}, 0.43}},
				"then": map[string]any{
					"set":       map[string]any{"risk_level": "medium"},
					"error_msg": "DTI exceeds 43% guideline.",
				},
			},
		},
		"state_model": map[string]any{
			"inputs": []string{"applicant_income", "loan_amount"},
			"derived": map[string]any{
				"debt_to_income_ratio": map[string]any{
					"eval": map[string]any{"/": []any{
						map[string]any{"var": "loan_amount"},
						map[string]any{"*": []any{map[string]any{"var": "applicant_income"}, 30}},
					}},
				},
				"max_loan_eligible": map[string]any{
					"eval": map[string]any{"*": []any{map[string]any{"var": "applicant_income"}, 4}},
				},
			},
		},
	}

	bytes, _ := json.Marshal(schema)
	return string(bytes)
}

func createLargeSchema(numDefs, numRules int) string {
	definitions := make(map[string]any)
	for i := 0; i < numDefs; i++ {
		definitions[string(rune('a'+i%26))+string(rune('0'+i/26))] = map[string]any{
			"type":  "number",
			"value": float64(i * 100),
		}
	}

	logicTree := make([]any, numRules)
	for i := 0; i < numRules; i++ {
		logicTree[i] = map[string]any{
			"id":   "rule_" + string(rune('0'+i)),
			"when": map[string]any{">": []any{map[string]any{"var": "a0"}, float64(i * 10)}},
			"then": map[string]any{"set": map[string]any{"b0": float64(i)}},
		}
	}

	schema := map[string]any{
		"definitions": definitions,
		"logic_tree":  logicTree,
	}

	bytes, _ := json.Marshal(schema)
	return string(bytes)
}
