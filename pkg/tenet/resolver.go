package tenet

import (
	"fmt"
	"strings"
)

// Engine holds state during execution of a schema.
type Engine struct {
	schema            *Schema
	errors            []ValidationError
	fieldsSet         map[string]string // tracks which fields were set by which rule (cycle detection)
	currentElement    any               // current element context for some/all/none operators
	derivedInProgress map[string]bool   // cycle detection for derived fields
}

// NewEngine creates an engine for the given schema.
func NewEngine(schema *Schema) *Engine {
	return &Engine{
		schema:            schema,
		errors:            make([]ValidationError, 0),
		fieldsSet:         make(map[string]string),
		derivedInProgress: make(map[string]bool),
	}
}

// resolve evaluates any JSON-logic node and returns its value.
// This is the recursive core of the VM.
// It is nil-safe: operations on nil values return appropriate defaults without crashing.
func (e *Engine) resolve(node any) any {
	if node == nil {
		return nil
	}

	switch v := node.(type) {
	case map[string]any:
		// It's an operator: {"==": [a, b]} or {"var": "field_name"}
		if len(v) == 1 {
			for op, args := range v {
				return e.executeOperator(op, args)
			}
		}
		// Multi-key map is treated as a literal object
		return v

	case []any:
		// Array literal - resolve each element
		result := make([]any, len(v))
		for i, elem := range v {
			result[i] = e.resolve(elem)
		}
		return result

	case string, float64, int, bool:
		// Literal value
		return v

	default:
		return v
	}
}

// getVar retrieves a value using dot notation: "user.address.city"
// Returns nil if the path doesn't exist (distinguishes "unknown" from "zero").
// Special case: empty path "" returns the current element context (used by some/all/none).
func (e *Engine) getVar(path string) any {
	if path == "" {
		// Return current element context for {"var": ""} in some/all/none
		return e.currentElement
	}

	parts := strings.Split(path, ".")

	// First, check derived state (derived values take precedence)
	if e.schema.StateModel != nil && e.schema.StateModel.Derived != nil {
		if derived, ok := e.schema.StateModel.Derived[parts[0]]; ok {
			if e.derivedInProgress[parts[0]] {
				e.addError("", "", ErrCycleDetected, fmt.Sprintf("Circular dependency detected in derived field '%s'", parts[0]), "")
				return nil
			}
			e.derivedInProgress[parts[0]] = true
			result := e.resolve(derived.Eval)
			delete(e.derivedInProgress, parts[0])
			if len(parts) == 1 {
				return result
			}
			return e.accessPath(result, parts[1:])
		}
	}

	// Then, check definitions
	if def, ok := e.schema.Definitions[parts[0]]; ok {
		if len(parts) == 1 {
			return def.Value
		}
		// Nested access into the value
		return e.accessPath(def.Value, parts[1:])
	}

	// Variable not found - add error (unless we're in a some/all/none context)
	if e.currentElement == nil {
		e.addError("", "", ErrRuntimeWarning, fmt.Sprintf("Undefined variable '%s' in logic expression", parts[0]), "")
	}

	return nil
}

// accessPath traverses nested maps/structs using the remaining path parts.
// Returns nil if any part of the path doesn't exist.
func (e *Engine) accessPath(value any, parts []string) any {
	if len(parts) == 0 || value == nil {
		return value
	}

	switch v := value.(type) {
	case map[string]any:
		next, ok := v[parts[0]]
		if !ok {
			return nil
		}
		return e.accessPath(next, parts[1:])

	default:
		// Can't traverse into non-map types
		return nil
	}
}

// isTruthy determines if a value is "truthy" in JSON-logic terms.
// nil, false, 0, and "" are falsy. Everything else is truthy.
func (e *Engine) isTruthy(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

// resolveArgs resolves an args node (expected to be []any) and returns the resolved values.
// If args is not an array or has fewer elements than expected, missing values are nil.
func (e *Engine) resolveArgs(args any, expected int) []any {
	result := make([]any, expected)

	arr, ok := args.([]any)
	if !ok {
		// Single value case (e.g., {"not": true})
		if expected > 0 {
			result[0] = e.resolve(args)
		}
		return result
	}

	for i := 0; i < expected && i < len(arr); i++ {
		result[i] = e.resolve(arr[i])
	}
	return result
}

// addError appends a validation error to the engine's error list.
func (e *Engine) addError(fieldID, ruleID string, kind ErrorKind, message, lawRef string) {
	e.errors = append(e.errors, ValidationError{
		FieldID: fieldID,
		RuleID:  ruleID,
		Kind:    kind,
		Message: message,
		LawRef:  lawRef,
	})
}
