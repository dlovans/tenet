package tenet

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Run executes the schema logic for a given effective date.
// It evaluates the logic tree, computes derived state, and validates the document.
// Returns the transformed JSON with computed state, errors, and status.
//
// This is the "Transformer" - it takes raw input and returns a fully evaluated document.
// Panic-safe: recovers from any unexpected panic and returns it as an error.
func Run(jsonText string, date time.Time) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = ""
			err = fmt.Errorf("internal error: %v", r)
		}
	}()

	// 1. Unmarshal
	var schema Schema
	if err := json.Unmarshal([]byte(jsonText), &schema); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	if schema.Definitions == nil {
		schema.Definitions = make(map[string]*Definition)
	}

	// Initialize default visibility for definitions
	for _, def := range schema.Definitions {
		if def != nil && def.Visible == nil {
			t := true
			def.Visible = &t
		}
	}

	engine := NewEngine(&schema)

	// 2. Validate and select temporal branch, prune inactive rules
	if len(schema.TemporalMap) > 0 {
		engine.validateTemporalMap()
		branch := engine.selectBranch(date)
		if branch != nil {
			engine.prune(branch)
		}
	}

	// 3. Compute derived state (so logic tree can use derived values)
	engine.computeDerived()

	// 4. Evaluate logic tree
	engine.evaluateLogicTree()

	// 5. Re-compute derived state (in case logic modified inputs)
	engine.computeDerived()

	// 6. Validate
	engine.validateDefinitions()
	engine.checkAttestations()

	// 7. Determine status and attach errors
	schema.Errors = engine.errors
	schema.Status = engine.determineStatus()

	// 8. Marshal result
	return engine.marshal()
}

// Verify checks that a completed document (newJson) was correctly derived from a base schema.
// It simulates the user's journey by iteratively copying visible field values and re-running.
//
// Optional maxIterations parameter (default: 100) limits the replay iterations.
//
// This is the "Auditor" - it proves the transformation was legal by replaying the journey.
// Returns a structured VerifyResult with all issues found (not just the first).
// Panic-safe: recovers from any unexpected panic and returns it as an internal_error issue.
func Verify(newJson, baseSchemaJson string, maxIter ...int) (vr VerifyResult) {
	defer func() {
		if r := recover(); r != nil {
			vr = VerifyResult{
				Valid: false,
				Issues: []VerifyIssue{{
					Code:    VerifyInternalError,
					Message: fmt.Sprintf("internal panic: %v", r),
				}},
				Error: fmt.Sprintf("internal panic: %v", r),
			}
		}
	}()

	maxIterations := 100
	if len(maxIter) > 0 && maxIter[0] > 0 {
		maxIterations = maxIter[0]
	}

	// Parse both documents
	var newSchema Schema
	if err := json.Unmarshal([]byte(newJson), &newSchema); err != nil {
		return VerifyResult{
			Valid: false,
			Issues: []VerifyIssue{{
				Code:    VerifyInternalError,
				Message: fmt.Sprintf("failed to parse submitted document: %v", err),
			}},
			Error: fmt.Sprintf("unmarshal newJson: %v", err),
		}
	}

	// Extract effective date from newJson
	effectiveDate := time.Now()
	if newSchema.ValidFrom != "" {
		if parsed, ok := parseDate(newSchema.ValidFrom); ok {
			effectiveDate = parsed
		}
	}

	// Start with base schema
	currentJson := baseSchemaJson
	previousVisibleSet := ""

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Parse current state
		var currentSchema Schema
		if err := json.Unmarshal([]byte(currentJson), &currentSchema); err != nil {
			return VerifyResult{
				Valid: false,
				Issues: []VerifyIssue{{
					Code:    VerifyInternalError,
					Message: fmt.Sprintf("failed to parse schema at iteration %d", iteration),
				}},
				Error: fmt.Sprintf("unmarshal current (iteration %d): %v", iteration, err),
			}
		}

		// Count visible editable fields before copying
		visibleEditable := getVisibleEditableFields(&currentSchema)

		// Copy values from newJson for visible, editable fields
		for fieldId := range visibleEditable {
			if newDef, ok := newSchema.Definitions[fieldId]; ok && newDef != nil {
				if currentDef, ok := currentSchema.Definitions[fieldId]; ok && currentDef != nil {
					currentDef.Value = newDef.Value
				}
			}
		}

		// Copy attestation states for visible attestations
		for attId, currentAtt := range currentSchema.Attestations {
			if currentAtt == nil {
				continue
			}
			if newAtt, ok := newSchema.Attestations[attId]; ok && newAtt != nil {
				currentAtt.Signed = newAtt.Signed
				currentAtt.Evidence = newAtt.Evidence
			}
		}

		// Run the schema
		modifiedJson, err := json.Marshal(currentSchema)
		if err != nil {
			return VerifyResult{
				Valid: false,
				Issues: []VerifyIssue{{
					Code:    VerifyInternalError,
					Message: fmt.Sprintf("failed to serialize schema at iteration %d", iteration),
				}},
				Error: fmt.Sprintf("marshal (iteration %d): %v", iteration, err),
			}
		}

		resultJson, err := Run(string(modifiedJson), effectiveDate)
		if err != nil {
			return VerifyResult{
				Valid: false,
				Issues: []VerifyIssue{{
					Code:    VerifyInternalError,
					Message: fmt.Sprintf("VM run failed at iteration %d", iteration),
				}},
				Error: fmt.Sprintf("run failed (iteration %d): %v", iteration, err),
			}
		}

		// Parse result
		var resultSchema Schema
		if err := json.Unmarshal([]byte(resultJson), &resultSchema); err != nil {
			return VerifyResult{
				Valid: false,
				Issues: []VerifyIssue{{
					Code:    VerifyInternalError,
					Message: fmt.Sprintf("failed to parse VM result at iteration %d", iteration),
				}},
				Error: fmt.Sprintf("unmarshal result (iteration %d): %v", iteration, err),
			}
		}

		// Build sorted set of visible field IDs for convergence check
		currentVisibleSet := visibleFieldSet(&resultSchema)

		// Check for convergence
		if currentVisibleSet == previousVisibleSet {
			// Converged - now validate the final state and return full result
			return validateFinalState(&newSchema, &resultSchema)
		}

		previousVisibleSet = currentVisibleSet
		currentJson = resultJson
	}

	return VerifyResult{
		Valid: false,
		Issues: []VerifyIssue{{
			Code:    VerifyConvergenceFailed,
			Message: fmt.Sprintf("document did not converge after %d iterations", maxIterations),
		}},
	}
}

