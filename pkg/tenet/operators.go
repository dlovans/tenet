package tenet

import (
	"fmt"
	"strings"
	"time"
)

// executeOperator handles all JSON-logic operators.
// Returns nil for operations on nil values (nil-safe behavior).
func (e *Engine) executeOperator(op string, args any) any {
	switch op {
	// === Variable Access ===
	case "var":
		path, ok := args.(string)
		if !ok {
			return nil
		}
		return e.getVar(path)

	// === Comparison Operators ===
	case "==":
		a := e.resolveArgs(args, 2)
		return e.compareEqual(a[0], a[1])

	case "!=":
		a := e.resolveArgs(args, 2)
		return !e.compareEqual(a[0], a[1])

	case ">":
		a := e.resolveArgs(args, 2)
		return e.compareNumeric(a[0], a[1], func(x, y float64) bool { return x > y })

	case "<":
		a := e.resolveArgs(args, 2)
		return e.compareNumeric(a[0], a[1], func(x, y float64) bool { return x < y })

	case ">=":
		a := e.resolveArgs(args, 2)
		return e.compareNumeric(a[0], a[1], func(x, y float64) bool { return x >= y })

	case "<=":
		a := e.resolveArgs(args, 2)
		return e.compareNumeric(a[0], a[1], func(x, y float64) bool { return x <= y })

	// === Logical Operators ===
	case "and":
		return e.opAnd(args)

	case "or":
		return e.opOr(args)

	case "not", "!":
		a := e.resolveArgs(args, 1)
		return !e.isTruthy(a[0])

	case "if":
		return e.opIf(args)

	// === Arithmetic Operators ===
	case "+":
		a := e.resolveArgs(args, 2)
		return e.opAdd(a[0], a[1])

	case "-":
		a := e.resolveArgs(args, 2)
		return e.opSubtract(a[0], a[1])

	case "*":
		a := e.resolveArgs(args, 2)
		return e.opMultiply(a[0], a[1])

	case "/":
		a := e.resolveArgs(args, 2)
		return e.opDivide(a[0], a[1])

	// === Date Operators ===
	case "before":
		a := e.resolveArgs(args, 2)
		return e.compareDates(a[0], a[1], func(x, y time.Time) bool { return x.Before(y) })

	case "after":
		a := e.resolveArgs(args, 2)
		return e.compareDates(a[0], a[1], func(x, y time.Time) bool { return x.After(y) })

	// === Collection Operators ===
	case "in":
		a := e.resolveArgs(args, 2)
		return e.opIn(a[0], a[1])

	case "some":
		return e.opSome(args)

	case "all":
		return e.opAll(args)

	case "none":
		return e.opNone(args)

	default:
		// Unknown operator - add error and return nil
		e.addError("", "", ErrRuntimeWarning, fmt.Sprintf("Unknown operator '%s' in logic expression", op), "")
		return nil
	}
}

// === Comparison Helpers ===

// compareEqual checks equality, handling type coercion.
// nil == nil is true, nil == anything_else is false.
func (e *Engine) compareEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try numeric comparison if both can be numbers
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if aOk && bOk {
		return aNum == bNum
	}

	// String comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// compareNumeric compares two values numerically.
// Returns false if either value is nil or non-numeric.
func (e *Engine) compareNumeric(a, b any, cmp func(float64, float64) bool) bool {
	if a == nil || b == nil {
		return false
	}

	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if !aOk || !bOk {
		return false
	}

	return cmp(aNum, bNum)
}

// compareDates compares two date values.
// Accepts strings (ISO format) or time.Time values.
// Returns false if either value is nil or unparseable.
func (e *Engine) compareDates(a, b any, cmp func(time.Time, time.Time) bool) bool {
	aTime, aOk := parseDate(a)
	bTime, bOk := parseDate(b)
	if !aOk || !bOk {
		return false
	}
	return cmp(aTime, bTime)
}

// === Logical Operators ===

// opAnd returns true if all arguments are truthy.
// Short-circuits on first falsy value.
func (e *Engine) opAnd(args any) bool {
	arr, ok := args.([]any)
	if !ok {
		return e.isTruthy(e.resolve(args))
	}

	for _, arg := range arr {
		if !e.isTruthy(e.resolve(arg)) {
			return false
		}
	}
	return true
}

// opOr returns true if any argument is truthy.
// Short-circuits on first truthy value.
func (e *Engine) opOr(args any) bool {
	arr, ok := args.([]any)
	if !ok {
		return e.isTruthy(e.resolve(args))
	}

	for _, arg := range arr {
		if e.isTruthy(e.resolve(arg)) {
			return true
		}
	}
	return false
}

// opIf implements ternary logic: {"if": [condition, then, else]}
// Also supports chained conditions: {"if": [c1, t1, c2, t2, ..., else]}
func (e *Engine) opIf(args any) any {
	arr, ok := args.([]any)
	if !ok || len(arr) < 2 {
		return nil
	}

	// Process condition-then pairs
	for i := 0; i+1 < len(arr); i += 2 {
		condition := e.resolve(arr[i])
		if e.isTruthy(condition) {
			return e.resolve(arr[i+1])
		}
	}

	// Else clause (odd number of elements = has else)
	if len(arr)%2 == 1 {
		return e.resolve(arr[len(arr)-1])
	}

	return nil
}

