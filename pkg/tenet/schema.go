// Package tenet provides a declarative logic VM for JSON schemas.
// It handles temporal routing, reactive state, and validation for legal compliance use cases.
package tenet

// Schema is the root container for a Tenet document.
// Only `definitions` is required. All other fields are optional.
type Schema struct {
	Protocol     string                  `json:"protocol,omitempty"`     // Protocol identifier (optional)
	SchemaID     string                  `json:"schema_id,omitempty"`    // Schema identifier (optional)
	Version      string                  `json:"version,omitempty"`      // Schema version (optional)
	ValidFrom    string                  `json:"valid_from,omitempty"`   // Effective date (optional)
	Definitions  map[string]*Definition  `json:"definitions"`            // REQUIRED: Field definitions
	Attestations map[string]*Attestation `json:"attestations,omitempty"` // Optional: Legal attestations
	LogicTree    []*Rule                 `json:"logic_tree,omitempty"`   // Optional: Reactive rules
	TemporalMap  []*TemporalBranch       `json:"temporal_map,omitempty"` // Optional: Version routing
	StateModel   *StateModel             `json:"state_model,omitempty"`  // Optional: Derived values

	// Output fields (populated by Run)
	Errors []ValidationError `json:"errors,omitempty"`
	Status DocStatus         `json:"status,omitempty"`
}

// DocStatus represents the validation state of a document.
type DocStatus string

const (
	StatusReady      DocStatus = "READY"      // All validations pass, all required fields present
	StatusIncomplete DocStatus = "INCOMPLETE" // Missing required fields or attestations
	StatusInvalid    DocStatus = "INVALID"    // Type errors or rule violations
)

// Definition represents a typed field with value and metadata.
// Value is kept as nil when not set (distinguishes "unknown" from "zero").
type Definition struct {
	Type     string   `json:"type"`               // "string", "number", "select", "attestation", "date", "boolean", "currency"
	Value    any      `json:"value,omitempty"`    // Current value (nil = not set)
	Options  []string `json:"options,omitempty"`  // For "select" type
	Label    string   `json:"label,omitempty"`    // Human-readable label
	Required bool     `json:"required,omitempty"` // Is this field required?
	Readonly bool     `json:"readonly,omitempty"` // True = computed, False = user-editable
	Visible  bool     `json:"visible"`            // UI visibility (default true)

	// Numeric constraints (for "number" and "currency" types)
	Min  *float64 `json:"min,omitempty"`  // Minimum allowed value (nil = no minimum)
	Max  *float64 `json:"max,omitempty"`  // Maximum allowed value (nil = no maximum)
	Step *float64 `json:"step,omitempty"` // Step increment for UI (e.g., 0.01 for currency)

	// String constraints
	MinLength *int   `json:"min_length,omitempty"` // Minimum string length
	MaxLength *int   `json:"max_length,omitempty"` // Maximum string length
	Pattern   string `json:"pattern,omitempty"`    // Regex pattern for validation

	// UI metadata that can be modified by rules
	UIClass   string `json:"ui_class,omitempty"`   // CSS class hint
	UIMessage string `json:"ui_message,omitempty"` // Inline message/hint
}

// Rule represents a logic tree node with a when-then structure.
// Each rule is anchored to a legal citation for audit purposes.
type Rule struct {
	ID           string         `json:"id"`
	LawRef       string         `json:"law_ref,omitempty"`       // Legal citation (e.g., "GDPR Art. 33(1)")
	LogicVersion string         `json:"logic_version,omitempty"` // Which temporal branch this belongs to
	When         map[string]any `json:"when"`                    // JSON-logic condition
	Then         *Action        `json:"then"`
	Disabled     bool           `json:"disabled,omitempty"` // Set by prune() for inactive rules
}

// Action represents what happens when a rule's condition is true.
type Action struct {
	Set      map[string]any `json:"set,omitempty"`       // Values to set in definitions
	UIModify map[string]any `json:"ui_modify,omitempty"` // UI metadata changes (visible, ui_class, etc.)
	ErrorMsg string         `json:"error_msg,omitempty"` // Validation error to emit
}

// TemporalBranch routes logic based on effective dates.
// Supports bitemporal logic with valid ranges.
type TemporalBranch struct {
	ValidRange   [2]*string `json:"valid_range"`   // [start, end?] ISO dates (nil end = open-ended)
	LogicVersion string     `json:"logic_version"` // Version identifier (e.g., "v1.2_legacy", "v2.0_current")
	Status       string     `json:"status"`        // "ACTIVE", "ARCHIVED"
}

// StateModel defines inputs and derived (computed) values.
// Derived values use JSON-logic expressions, evaluated reactively.
type StateModel struct {
	Inputs  []string               `json:"inputs"`  // Fields that trigger recomputation
	Derived map[string]*DerivedDef `json:"derived"` // Computed fields
}

// DerivedDef is a computed field whose value is determined by a JSON-logic expression.
type DerivedDef struct {
	Eval map[string]any `json:"eval"` // JSON-logic expression (uses same syntax as Rule.When)
}

// ValidationError represents a validation failure tied to a field and law reference.
type ValidationError struct {
	FieldID string `json:"field_id,omitempty"` // Which definition failed
	RuleID  string `json:"rule_id,omitempty"`  // Which rule emitted this error
	Message string `json:"message"`            // Human-readable error
	LawRef  string `json:"law_ref,omitempty"`  // Legal citation for the rule
}

// Attestation represents a legally-binding signature requirement.
// The VM validates attestations but does not perform signing — that's the app's job.
type Attestation struct {
	LawRef       string `json:"law_ref,omitempty"`       // Legal citation (e.g., "OSHA Section 1910.12")
	Statement    string `json:"statement"`               // What they're signing
	RequiredRole string `json:"required_role,omitempty"` // Who can sign (e.g., "Compliance_Officer")
	Provider     string `json:"provider,omitempty"`      // "DocuSign", "OpenID", "Manual"
	Required     bool   `json:"required,omitempty"`      // Is signature required for READY?

	// Filled by the orchestrating application, validated by VM
	Signed   bool      `json:"signed"`             // Has the attestation been signed?
	Evidence *Evidence `json:"evidence,omitempty"` // Proof of signing (filled by app)

	// Actions to execute when signed: true (processed during Run)
	OnSign *Action `json:"on_sign,omitempty"`
}

// Evidence contains the audit trail from a signing provider.
// The VM validates this is populated when signed is true, but doesn't verify the signature.
type Evidence struct {
	ProviderAuditID string `json:"provider_audit_id,omitempty"` // External ID from DocuSign, etc.
	Timestamp       string `json:"timestamp,omitempty"`         // ISO 8601 when signed
	SignerID        string `json:"signer_id,omitempty"`         // Who signed (email, user ID)
	LogicVersion    string `json:"logic_version,omitempty"`     // Schema version at signing time
}
