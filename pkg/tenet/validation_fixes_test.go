package tenet

import (
	"encoding/json"
	"testing"
	"time"
)

// ===========================================================================
// Issue 1: Empty String Required Validation
// ===========================================================================

func TestEmptyStringRequiredValidation(t *testing.T) {
	// Test that empty string is treated as "missing" for required string fields
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"name": {"type": "string", "value": "", "required": true}
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

	// Status should be INCOMPLETE because empty string is "missing"
	if schema.Status != StatusIncomplete {
		t.Errorf("Expected status INCOMPLETE for empty required string, got %v", schema.Status)
	}

	// Should have an error for the missing field
	if len(schema.Errors) == 0 {
		t.Error("Expected error for empty required string field")
	}

	// Verify the error message mentions the field
	hasNameError := false
	for _, err := range schema.Errors {
		if err.FieldID == "name" || containsString(err.Message, "name") {
			hasNameError = true
			break
		}
	}
	if !hasNameError {
		t.Error("Expected error message to mention 'name' field")
	}
}

func TestEmptyStringNonRequiredIsValid(t *testing.T) {
	// Test that empty string is valid for non-required fields
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"notes": {"type": "string", "value": "", "required": false}
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

	// Status should be READY because field is not required
	if schema.Status != StatusReady {
		t.Errorf("Expected status READY for empty non-required string, got %v", schema.Status)
	}
}

func TestNonStringRequiredEmptyValueStillWorks(t *testing.T) {
	// Test that empty check only applies to strings (numbers with 0 should be valid)
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"quantity": {"type": "number", "value": 0, "required": true}
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

	// Status should be READY (0 is a valid value for required number)
	if schema.Status != StatusReady {
		t.Errorf("Expected status READY for zero required number, got %v", schema.Status)
	}
}

