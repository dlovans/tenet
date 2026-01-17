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
  4. Repeat until no new fields appear
  5. Compare computed values with newJson
```

## Algorithm

```go
func Verify(newJson, baseSchema string) (bool, error) {
    current := clone(baseSchema)
    previousFieldCount := 0
    
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
        
        // 5. Check for convergence (no new fields)
        currentFieldCount := countVisibleFields(result)
        if currentFieldCount == previousFieldCount {
            // Converged - now compare computed values
            return compareComputedValues(result, newJson)
        }
        
        previousFieldCount = currentFieldCount
        current = result
    }
    
    return false, errors.New("verification did not converge")
}
```

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
          No new fields → converged
  Step 3: Compare computed values → match → Verify = true
```

## Example: Invalid Path

```
newJson (user claims field_b, but revenue=3000):
  definitions:
    revenue: { value: 3000 }
    field_a: { value: null }
    field_b: { value: "INVALID" }

Verification:
  Step 1: copy revenue=3000 → Run()
          field_a becomes visible, NOT field_b
  Step 2: field_a is visible but no value in newJson → incomplete
          OR field_b has value but is not visible → invalid
  Result: Verify = false (field_b shouldn't have a value)
```

## Edge Cases

1. **Attestations**: Only copy attestation states when the attestation is visible
2. **Max iterations**: Prevent infinite loops (default: 100)
3. **Computed fields**: Always compare after convergence, never copy
4. **Initial visibility**: Some fields start visible, some start hidden

## Complexity

- Time: O(iterations × Run_cost)
- Typical: 2-5 iterations for most forms
- Max: Bounded by MAX_ITERATIONS constant
