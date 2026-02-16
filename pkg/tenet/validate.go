package tenet

import (
	"fmt"
	"regexp"
	"strings"
)

// validateDefinitions checks all definitions for type correctness and required fields.
// Accumulates all errors (non-blocking).
func (e *Engine) validateDefinitions() {
	for id, def := range e.schema.Definitions {
		if def == nil {
			continue
		}

		// Check required fields
		if def.Required {
			if def.Value == nil {
				e.addError(id, "", ErrMissingRequired, fmt.Sprintf("Required field '%s' is missing", id), "")
			} else if def.Type == "string" || def.Type == "select" {
				// Empty string is also considered "missing" for required string/select fields
				if strVal, ok := def.Value.(string); ok && strVal == "" {
					e.addError(id, "", ErrMissingRequired, fmt.Sprintf("Required field '%s' is missing", id), "")
				}
			}
		}

		// Validate type if value is present
		if def.Value != nil {
			e.validateType(id, def)
		}
	}
}

// validateType ensures a value matches its definition type and constraints.
// Array values are allowed — the declared type describes the element type,
// used by collection operators (some/all/none). Scalar validation is skipped for arrays.
func (e *Engine) validateType(id string, def *Definition) {
	value := def.Value

	// Skip scalar validation for array values (used with some/all/none operators)
	if isSlice(value) {
		return
	}

	switch def.Type {
	case "string":
		strVal, ok := value.(string)
		if !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Field '%s' must be a string", id), "")
			return
		}
		// Validate string length constraints
		e.validateStringConstraints(id, strVal, def)

	case "number", "currency":
		numVal, ok := toFloat(value)
		if !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Field '%s' must be a number", id), "")
			return
		}
		// Validate numeric range constraints
		e.validateNumericConstraints(id, numVal, def)

	case "boolean":
		if _, ok := value.(bool); !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Field '%s' must be a boolean", id), "")
		}

	case "select":
		// Validate that value is one of the allowed options
		strVal, ok := value.(string)
		if !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Field '%s' must be a string", id), "")
			return
		}
		if !e.isValidOption(strVal, def.Options) {
			e.addError(id, "", ErrConstraintViolation, fmt.Sprintf("Field '%s' value '%s' is not a valid option", id, strVal), "")
		}

	case "attestation":
		// Attestations must be boolean
		if _, ok := value.(bool); !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Attestation '%s' must be a boolean", id), "")
		}

	case "date":
		// Validate date format
		if _, ok := parseDate(value); !ok {
			e.addError(id, "", ErrTypeMismatch, fmt.Sprintf("Field '%s' must be a valid date", id), "")
		}
	}
}

// validateNumericConstraints checks min/max bounds for numeric values.
func (e *Engine) validateNumericConstraints(id string, value float64, def *Definition) {
	if def.Min != nil && value < *def.Min {
		e.addError(id, "", ErrConstraintViolation, fmt.Sprintf("Field '%s' value %.2f is below minimum %.2f", id, value, *def.Min), "")
	}
	if def.Max != nil && value > *def.Max {
		e.addError(id, "", ErrConstraintViolation, fmt.Sprintf("Field '%s' value %.2f exceeds maximum %.2f", id, value, *def.Max), "")
	}
}

// validateStringConstraints checks length and pattern constraints for strings.
func (e *Engine) validateStringConstraints(id string, value string, def *Definition) {
	if def.MinLength != nil && len(value) < *def.MinLength {
		e.addError(id, "", ErrConstraintViolation, fmt.Sprintf("Field '%s' is too short (minimum %d characters)", id, *def.MinLength), "")
	}
	if def.MaxLength != nil && len(value) > *def.MaxLength {
		e.addError(id, "", ErrConstraintViolation, fmt.Sprintf("Field '%s' is too long (maximum %d characters)", id, *def.MaxLength), "")
	}
	if def.Pattern != "" {
		re, err := regexp.Compile(def.Pattern)
		if err == nil && !re.MatchString(value) {
			e.addError(id, "", ErrConstraintViolation,
				fmt.Sprintf("Field '%s' does not match required pattern", id), "")
		}
	}
}

// checkAttestations ensures all required attestations are confirmed.
// Validates both legacy attestations in definitions and rich attestations.
func (e *Engine) checkAttestations() {
	// Check legacy attestations in definitions (simple type: attestation)
	for id, def := range e.schema.Definitions {
		if def == nil || def.Type != "attestation" {
			continue
		}
		if def.Required && def.Value != true {
			e.addError(id, "", ErrAttestationIncomplete, fmt.Sprintf("Required attestation '%s' not confirmed", id), "")
		}
	}

	// Check rich attestations
	for id, att := range e.schema.Attestations {
		if att == nil {
			continue
		}

		// Process on_sign if signed is true
		if att.Signed && att.OnSign != nil {
			e.applyAction(att.OnSign, "attestation_"+id, att.LawRef)
		}

		// Validate required attestations
		if att.Required {
			if !att.Signed {
				e.addError(id, "", ErrAttestationIncomplete, fmt.Sprintf("Required attestation '%s' not signed", id), att.LawRef)
			} else if att.Evidence == nil || att.Evidence.ProviderAuditID == "" {
				e.addError(id, "", ErrAttestationIncomplete, fmt.Sprintf("Attestation '%s' signed but missing evidence", id), att.LawRef)
			}
		}
	}
}

// isValidOption checks if a value is in the allowed options list.
func (e *Engine) isValidOption(value string, options []string) bool {
	if options == nil {
		return true // No restrictions
	}
	for _, opt := range options {
		if opt == value {
			return true
		}
	}
	return false
}

// determineStatus calculates the document status based on validation errors.
func (e *Engine) determineStatus() DocStatus {
	for _, err := range e.errors {
		if err.Kind == ErrTypeMismatch {
			return StatusInvalid
		}
	}
	for _, err := range e.errors {
		if err.Kind == ErrMissingRequired || err.Kind == ErrAttestationIncomplete {
			return StatusIncomplete
		}
	}
	for _, err := range e.errors {
		if err.Kind == ErrConstraintViolation {
			return StatusInvalid
		}
	}
	return StatusReady
}

// containsString is a simple substring check.
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