// getVisibleEditableFields returns field IDs that are visible and not readonly
func getVisibleEditableFields(schema *Schema) map[string]bool {
	result := make(map[string]bool)
	for id, def := range schema.Definitions {
		if def != nil && def.Visible != nil && *def.Visible && !def.Readonly {
			result[id] = true
		}
	}
	return result
}

// visibleFieldSet returns a sorted string of visible field IDs for convergence checking.
func visibleFieldSet(schema *Schema) string {
	var ids []string
	for id, def := range schema.Definitions {
		if def != nil && def.Visible != nil && *def.Visible {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

// validateFinalState compares computed values and attestation fulfillment.
// Collects ALL issues instead of bailing on the first — the UI needs the complete picture.
func validateFinalState(newSchema, resultSchema *Schema) VerifyResult {
	engine := &Engine{}
	var issues []VerifyIssue

	// Check for unknown/injected fields in newSchema that don't exist in result
	for id := range newSchema.Definitions {
		if _, existsInResult := resultSchema.Definitions[id]; !existsInResult {
			issues = append(issues, VerifyIssue{
				Code:    VerifyUnknownField,
				FieldID: id,
				Message: fmt.Sprintf("field '%s' does not exist in the schema", id),
			})
		}
	}

	// Compare computed (readonly) values
	for id, resultDef := range resultSchema.Definitions {
		if resultDef == nil || !resultDef.Readonly {
			continue
		}

		newDef, ok := newSchema.Definitions[id]
		if !ok {
			issues = append(issues, VerifyIssue{
				Code:     VerifyComputedMismatch,
				FieldID:  id,
				Message:  fmt.Sprintf("computed field '%s' is missing from the submitted document", id),
				Expected: resultDef.Value,
			})
			continue
		}

		if !engine.compareEqual(newDef.Value, resultDef.Value) {
			issues = append(issues, VerifyIssue{
				Code:     VerifyComputedMismatch,
				FieldID:  id,
				Message:  fmt.Sprintf("computed field '%s' was modified", id),
				Expected: resultDef.Value,
				Claimed:  newDef.Value,
			})
		}
	}

	// Verify attestations are fulfilled
	for id, resultAtt := range resultSchema.Attestations {
		if resultAtt == nil || !resultAtt.Required {
			continue
		}

		newAtt, ok := newSchema.Attestations[id]
		if !ok {
			continue
		}

		if !newAtt.Signed {
			issues = append(issues, VerifyIssue{
				Code:    VerifyAttestationUnsigned,
				FieldID: id,
				Message: fmt.Sprintf("required attestation '%s' has not been signed", id),
			})
			continue // No point checking evidence if unsigned
		}

		if newAtt.Evidence == nil || newAtt.Evidence.ProviderAuditID == "" {
			issues = append(issues, VerifyIssue{
				Code:    VerifyAttestationNoEvidence,
				FieldID: id,
				Message: fmt.Sprintf("attestation '%s' is signed but missing proof of signing", id),
			})
		}

		if newAtt.Evidence == nil || newAtt.Evidence.Timestamp == "" {
			issues = append(issues, VerifyIssue{
				Code:    VerifyAttestationNoTimestamp,
				FieldID: id,
				Message: fmt.Sprintf("attestation '%s' is signed but missing a timestamp", id),
			})
		}
	}

	// Verify status matches
	if newSchema.Status != resultSchema.Status {
		issues = append(issues, VerifyIssue{
			Code:     VerifyStatusMismatch,
			Message:  "the document status does not match what was computed",
			Expected: resultSchema.Status,
			Claimed:  newSchema.Status,
		})
	}

	return VerifyResult{
		Valid:  len(issues) == 0,
		Status: resultSchema.Status,
		Issues: issues,
		Schema: resultSchema,
	}
}

// evaluateLogicTree processes all active rules in order.
func (e *Engine) evaluateLogicTree() {
	for _, rule := range e.schema.LogicTree {
		if rule == nil || rule.Disabled {
			continue
		}

		// Evaluate the condition
		condition := e.resolve(rule.When)
		if e.isTruthy(condition) {
			e.applyAction(rule.Then, rule.ID, rule.LawRef)
		}
	}
}

// applyAction executes a rule's action: setting values, modifying UI, or emitting errors.
func (e *Engine) applyAction(action *Action, ruleID, lawRef string) {
	if action == nil {
		return
	}

	// Apply value mutations
	if action.Set != nil {
		for key, value := range action.Set {
			// Resolve the value in case it's an expression
			resolvedValue := e.resolve(value)
			e.setDefinitionValue(key, resolvedValue, ruleID)
		}
	}

	// Apply UI modifications
	if action.UIModify != nil {
		for key, mods := range action.UIModify {
			e.applyUIModify(key, mods)
		}
	}

	// Emit error if specified
	if action.ErrorMsg != "" {
		kind := action.ErrorKind
		if kind == "" {
			kind = ErrConstraintViolation
		}
		e.addError("", ruleID, kind, action.ErrorMsg, lawRef)
	}
}

// setDefinitionValue updates or creates a definition value.
// Tracks which rule set each field to detect potential cycles.
func (e *Engine) setDefinitionValue(key string, value any, ruleID string) {
	// Cycle detection: check if this field was already set by a different rule
	if prevRule, alreadySet := e.fieldsSet[key]; alreadySet && prevRule != ruleID {
		e.addError(key, ruleID, ErrCycleDetected, fmt.Sprintf(
			"potential cycle: field '%s' set by rule '%s' and again by rule '%s'",
			key, prevRule, ruleID), "")
	}
	e.fieldsSet[key] = ruleID

	def, ok := e.schema.Definitions[key]
	if !ok {
		// Create new definition if it doesn't exist
		t := true
		e.schema.Definitions[key] = &Definition{
			Type:    inferType(value),
			Value:   value,
			Visible: &t,
		}
		return
	}

	def.Value = value
}

// applyUIModify applies UI metadata changes to a definition.
func (e *Engine) applyUIModify(key string, mods any) {
	def, ok := e.schema.Definitions[key]
	if !ok || def == nil {
		return
	}

	modMap, ok := mods.(map[string]any)
	if !ok {
		return
	}

	// Apply visibility and metadata modifications
	if visible, ok := modMap["visible"].(bool); ok {
		def.Visible = &visible
	}
	if uiClass, ok := modMap["ui_class"].(string); ok {
		def.UIClass = uiClass
	}
	if uiMessage, ok := modMap["ui_message"].(string); ok {
		def.UIMessage = uiMessage
	}
	if required, ok := modMap["required"].(bool); ok {
		def.Required = required
	}

	// Apply numeric constraints (min, max, step)
	if minVal, ok := toFloat(modMap["min"]); ok {
		def.Min = &minVal
	}
	if maxVal, ok := toFloat(modMap["max"]); ok {
		def.Max = &maxVal
	}
	if stepVal, ok := toFloat(modMap["step"]); ok {
		def.Step = &stepVal
	}

	// Apply string constraints (min_length, max_length)
	if minLen, ok := modMap["min_length"].(float64); ok {
		intVal := int(minLen)
		def.MinLength = &intVal
	}
	if maxLen, ok := modMap["max_length"].(float64); ok {
		intVal := int(maxLen)
		def.MaxLength = &intVal
	}
	if pattern, ok := modMap["pattern"].(string); ok {
		def.Pattern = pattern
	}
}

// computeDerived evaluates all derived fields in the state model.
func (e *Engine) computeDerived() {
	if e.schema.StateModel == nil || e.schema.StateModel.Derived == nil {
		return
	}

	for name, derivedDef := range e.schema.StateModel.Derived {
		if derivedDef == nil || derivedDef.Eval == nil {
			continue
		}

		// Evaluate the expression
		value := e.resolve(derivedDef.Eval)

		if existing, ok := e.schema.Definitions[name]; ok && existing != nil {
			existing.Value = value
			existing.Readonly = true
			if existing.Visible == nil {
				t := true
				existing.Visible = &t
			}
		} else {
			t := true
			e.schema.Definitions[name] = &Definition{
				Type:     inferType(value),
				Value:    value,
				Readonly: true,
				Visible:  &t,
			}
		}
	}
}

// marshal converts the schema back to JSON.
func (e *Engine) marshal() (string, error) {
	result, err := json.MarshalIndent(e.schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	return string(result), nil
}

// inferType determines the type string for a value.
func inferType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	default:
		return "string"
	}
}
