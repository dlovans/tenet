package tenet

import (
	"encoding/json"
	"fmt"
	"time"
)

// Run executes the schema logic for a given effective date.
// It evaluates the logic tree, computes derived state, and validates the document.
// Returns the transformed JSON with computed state, errors, and status.
//
// This is the "Transformer" - it takes raw input and returns a fully evaluated document.
func Run(jsonText string, date time.Time) (string, error) {
	// 1. Unmarshal
	var schema Schema
	if err := json.Unmarshal([]byte(jsonText), &schema); err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}

	// Initialize default visibility for definitions
	for _, def := range schema.Definitions {
		if def != nil {
			// Default visible to true if not explicitly set
			// (Go zero value is false, so we need to handle this)
			def.Visible = true
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
func Verify(newJson, baseSchemaJson string, maxIter ...int) (bool, error) {
	maxIterations := 100
	if len(maxIter) > 0 && maxIter[0] > 0 {
		maxIterations = maxIter[0]
	}

	// Parse both documents
	var newSchema Schema
	if err := json.Unmarshal([]byte(newJson), &newSchema); err != nil {
		return false, fmt.Errorf("unmarshal newJson: %w", err)
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
	previousVisibleCount := -1

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Parse current state
		var currentSchema Schema
		if err := json.Unmarshal([]byte(currentJson), &currentSchema); err != nil {
			return false, fmt.Errorf("unmarshal current (iteration %d): %w", iteration, err)
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
			return false, fmt.Errorf("marshal (iteration %d): %w", iteration, err)
		}

		resultJson, err := Run(string(modifiedJson), effectiveDate)
		if err != nil {
			return false, fmt.Errorf("run failed (iteration %d): %w", iteration, err)
		}

		// Parse result
		var resultSchema Schema
		if err := json.Unmarshal([]byte(resultJson), &resultSchema); err != nil {
			return false, fmt.Errorf("unmarshal result (iteration %d): %w", iteration, err)
		}

		// Count visible fields after run
		currentVisibleCount := countVisibleFields(&resultSchema)

		// Check for convergence
		if currentVisibleCount == previousVisibleCount {
			// Converged - now validate the final state
			return validateFinalState(&newSchema, &resultSchema)
		}

		previousVisibleCount = currentVisibleCount
		currentJson = resultJson
	}

	return false, fmt.Errorf("verification did not converge after %d iterations", maxIterations)
}

// getVisibleEditableFields returns field IDs that are visible and not readonly
func getVisibleEditableFields(schema *Schema) map[string]bool {
	result := make(map[string]bool)
	for id, def := range schema.Definitions {
		if def != nil && def.Visible && !def.Readonly {
			result[id] = true
		}
	}
	return result
}

// countVisibleFields returns the number of visible fields
func countVisibleFields(schema *Schema) int {
	count := 0
	for _, def := range schema.Definitions {
		if def != nil && def.Visible {
			count++
		}
	}
	return count
}

// validateFinalState compares computed values and attestation fulfillment
func validateFinalState(newSchema, resultSchema *Schema) (bool, error) {
	engine := &Engine{}

	// Check for unknown/injected fields in newSchema that don't exist in result
	for id := range newSchema.Definitions {
		if _, existsInResult := resultSchema.Definitions[id]; !existsInResult {
			return false, fmt.Errorf("unknown field '%s' not in schema", id)
		}
	}

	// Compare computed (readonly) values
	for id, resultDef := range resultSchema.Definitions {
		if resultDef == nil || !resultDef.Readonly {
			continue
		}

		newDef, ok := newSchema.Definitions[id]
		if !ok {
			return false, fmt.Errorf("computed field '%s' missing in submitted document", id)
		}

		if !engine.compareEqual(newDef.Value, resultDef.Value) {
			return false, fmt.Errorf("computed field '%s' mismatch: claimed %v, expected %v",
				id, newDef.Value, resultDef.Value)
		}
	}

	// Verify attestations are fulfilled
	for id, resultAtt := range resultSchema.Attestations {
		if resultAtt == nil {
			continue
		}

		newAtt, ok := newSchema.Attestations[id]
		if !ok {
			continue // Attestation not required in result
		}

		if resultAtt.Required {
			if !newAtt.Signed {
				return false, fmt.Errorf("required attestation '%s' not signed", id)
			}
			if newAtt.Evidence == nil || newAtt.Evidence.ProviderAuditID == "" {
				return false, fmt.Errorf("attestation '%s' missing evidence", id)
			}
			if newAtt.Evidence.Timestamp == "" {
				return false, fmt.Errorf("attestation '%s' missing timestamp", id)
			}
		}
	}

	// Verify status matches
	if newSchema.Status != resultSchema.Status {
		return false, fmt.Errorf("status mismatch: claimed %s, expected %s",
			newSchema.Status, resultSchema.Status)
	}

	return true, nil
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
		e.addError("", ruleID, action.ErrorMsg, lawRef)
	}
}

// setDefinitionValue updates or creates a definition value.
// Tracks which rule set each field to detect potential cycles.
func (e *Engine) setDefinitionValue(key string, value any, ruleID string) {
	// Cycle detection: check if this field was already set by a different rule
	if prevRule, alreadySet := e.fieldsSet[key]; alreadySet && prevRule != ruleID {
		e.addError(key, ruleID, fmt.Sprintf(
			"potential cycle: field '%s' set by rule '%s' and again by rule '%s'",
			key, prevRule, ruleID), "")
	}
	e.fieldsSet[key] = ruleID

	def, ok := e.schema.Definitions[key]
	if !ok {
		// Create new definition if it doesn't exist
		e.schema.Definitions[key] = &Definition{
			Type:    inferType(value),
			Value:   value,
			Visible: true,
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
		def.Visible = visible
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

		// Store the computed value as a definition (readonly)
		e.schema.Definitions[name] = &Definition{
			Type:     inferType(value),
			Value:    value,
			Readonly: true,
			Visible:  true,
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
		return "any"
	}
}
