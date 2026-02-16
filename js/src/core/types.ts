/**
 * All type definitions for the Tenet VM.
 * This is the single source of truth — other modules re-export from here.
 */

export interface TenetResult {
    result?: TenetSchema;
    error?: string;
}

/**
 * Machine-parseable codes for verification issues.
 * UI layers map these to customer-friendly messages; the VM never decides presentation.
 */
export type VerifyIssueCode =
    | 'unknown_field'
    | 'computed_mismatch'
    | 'attestation_unsigned'
    | 'attestation_no_evidence'
    | 'attestation_no_timestamp'
    | 'status_mismatch'
    | 'convergence_failed'
    | 'internal_error';

/**
 * A single structured problem found during verification.
 */
export interface VerifyIssue {
    code: VerifyIssueCode;
    field_id?: string;
    message: string;
    expected?: unknown;
    claimed?: unknown;
}

export interface TenetVerifyResult {
    valid: boolean;
    status?: string;
    issues?: VerifyIssue[];
    schema?: TenetSchema;
    error?: string;
}

export interface Evidence {
    provider_audit_id?: string;
    timestamp?: string;
    signer_id?: string;
    logic_version?: string;
}

export interface Attestation {
    statement: string;
    law_ref?: string;
    required_role?: string;
    provider?: string;
    required?: boolean;
    signed?: boolean;
    evidence?: Evidence;
    on_sign?: Action;
}

export interface TenetSchema {
    protocol?: string;
    schema_id?: string;
    version?: string;
    valid_from?: string;
    definitions: Record<string, Definition>;
    logic_tree?: Rule[];
    temporal_map?: TemporalBranch[];
    state_model?: StateModel;
    errors?: ValidationError[];
    status?: 'READY' | 'INCOMPLETE' | 'INVALID';
    attestations?: Record<string, Attestation>;
}

export interface Definition {
    type: 'string' | 'number' | 'boolean' | 'select' | 'date' | 'attestation' | 'currency';
    value?: unknown;
    options?: string[];
    label?: string;
    required?: boolean;
    readonly?: boolean;
    visible?: boolean;
    min?: number;
    max?: number;
    step?: number;
    min_length?: number;
    max_length?: number;
    pattern?: string;
    ui_class?: string;
    ui_message?: string;
}

export interface Rule {
    id: string;
    law_ref?: string;
    logic_version?: string;
    when: Record<string, unknown>;
    then: Action;
    disabled?: boolean;
}

export interface Action {
    set?: Record<string, unknown>;
    ui_modify?: Record<string, unknown>;
    error_msg?: string;
}

export interface TemporalBranch {
    valid_range: [string | null, string | null];
    logic_version: string;
    status: 'ACTIVE' | 'ARCHIVED';
}

export interface StateModel {
    inputs: string[];
    derived: Record<string, DerivedDef>;
}

export interface DerivedDef {
    eval: Record<string, unknown>;
}

export type ErrorKind =
    | 'type_mismatch'
    | 'missing_required'
    | 'constraint_violation'
    | 'attestation_incomplete'
    | 'runtime_warning'
    | 'cycle_detected';

export interface ValidationError {
    field_id?: string;
    rule_id?: string;
    kind: ErrorKind;
    message: string;
    law_ref?: string;
}

/**
 * Evaluation context for collection operators (some/all/none).
 * When iterating over an array, provides access to the current element.
 */
export interface EvalContext {
    /** Current element being evaluated */
    item: unknown;
}

/**
 * Internal state during rule evaluation.
 * Passed through the resolver and operators.
 */
export interface EvalState {
    /** The schema being evaluated (mutable copy) */
    schema: TenetSchema;
    /** Effective date for temporal routing */
    effectiveDate: Date;
    /** Tracks which fields were set by which rule (cycle detection) */
    fieldsSet: Map<string, string>;
    /** Current element context for some/all/none operators */
    currentElement?: unknown;
    /** Accumulated validation errors */
    errors: ValidationError[];
    /** Cycle detection for derived fields */
    derivedInProgress: Set<string>;
}

/**
 * Document status values
 */
export type DocStatus = 'READY' | 'INCOMPLETE' | 'INVALID';
