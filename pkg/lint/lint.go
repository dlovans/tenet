// Package lint provides static analysis for Tenet schemas.
// It detects potential issues without executing the schema.
package lint

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Issue represents a problem found during static analysis.
type Issue struct {
	Severity string `json:"severity"` // "error", "warning", "info"
	Field    string `json:"field,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Message  string `json:"message"`
}

// Result contains all issues found by the linter.
type Result struct {
	Valid  bool    `json:"valid"`
	Issues []Issue `json:"issues"`
}

// Schema types (minimal subset for linting - no execution logic)

type schema struct {
	Definitions  map[string]*definition  `json:"definitions"`
	LogicTree    []*rule                 `json:"logic_tree,omitempty"`
	TemporalMap  []*temporalBranch       `json:"temporal_map,omitempty"`
	StateModel   *stateModel             `json:"state_model,omitempty"`
	Attestations map[string]*attestation `json:"attestations,omitempty"`
}

type definition struct {
	Type string `json:"type,omitempty"`
}

type rule struct {
	ID   string  `json:"id,omitempty"`
	When any     `json:"when,omitempty"`
	Then *action `json:"then,omitempty"`
}

type action struct {
	Set map[string]any `json:"set,omitempty"`
}

type temporalBranch struct {
	LogicVersion string `json:"logic_version,omitempty"`
}

type stateModel struct {
	Derived map[string]*derivedDef `json:"derived,omitempty"`
}

type derivedDef struct {
	Eval any `json:"eval,omitempty"`
}

type attestation struct {
	Statement string `json:"statement,omitempty"`
}

// Run performs static analysis on a schema without executing it.
// Detects potential issues like undefined variables, type mismatches, and cycles.
func Run(jsonText string) (*Result, error) {
	var s schema
	if err := json.Unmarshal([]byte(jsonText), &s); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	result := &Result{
		Valid:  true,
		Issues: make([]Issue, 0),
	}

	// Collect all defined field names
	definedFields := make(map[string]bool)
	for name := range s.Definitions {
		definedFields[name] = true
	}

	// Add derived fields
	if s.StateModel != nil && s.StateModel.Derived != nil {
		for name := range s.StateModel.Derived {
			definedFields[name] = true
		}
	}

	// Check 1: Undefined variables in logic tree
	for _, rule := range s.LogicTree {
		if rule == nil {
			continue
		}

		// Check variables in "when" condition
		varsInWhen := extractVars(rule.When)
		for _, v := range varsInWhen {
			if !definedFields[v] {
				result.addError(v, rule.ID, fmt.Sprintf("undefined variable '%s' in rule condition", v))
			}
		}
	}

	// Check 2: Potential cycles (fields set by multiple rules)
	fieldSetBy := make(map[string][]string)
	for _, rule := range s.LogicTree {
		if rule == nil || rule.Then == nil || rule.Then.Set == nil {
			continue
		}
		for field := range rule.Then.Set {
			fieldSetBy[field] = append(fieldSetBy[field], rule.ID)
		}
	}

	for field, rules := range fieldSetBy {
		if len(rules) > 1 {
			sort.Strings(rules)
			result.addWarning(field, "", fmt.Sprintf(
				"field '%s' may be set by multiple rules: %v (potential cycle or conflict)",
				field, rules))
		}
	}

	// Check 3: Temporal map status validation
	for i, branch := range s.TemporalMap {
		if branch == nil {
			continue
		}
		if branch.LogicVersion == "" {
			result.addWarning("", "", fmt.Sprintf(
				"temporal branch %d has no logic_version", i))
		}
	}

	// Check 4: Empty required fields in definitions
	for name, def := range s.Definitions {
		if def == nil {
			continue
		}
		if def.Type == "" {
			result.addWarning(name, "", fmt.Sprintf("definition '%s' has no type specified", name))
		}
	}

	// Check 5: Attestations without statements
	for name, att := range s.Attestations {
		if att == nil {
			continue
		}
		if att.Statement == "" {
			result.addWarning(name, "", fmt.Sprintf("attestation '%s' has no statement", name))
		}
	}

	return result, nil
}

func (r *Result) addError(field, rule, message string) {
	r.Valid = false
	r.Issues = append(r.Issues, Issue{
		Severity: "error",
		Field:    field,
		Rule:     rule,
		Message:  message,
	})
}

func (r *Result) addWarning(field, rule, message string) {
	r.Issues = append(r.Issues, Issue{
		Severity: "warning",
		Field:    field,
		Rule:     rule,
		Message:  message,
	})
}

// extractVars recursively finds all {"var": "name"} references in a JSON-logic tree.
func extractVars(node any) []string {
	if node == nil {
		return nil
	}

	var vars []string

	switch v := node.(type) {
	case map[string]any:
		// Check if this is a var reference
		if varName, ok := v["var"]; ok {
			if name, isString := varName.(string); isString {
				// Get the root variable name (before any dot notation)
				parts := splitFirst(name, ".")
				vars = append(vars, parts[0])
			}
		}
		// Recurse into all values
		for _, val := range v {
			vars = append(vars, extractVars(val)...)
		}

	case []any:
		for _, elem := range v {
			vars = append(vars, extractVars(elem)...)
		}
	}

	return vars
}

// splitFirst splits a string by the first occurrence of sep.
func splitFirst(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
