/**
 * Tenet - Declarative Logic VM for JSON Schemas
 * 
 * This module provides a JavaScript/TypeScript wrapper around the Tenet WASM binary.
 * Works in both browser and Node.js environments.
 */

// Re-export lint functions (pure TypeScript, no WASM needed)
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

// Global references set by WASM
declare global {
    var TenetRun: (json: string, date: string) => TenetResult;
    var TenetVerify: (newJson: string, oldJson: string) => TenetVerifyResult;
    var Go: new () => GoInstance;
}

interface GoInstance {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
}

let wasmReady = false;
let wasmReadyPromise: Promise<void> | null = null;

/**
 * Initialize the Tenet WASM module.
 * Must be called before using run() or verify().
 * 
 * @param wasmPath - Path or URL to tenet.wasm file
 */
export async function init(wasmPath: string = './tenet.wasm'): Promise<void> {
    if (wasmReady) return;
    if (wasmReadyPromise) return wasmReadyPromise;

    wasmReadyPromise = loadWasm(wasmPath);
    await wasmReadyPromise;
    wasmReady = true;
}

async function loadWasm(wasmPath: string): Promise<void> {
    // Detect environment
    const isBrowser = typeof window !== 'undefined';
    const isNode = typeof process !== 'undefined' && process.versions?.node;

    if (isBrowser) {
        // Browser environment
        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch(wasmPath),
            go.importObject
        );
        go.run(result.instance);
    } else if (isNode) {
        // Node.js environment
        const fs = await import('fs');
        const path = await import('path');
        const { fileURLToPath } = await import('url');
        const { createRequire } = await import('module');

        // ESM-compatible __dirname and require
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        const require = createRequire(import.meta.url);

        // Load wasm_exec.js (Go's JS runtime)
        const wasmExecPath = path.resolve(__dirname, '../wasm/wasm_exec.js');
        require(wasmExecPath);

        const go = new Go();
        const wasmBuffer = fs.readFileSync(wasmPath);
        const result = await WebAssembly.instantiate(wasmBuffer, go.importObject);
        go.run(result.instance);
    } else {
        throw new Error('Unsupported environment');
    }
}

/**
 * Run the Tenet VM on a schema.
 * 
 * @param schema - The schema object or JSON string
 * @param date - Effective date (ISO 8601 string or Date object)
 * @returns The transformed schema with computed state, errors, and status
 */
export function run(schema: TenetSchema | string, date: Date | string = new Date()): TenetResult {
    if (!wasmReady) {
        throw new Error('Tenet not initialized. Call init() first.');
    }

    const jsonStr = typeof schema === 'string' ? schema : JSON.stringify(schema);
    const dateStr = date instanceof Date ? date.toISOString() : date;

    return globalThis.TenetRun(jsonStr, dateStr);
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
    if (!wasmReady) {
        throw new Error('Tenet not initialized. Call init() first.');
    }

    const newJson = typeof newSchema === 'string' ? newSchema : JSON.stringify(newSchema);
    const oldJson = typeof oldSchema === 'string' ? oldSchema : JSON.stringify(oldSchema);

    return globalThis.TenetVerify(newJson, oldJson);
}

/**
 * Check if the WASM module is ready.
 */
export function isReady(): boolean {
    return wasmReady;
}
