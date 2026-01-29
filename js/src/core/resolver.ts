/**
 * JSON-logic expression resolver.
 * Recursively evaluates JSON-logic expressions and returns their values.
 */

import type { EvalState } from './types.js';
import { applyOperator } from './operators.js';

/**
 * Resolve any JSON-logic node and return its value.
 * This is the recursive core of the VM.
 * It is nil-safe: operations on nil values return appropriate defaults.
 */
export function resolve(node: unknown, state: EvalState): unknown {
    if (node === null || node === undefined) {
        return null;
    }

    // Object - could be an operator or a literal map
    if (typeof node === 'object' && !Array.isArray(node)) {
        const obj = node as Record<string, unknown>;
        const keys = Object.keys(obj);

        // Single key = operator: {"==": [a, b]} or {"var": "field_name"}
        if (keys.length === 1) {
            const op = keys[0];
            const args = obj[op];
            return applyOperator(op, args, state, resolve);
        }

        // Multi-key object is treated as a literal (return as-is)
        return node;
    }

    // Array literal - resolve each element
    if (Array.isArray(node)) {
        return node.map(elem => resolve(elem, state));
    }

    // Primitives (string, number, boolean) - return as-is
    return node;
}
