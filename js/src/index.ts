/**
 * Tenet - Declarative Logic VM for JSON Schemas
 *
 * This module provides a pure TypeScript implementation of the Tenet VM.
 * Works in both browser and Node.js environments with no WASM dependencies.
 */

// Re-export lint functions (pure TypeScript)
export { lint, isTenetSchema, SCHEMA_URL } from './lint.js';
export type { LintIssue, LintResult } from './lint.js';

// Type definitions
export interface TenetResult {
    result?: TenetSchema;
    error?: string;
}

export interface TenetVerifyResult {
    valid: boolean;
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

export interface ValidationError {
    field_id?: string;
    rule_id?: string;
    message: string;
    law_ref?: string;
}

// Import core engine functions
import { run as coreRun, verify as coreVerify } from './core/engine.js';

/**
 * Initialize the Tenet VM.
 * This is a no-op in the pure TypeScript implementation (kept for backwards compatibility).
 *
 * @deprecated No longer needed - the VM is ready immediately after import.
 */
export async function init(_wasmPath?: string): Promise<void> {
    // No-op: pure TypeScript implementation doesn't need initialization
    return Promise.resolve();
}

/**
 * Run the Tenet VM on a schema.
 *
 * @param schema - The schema object or JSON string
 * @param date - Effective date (ISO 8601 string or Date object)
 * @returns The transformed schema with computed state, errors, and status
 */
export function run(schema: TenetSchema | string, date: Date | string = new Date()): TenetResult {
    return coreRun(schema, date);
}

/**
 * Verify that a schema transformation is legal.
 * Re-runs the logic on the old schema and compares with the new schema.
 *
 * @param newSchema - The transformed schema
 * @param oldSchema - The original schema
 * @returns Whether the transformation is valid
 */
export function verify(
    newSchema: TenetSchema | string,
    oldSchema: TenetSchema | string
): TenetVerifyResult {
    return coreVerify(newSchema, oldSchema);
}

/**
 * Check if the VM is ready.
 * Always returns true in the pure TypeScript implementation.
 */
export function isReady(): boolean {
    return true;
}
