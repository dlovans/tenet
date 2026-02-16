/**
 * Tenet - Declarative Logic VM for JSON Schemas
 *
 * This module provides a pure TypeScript implementation of the Tenet VM.
 * Works in both browser and Node.js environments with no WASM dependencies.
 */

// Re-export public types from the single source of truth
export type {
    TenetSchema,
    TenetResult,
    TenetVerifyResult,
    VerifyIssue,
    VerifyIssueCode,
    Definition,
    Rule,
    Action,
    TemporalBranch,
    StateModel,
    DerivedDef,
    ValidationError,
    Evidence,
    Attestation,
    ErrorKind,
} from './core/types.js';

// Import core engine functions
import type { TenetSchema, TenetResult, TenetVerifyResult } from './core/types.js';
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