// === Arithmetic Operators ===

// opAdd adds two numbers. Returns nil if either is nil.
func (e *Engine) opAdd(a, b any) any {
	if a == nil || b == nil {
		return nil
	}
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if !aOk || !bOk {
		return nil
	}
	return aNum + bNum
}

// opSubtract subtracts b from a. Returns nil if either is nil.
func (e *Engine) opSubtract(a, b any) any {
	if a == nil || b == nil {
		return nil
	}
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if !aOk || !bOk {
		return nil
	}
	return aNum - bNum
}

// opMultiply multiplies two numbers. Returns nil if either is nil.
func (e *Engine) opMultiply(a, b any) any {
	if a == nil || b == nil {
		return nil
	}
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if !aOk || !bOk {
		return nil
	}
	return aNum * bNum
}

// opDivide divides a by b. Returns nil if either is nil or b is zero.
func (e *Engine) opDivide(a, b any) any {
	if a == nil || b == nil {
		return nil
	}
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if !aOk || !bOk || bNum == 0 {
		return nil
	}
	return aNum / bNum
}

// === Collection Operators ===

// opSome returns true if ANY element in the array satisfies the condition.
// Syntax: {"some": [{"var": "items"}, {"==": [{"var": ""}, "special"]}]}
func (e *Engine) opSome(args any) bool {
	arr, ok := args.([]any)
	if !ok || len(arr) < 2 {
		return false
	}

	// First arg is the array (resolve it)
	collection := e.resolve(arr[0])
	items, ok := collection.([]any)
	if !ok || len(items) == 0 {
		return false
	}

	// Second arg is the condition
	condition := arr[1]

	// Check if any element satisfies the condition
	for _, item := range items {
		if e.evalWithContext(condition, item) {
			return true
		}
	}
	return false
}

// opAll returns true if ALL elements in the array satisfy the condition.
// Syntax: {"all": [{"var": "items"}, {">=": [{"var": ""}, 50]}]}
func (e *Engine) opAll(args any) bool {
	arr, ok := args.([]any)
	if !ok || len(arr) < 2 {
		return false
	}

	// First arg is the array (resolve it)
	collection := e.resolve(arr[0])
	items, ok := collection.([]any)
	if !ok {
		return false
	}

	// Empty array returns true for "all" (vacuous truth)
	if len(items) == 0 {
		return true
	}

	// Second arg is the condition
	condition := arr[1]

	// Check if all elements satisfy the condition
	for _, item := range items {
		if !e.evalWithContext(condition, item) {
			return false
		}
	}
	return true
}

// opNone returns true if NO elements in the array satisfy the condition.
// Syntax: {"none": [{"var": "items"}, {"==": [{"var": ""}, "forbidden"]}]}
func (e *Engine) opNone(args any) bool {
	arr, ok := args.([]any)
	if !ok || len(arr) < 2 {
		return false
	}

	// First arg is the array (resolve it)
	collection := e.resolve(arr[0])
	items, ok := collection.([]any)
	if !ok {
		return false
	}

	// Empty array returns true for "none"
	if len(items) == 0 {
		return true
	}

	// Second arg is the condition
	condition := arr[1]

	// Check that no element satisfies the condition
	for _, item := range items {
		if e.evalWithContext(condition, item) {
			return false
		}
	}
	return true
}

// evalWithContext evaluates a condition with a temporary context value.
// Used by some/all/none to set the current element as {"var": ""}.
func (e *Engine) evalWithContext(condition any, contextValue any) bool {
	// Save and restore the context value for {"var": ""}
	oldContext := e.currentElement
	e.currentElement = contextValue
	result := e.isTruthy(e.resolve(condition))
	e.currentElement = oldContext
	return result
}

// opIn checks if needle is in haystack (array or string).
func (e *Engine) opIn(needle, haystack any) bool {
	if needle == nil || haystack == nil {
		return false
	}

	switch h := haystack.(type) {
	case []any:
		for _, item := range h {
			if e.compareEqual(needle, item) {
				return true
			}
		}
		return false

	case string:
		needleStr, ok := needle.(string)
		if !ok {
			return false
		}
		return strings.Contains(h, needleStr)

	default:
		return false
	}
}

// === Helper Functions ===

// isSlice returns true if the value is a slice/array (e.g. []any from JSON).
func isSlice(v any) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case []any, []string, []float64, []int:
		return true
	default:
		return false
	}
}

// toFloat converts a value to float64 if possible.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		// Don't auto-convert strings to numbers
		return 0, false
	default:
		return 0, false
	}
}

// parseDate parses a date value (string or time.Time).
// Supports ISO 8601 formats.
func parseDate(v any) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}

	switch d := v.(type) {
	case time.Time:
		return d, true
	case string:
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, d); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

