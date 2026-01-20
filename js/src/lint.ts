/**
 * Tenet Linter - Static Analysis for Tenet Schemas
 * 
 * Pure TypeScript implementation - no WASM required.
 * Can be used in browsers, Node.js, and edge runtimes.
 */

import type { TenetSchema, Rule } from './index';

// JSON Schema URL for IDE integration
export const SCHEMA_URL = 'https://tenet.dev/schema/v1.json';

export interface LintIssue {
    severity: 'error' | 'warning' | 'info';
    field?: string;
    rule?: string;
    message: string;
}

export interface LintResult {
    valid: boolean;
    issues: LintIssue[];
}

/**
 * Perform static analysis on a Tenet schema without executing it.
 * Detects potential issues like undefined variables, cycles, and missing fields.
 * 
 * @param schema - The schema object or JSON string
 * @returns Lint result with issues found
 */
export function lint(schema: TenetSchema | string): LintResult {
    let parsed: TenetSchema;

    try {
        parsed = typeof schema === 'string' ? JSON.parse(schema) : schema;
    } catch (e) {
        return {
            valid: false,
            issues: [{ severity: 'error', message: `Parse error: ${e}` }]
        };
    }

    const result: LintResult = {
        valid: true,
        issues: []
    };

    // Collect all defined field names
    const definedFields = new Set<string>();

    if (parsed.definitions) {
        for (const name of Object.keys(parsed.definitions)) {
            definedFields.add(name);
        }
    }

    // Add derived fields
    if (parsed.state_model?.derived) {
        for (const name of Object.keys(parsed.state_model.derived)) {
            definedFields.add(name);
        }
    }

    // Check 1: Schema identification
    if (!parsed.protocol && !parsed['$schema' as keyof TenetSchema]) {
        result.issues.push({
            severity: 'info',
            message: `Consider adding "protocol": "Tenet_v1.0" or "$schema": "${SCHEMA_URL}" for IDE support`
        });
    }

    // Check 2: Undefined variables in logic tree
    if (parsed.logic_tree) {
        for (const rule of parsed.logic_tree) {
            if (!rule) continue;

            const varsInWhen = extractVars(rule.when);
            for (const v of varsInWhen) {
                if (!definedFields.has(v)) {
                    addError(result, v, rule.id, `Undefined variable '${v}' in rule condition`);
                }
            }
        }
    }

    // Check 3: Potential cycles (fields set by multiple rules)
    const fieldSetBy = new Map<string, string[]>();

    if (parsed.logic_tree) {
        for (const rule of parsed.logic_tree) {
            if (!rule?.then?.set) continue;

            for (const field of Object.keys(rule.then.set)) {
                const rules = fieldSetBy.get(field) || [];
                rules.push(rule.id);
                fieldSetBy.set(field, rules);
            }
        }
    }

    for (const [field, rules] of fieldSetBy) {
        if (rules.length > 1) {
            addWarning(result, field, '',
                `Field '${field}' may be set by multiple rules: [${rules.sort().join(', ')}] (potential cycle)`);
        }
    }

    // Check 4: Temporal map validation
    if (parsed.temporal_map) {
        for (let i = 0; i < parsed.temporal_map.length; i++) {
            const branch = parsed.temporal_map[i];
            if (!branch) continue;

            if (!branch.logic_version) {
                addWarning(result, '', '', `Temporal branch ${i} has no logic_version`);
            }
        }
    }

    // Check 5: Empty type in definitions
    if (parsed.definitions) {
        for (const [name, def] of Object.entries(parsed.definitions)) {
            if (!def) continue;

            if (!def.type) {
                addWarning(result, name, '', `Definition '${name}' has no type specified`);
            }
        }
    }

    return result;
}

/**
 * Check if a schema is a valid Tenet schema (basic detection).
 * Useful for IDE integration to detect Tenet files.
 */
export function isTenetSchema(schema: unknown): schema is TenetSchema {
    if (typeof schema !== 'object' || schema === null) return false;

    const obj = schema as Record<string, unknown>;

    // Check for $schema URL
    if (obj['$schema'] === SCHEMA_URL) return true;

    // Check for protocol field
    if (typeof obj.protocol === 'string' && obj.protocol.startsWith('Tenet')) return true;

    // Check for definitions + logic_tree structure
    if (obj.definitions && typeof obj.definitions === 'object') return true;

    return false;
}

// Helper functions

function addError(result: LintResult, field: string, rule: string, message: string): void {
    result.valid = false;
    result.issues.push({ severity: 'error', field, rule, message });
}

function addWarning(result: LintResult, field: string, rule: string, message: string): void {
    result.issues.push({ severity: 'warning', field, rule, message });
}

/**
 * Extract all variable references from a JSON-logic expression.
 */
function extractVars(node: unknown): string[] {
    if (node === null || node === undefined) return [];

    const vars: string[] = [];

    if (typeof node === 'object') {
        if (Array.isArray(node)) {
            for (const elem of node) {
                vars.push(...extractVars(elem));
            }
        } else {
            const obj = node as Record<string, unknown>;

            // Check if this is a var reference
            if ('var' in obj && typeof obj.var === 'string') {
                // Get root variable name (before any dot notation)
                const varName = obj.var.split('.')[0];
                vars.push(varName);
            }

            // Recurse into all values
            for (const val of Object.values(obj)) {
                vars.push(...extractVars(val));
            }
        }
    }

    return vars;
}