// User's original survey schema test case
func TestSurveySchemaEmptyAllergyNote(t *testing.T) {
	input := `{
		"protocol": "CoffeePreferenceSurvey_v1",
		"schema_id": "coffee-pref-001",
		"definitions": {
			"respondent_name": {
				"type": "string",
				"label": "Your Name",
				"required": true,
				"value": "Jane Doe"
			},
			"allergy_note": {
				"type": "string",
				"label": "Please describe your allergy",
				"required": true,
				"visible": true,
				"value": ""
			}
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

	// Should be INCOMPLETE because allergy_note is required but empty
	if schema.Status != StatusIncomplete {
		t.Errorf("Expected status INCOMPLETE for survey with empty required allergy_note, got %v", schema.Status)
	}
}

// ===========================================================================
// Issue 2: Derived Fields Shadowing
// ===========================================================================

func TestDerivedFieldTakesPrecedence(t *testing.T) {
	// Test that derived value is computed even when field exists in definitions
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"gross": {"type": "number", "value": 100},
			"tax": {"type": "number", "value": null, "readonly": true}
		},
		"state_model": {
			"derived": {
				"tax": {"eval": {"*": [{"var": "gross"}, 0.1]}}
			}
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

	// Tax should be computed as 10 (100 * 0.1)
	taxDef, ok := schema.Definitions["tax"]
	if !ok {
		t.Fatal("Expected 'tax' definition to exist")
	}

	taxVal, ok := toFloat(taxDef.Value)
	if !ok || taxVal != 10 {
		t.Errorf("Expected tax = 10, got %v", taxDef.Value)
	}
}

func TestDerivedFieldUsedInLogic(t *testing.T) {
	// Test that logic tree can use derived values
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"gross": {"type": "number", "value": 100}
		},
		"state_model": {
			"derived": {
				"tax": {"eval": {"*": [{"var": "gross"}, 0.1]}}
			}
		},
		"logic_tree": [
			{
				"id": "check_tax",
				"when": {">": [{"var": "tax"}, 5]},
				"then": {"set": {"high_tax": true}}
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

	// high_tax should be set because tax (10) > 5
	highTaxDef, ok := schema.Definitions["high_tax"]
	if !ok {
		t.Fatal("Expected 'high_tax' definition to be created (derived field should be available in logic)")
	}
	if highTaxDef.Value != true {
		t.Errorf("Expected high_tax = true, got %v", highTaxDef.Value)
	}
}

// ===========================================================================
// Issue 3: Execution Order (Logic Before Derived)
// ===========================================================================

func TestLogicCanUseDerivedValues(t *testing.T) {
	// Test that derived values are computed before logic tree evaluation
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"income": {"type": "number", "value": 50000},
			"deductions": {"type": "number", "value": 10000}
		},
		"state_model": {
			"derived": {
				"taxable_income": {"eval": {"-": [{"var": "income"}, {"var": "deductions"}]}}
			}
		},
		"logic_tree": [
			{
				"id": "high_income_bracket",
				"when": {">": [{"var": "taxable_income"}, 30000]},
				"then": {"set": {"tax_bracket": "high"}}
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

	// taxable_income should be 40000 (50000 - 10000)
	taxableIncomeDef, ok := schema.Definitions["taxable_income"]
	if !ok {
		t.Fatal("Expected 'taxable_income' definition to exist")
	}
	taxableVal, ok := toFloat(taxableIncomeDef.Value)
	if !ok || taxableVal != 40000 {
		t.Errorf("Expected taxable_income = 40000, got %v", taxableIncomeDef.Value)
	}

	// tax_bracket should be "high" because taxable_income (40000) > 30000
	taxBracketDef, ok := schema.Definitions["tax_bracket"]
	if !ok {
		t.Fatal("Expected 'tax_bracket' to be set by logic rule")
	}
	if taxBracketDef.Value != "high" {
		t.Errorf("Expected tax_bracket = 'high', got %v", taxBracketDef.Value)
	}
}

// User's original tax calculator schema test case
func TestTaxCalculatorSchema(t *testing.T) {
	input := `{
		"protocol": "IncomeTaxCalculator_v1",
		"schema_id": "tax-calc-001",
		"definitions": {
			"gross_annual_income": {
				"type": "currency",
				"value": 85000
			},
			"filing_status": {
				"type": "select",
				"options": ["single", "married_joint", "married_separate"],
				"value": "single"
			},
			"standard_deduction": {
				"type": "currency",
				"readonly": true,
				"value": null
			},
			"taxable_income": {
				"type": "currency",
				"readonly": true,
				"value": null
			},
			"effective_tax_rate": {
				"type": "number",
				"readonly": true,
				"value": null
			}
		},
		"state_model": {
			"derived": {
				"standard_deduction": {
					"eval": {
						"if": [
							{"==": [{"var": "filing_status"}, "single"]}, 14600,
							{"==": [{"var": "filing_status"}, "married_joint"]}, 29200,
							21900
						]
					}
				},
				"taxable_income": {
					"eval": {
						"-": [
							{"var": "gross_annual_income"},
							{"var": "standard_deduction"}
						]
					}
				}
			}
		},
		"logic_tree": [
			{
				"id": "calc_effective_rate",
				"when": {">": [{"var": "taxable_income"}, 0]},
				"then": {
					"set": {
						"effective_tax_rate": {
							"/": [
								{"*": [{"var": "taxable_income"}, 0.22]},
								{"var": "gross_annual_income"}
							]
						}
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

	// Check standard_deduction is 14600 for single
	stdDedDef, ok := schema.Definitions["standard_deduction"]
	if !ok {
		t.Fatal("Expected 'standard_deduction' definition")
	}
	stdDedVal, ok := toFloat(stdDedDef.Value)
	if !ok || stdDedVal != 14600 {
		t.Errorf("Expected standard_deduction = 14600, got %v", stdDedDef.Value)
	}

	// Check taxable_income is 70400 (85000 - 14600)
	taxableIncomeDef, ok := schema.Definitions["taxable_income"]
	if !ok {
		t.Fatal("Expected 'taxable_income' definition")
	}
	taxableVal, ok := toFloat(taxableIncomeDef.Value)
	if !ok || taxableVal != 70400 {
		t.Errorf("Expected taxable_income = 70400, got %v", taxableIncomeDef.Value)
	}

	// Check effective_tax_rate is calculated (70400 * 0.22 / 85000 = ~0.182)
	effectiveRateDef, ok := schema.Definitions["effective_tax_rate"]
	if !ok {
		t.Fatal("Expected 'effective_tax_rate' to be calculated by logic rule")
	}
	effectiveVal, ok := toFloat(effectiveRateDef.Value)
	if !ok || effectiveVal < 0.18 || effectiveVal > 0.19 {
		t.Errorf("Expected effective_tax_rate around 0.182, got %v", effectiveRateDef.Value)
	}
}

// ===========================================================================
// Edge Cases & Complex Scenarios
// ===========================================================================

func TestDerivedThatDependsOnOtherDerived(t *testing.T) {
	// Test chained derived fields
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"base": {"type": "number", "value": 100}
		},
		"state_model": {
			"derived": {
				"level1": {"eval": {"*": [{"var": "base"}, 2]}},
				"level2": {"eval": {"*": [{"var": "level1"}, 3]}}
			}
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

	// level1 = 100 * 2 = 200
	level1Def, ok := schema.Definitions["level1"]
	if !ok {
		t.Fatal("Expected 'level1' definition")
	}
	level1Val, ok := toFloat(level1Def.Value)
	if !ok || level1Val != 200 {
		t.Errorf("Expected level1 = 200, got %v", level1Def.Value)
	}

	// level2 = 200 * 3 = 600
	level2Def, ok := schema.Definitions["level2"]
	if !ok {
		t.Fatal("Expected 'level2' definition")
	}
	level2Val, ok := toFloat(level2Def.Value)
	if !ok || level2Val != 600 {
		t.Errorf("Expected level2 = 600, got %v", level2Def.Value)
	}
}

func TestLogicModifiesDerivedInput(t *testing.T) {
	// Test that derived values are re-computed after logic modifies their inputs
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"discount_eligible": {"type": "boolean", "value": false},
			"base_price": {"type": "number", "value": 100}
		},
		"state_model": {
			"derived": {
				"final_price": {
					"eval": {
						"if": [
							{"var": "discount_eligible"},
							{"*": [{"var": "base_price"}, 0.9]},
							{"var": "base_price"}
						]
					}
				}
			}
		},
		"logic_tree": [
			{
				"id": "apply_discount",
				"when": {">": [{"var": "base_price"}, 50]},
				"then": {"set": {"discount_eligible": true}}
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

	// discount_eligible should be true (base_price 100 > 50)
	discountDef, ok := schema.Definitions["discount_eligible"]
	if !ok || discountDef.Value != true {
		t.Errorf("Expected discount_eligible = true, got %v", discountDef)
	}

	// final_price should be 90 (100 * 0.9) because discount_eligible was set by logic
	// and derived was re-computed
	finalPriceDef, ok := schema.Definitions["final_price"]
	if !ok {
		t.Fatal("Expected 'final_price' definition")
	}
	finalVal, ok := toFloat(finalPriceDef.Value)
	if !ok || finalVal != 90 {
		t.Errorf("Expected final_price = 90, got %v", finalPriceDef.Value)
	}
}

func TestMultipleRequiredEmptyStrings(t *testing.T) {
	// Test multiple required empty strings
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"first_name": {"type": "string", "value": "", "required": true},
			"last_name": {"type": "string", "value": "", "required": true},
			"nickname": {"type": "string", "value": "", "required": false}
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

	// Status should be INCOMPLETE
	if schema.Status != StatusIncomplete {
		t.Errorf("Expected status INCOMPLETE, got %v", schema.Status)
	}

	// Should have errors for both first_name and last_name
	if len(schema.Errors) < 2 {
		t.Errorf("Expected at least 2 errors, got %d", len(schema.Errors))
	}
}

func TestSelectTypeEmptyStringRequired(t *testing.T) {
	// Test that empty string in select type is also caught
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"country": {"type": "select", "value": "", "required": true, "options": ["US", "CA", "UK"]}
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

	// Status should be INCOMPLETE or INVALID (empty is not a valid option)
	if schema.Status == StatusReady {
		t.Errorf("Expected status to not be READY for empty required select, got %v", schema.Status)
	}
}

func TestDerivedWithNullDefinitionValue(t *testing.T) {
	// Test that derived value is used even when definition has null value
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"input_a": {"type": "number", "value": 10},
			"input_b": {"type": "number", "value": 20},
			"result": {"type": "number", "value": null, "readonly": true}
		},
		"state_model": {
			"derived": {
				"result": {"eval": {"+": [{"var": "input_a"}, {"var": "input_b"}]}}
			}
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

	// result should be 30 (10 + 20), not null
	resultDef, ok := schema.Definitions["result"]
	if !ok {
		t.Fatal("Expected 'result' definition")
	}
	resultVal, ok := toFloat(resultDef.Value)
	if !ok || resultVal != 30 {
		t.Errorf("Expected result = 30, got %v", resultDef.Value)
	}
}

func TestLogicWithDerivedComparison(t *testing.T) {
	// Test logic rule that compares two derived values
	input := `{
		"protocol": "Test_v1",
		"schema_id": "test",
		"definitions": {
			"price_a": {"type": "number", "value": 100},
			"price_b": {"type": "number", "value": 80}
		},
		"state_model": {
			"derived": {
				"discounted_a": {"eval": {"*": [{"var": "price_a"}, 0.8]}},
				"discounted_b": {"eval": {"*": [{"var": "price_b"}, 0.9]}}
			}
		},
		"logic_tree": [
			{
				"id": "compare_prices",
				"when": {">": [{"var": "discounted_a"}, {"var": "discounted_b"}]},
				"then": {"set": {"best_deal": "B"}}
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

	// discounted_a = 100 * 0.8 = 80
	// discounted_b = 80 * 0.9 = 72
	// 80 > 72, so best_deal = "B"
	bestDealDef, ok := schema.Definitions["best_deal"]
	if !ok {
		t.Fatal("Expected 'best_deal' to be set")
	}
	if bestDealDef.Value != "B" {
		t.Errorf("Expected best_deal = 'B', got %v", bestDealDef.Value)
	}
}
