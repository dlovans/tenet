# Verification Algorithm

The `Verify()` function proves that a completed document was correctly derived from a base schema by simulating the user's journey step-by-step.

## Problem

Simple comparison doesn't work because:
- Logic rules can reveal/hide fields based on user input
- Different inputs lead to different visible fields (branching)
- We can't know which fields should exist without replaying the journey

## Turn-Based Approach

```
Input:
  - newJson: Completed document with user inputs + computed values
  - baseSchema: Original template before any user interaction

Process:
  1. Start with baseSchema (only initial visible fields)
  2. Extract values for those fields from newJson
  3. Run() to reveal next set of fields
  4. Repeat until no new fields appear (convergence)
  5. Compare computed values, attestations, and status with newJson
  6. Return structured VerifyResult with all issues found
```

## Algorithm

```go
func Verify(newJson, baseSchema string) VerifyResult {
    current := clone(baseSchema)
    var previousFieldIDs []string

    for iterations := 0; iterations < MAX_ITERATIONS; iterations++ {
        // 1. Get visible, non-readonly fields in current
        visibleFields := getVisibleEditableFields(current)

        // 2. Copy values from newJson for those fields
        for _, fieldId := range visibleFields {
            if value, exists := newJson.definitions[fieldId]; exists {
                current.definitions[fieldId].value = value
            }
        }

        // 3. Copy attestation states for visible attestations
        copyVisibleAttestations(current, newJson)

        // 4. Run() to process rules and reveal new fields
        result := Run(current, effectiveDate)

        // 5. Check for convergence using sorted visible field ID sets
        currentFieldIDs := getSortedVisibleFieldIDs(result)
        if slices.Equal(currentFieldIDs, previousFieldIDs) {
            // Converged — now validate the final state
            return validateFinalState(result, newJson)
        }

        previousFieldIDs = currentFieldIDs
        current = result
    }

    return VerifyResult{
        Valid: false,
        Issues: []VerifyIssue{{
            Code:    "convergence_failed",
            Message: "verification did not converge",
        }},
    }
}
```

> **Note:** Convergence is detected by comparing sorted visible field ID sets, not just field counts. This prevents false convergence when fields swap visibility (one hides, another shows).

## Structured Output

`Verify()` returns a `VerifyResult` containing **all** issues found, not just the first. This enables UI layers to display everything wrong at once:

```go
type VerifyResult struct {
    Valid  bool          // Overall pass/fail
    Status DocStatus     // Document status from final run()
    Issues []VerifyIssue // All problems found
    Schema *Schema       // The full re-run result
    Error  string        // Internal error (parse failure, panic)
}
```

### Issue Codes

| Code | Meaning |
|------|---------|
| `unknown_field` | Submitted field doesn't exist in base schema or derived state |
| `computed_mismatch` | Readonly field value was tampered (includes expected/claimed) |
| `attestation_unsigned` | Required attestation not signed |
| `attestation_no_evidence` | Signed but missing evidence object |
| `attestation_no_timestamp` | Evidence present but missing timestamp |
| `status_mismatch` | Claimed status doesn't match what the VM computed |
| `convergence_failed` | Document didn't converge within max iterations |
| `internal_error` | Unexpected error (parse failure, panic recovery, etc.) |

### Final State Validation

After convergence, `validateFinalState` checks:

1. **Unknown fields** — Every field in the submitted document must exist in the base schema or have been created by `set` operations. Fields that appear in neither are flagged as `unknown_field`.

2. **Computed value integrity** — Every `readonly` field's value in the submitted document must match what the VM computed. Mismatches are flagged as `computed_mismatch` with `expected` and `claimed` values.

3. **Attestation completeness** — Required attestations must be signed with evidence containing a timestamp.

4. **Status consistency** — The submitted `status` must match what the VM computed from the final state.

## Example: Branching

```
baseSchema:
  definitions:
    revenue: { type: number, value: null }
    field_a: { type: string, visible: false }
    field_b: { type: string, visible: false }
  logic_tree:
    - when: revenue <= 5000 → show field_a
    - when: revenue > 5000 → show field_b

newJson (user entered revenue=3000, filled field_a):
  definitions:
    revenue: { value: 3000 }
    field_a: { value: "small business" }
    field_b: { value: null }  # never shown

Verification:
  Step 1: revenue is visible → copy 3000 → Run()
          Result: field_a becomes visible (3000 <= 5000)
  Step 2: field_a is visible → copy "small business" → Run()
          Sorted field IDs match previous → converged
  Step 3: Validate final state → all computed values match → valid: true
```

## Example: Tampered Document

```
newJson (user tampered tax_bracket from "high" to "low"):
  definitions:
    revenue: { value: 10000 }
    tax_bracket: { value: "low", readonly: true }  # VM would compute "high"

Verification:
  Step 1-N: Replay converges normally
  Final validation:
    tax_bracket: expected "high", claimed "low" → computed_mismatch

  Result:
    valid: false
    issues: [{
      code: "computed_mismatch",
      field_id: "tax_bracket",
      message: "readonly field value mismatch: expected high, got low",
      expected: "high",
      claimed: "low"
    }]
```

## Example: Injected Field

```
newJson (attacker added a field):
  definitions:
    revenue: { value: 3000 }
    INJECTED_FIELD: { value: "hacked" }  # doesn't exist in schema

Result:
  valid: false
  issues: [{
    code: "unknown_field",
    field_id: "INJECTED_FIELD",
    message: "field not found in base schema or derived state"
  }]
```

## Edge Cases

1. **Attestations**: Only copy attestation states when the attestation is visible
2. **Max iterations**: Prevent infinite loops (default: 100)
3. **Computed fields**: Always compare after convergence, never copy during replay
4. **Initial visibility**: `visible` defaults to `true` when not specified in the schema
5. **Panic recovery**: Both Go `Run()` and `Verify()` include `defer recover()` — panics are caught and returned as `internal_error` issues

## Complexity

- Time: O(iterations x Run_cost)
- Typical: 2-5 iterations for most forms
- Max: Bounded by MAX_ITERATIONS constant
