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

	// 2. Select temporal branch and prune inactive rules
	if len(schema.TemporalMap) > 0 {
		branch := engine.selectBranch(date)
		if branch != nil {
			engine.prune(branch)
		}
	}

	// 3. Evaluate logic tree
	engine.evaluateLogicTree()

	// 4. Compute derived state
	engine.computeDerived()

	// 5. Validate
	engine.validateDefinitions()
	engine.checkAttestations()

	// 6. Determine status and attach errors
	schema.Errors = engine.errors
	schema.Status = engine.determineStatus()

	// 7. Marshal result
	return engine.marshal()
}

// Verify checks that newJson is a valid derivation of oldJson.
// It re-runs the logic on oldJson with the effective date from newJson
// and compares the resulting values.
//
// This is the "Auditor" - it proves the transformation was legal.
func Verify(newJson, oldJson string) (bool, error) {
	// Parse both documents
	var newSchema, oldSchema Schema
	if err := json.Unmarshal([]byte(newJson), &newSchema); err != nil {
		return false, fmt.Errorf("unmarshal newJson: %w", err)
	}
	if err := json.Unmarshal([]byte(oldJson), &oldSchema); err != nil {
		return false, fmt.Errorf("unmarshal oldJson: %w", err)
	}

	// Extract effective date from newJson (use ValidFrom or current time)
	effectiveDate := time.Now()
	if newSchema.ValidFrom != "" {
		if parsed, ok := parseDate(newSchema.ValidFrom); ok {
			effectiveDate = parsed
		}
	}

	// Re-run the logic on oldJson
	resultJson, err := Run(oldJson, effectiveDate)
	if err != nil {
		return false, fmt.Errorf("replay failed: %w", err)
	}

	// Parse the result and compare definition values
	var resultSchema Schema
	if err := json.Unmarshal([]byte(resultJson), &resultSchema); err != nil {
		return false, fmt.Errorf("unmarshal result: %w", err)
	}

	// Compare all definition values
	for id, newDef := range newSchema.Definitions {
		if newDef == nil {
			continue
		}

		resultDef, ok := resultSchema.Definitions[id]
		if !ok {
			return false, fmt.Errorf("definition '%s' missing in replay", id)
		}

		// Compare values using the same equality logic
		engine := &Engine{}
		if !engine.compareEqual(newDef.Value, resultDef.Value) {
			return false, fmt.Errorf("definition '%s' value mismatch: got %v, expected %v",
				id, newDef.Value, resultDef.Value)
		}
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
			e.setDefinitionValue(key, resolvedValue)
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
func (e *Engine) setDefinitionValue(key string, value any) {
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
