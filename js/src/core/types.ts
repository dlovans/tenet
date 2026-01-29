/**
 * Internal types for the Tenet VM core.
 * Public types are re-exported from index.ts
 */

// Re-export public types for internal use
export type {
    TenetSchema,
    TenetResult,
    TenetVerifyResult,
    Definition,
    Rule,
    Action,
    TemporalBranch,
    StateModel,
    DerivedDef,
    ValidationError,
    Attestation,
    Evidence,
} from '../index.js';

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
    schema: import('../index.js').TenetSchema;
    /** Effective date for temporal routing */
    effectiveDate: Date;
    /** Tracks which fields were set by which rule (cycle detection) */
    fieldsSet: Map<string, string>;
    /** Current element context for some/all/none operators */
    currentElement?: unknown;
    /** Accumulated validation errors */
    errors: import('../index.js').ValidationError[];
}

/**
 * Document status values
 */
export type DocStatus = 'READY' | 'INCOMPLETE' | 'INVALID';
