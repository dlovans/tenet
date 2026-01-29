/**
 * JSON-logic operators for the Tenet VM.
 * All operators are nil-safe: operations on nil/undefined return appropriate defaults.
 */

import type { EvalState } from './types.js';

/** Operator function signature */
type OperatorFn = (args: unknown, state: EvalState, resolve: ResolveFn) => unknown;
type ResolveFn = (node: unknown, state: EvalState) => unknown;

/**
 * Convert a value to a number if possible.
 */
export function toFloat(v: unknown): [number, boolean] {
    if (v === null || v === undefined) {
        return [0, false];
    }
    if (typeof v === 'number') {
        return [v, true];
    }
    if (typeof v === 'string') {
        // Don't auto-convert strings to numbers
        return [0, false];
    }
    return [0, false];
}

/**
 * Parse a date value (string or Date).
 * Supports ISO 8601 formats.
 */
export function parseDate(v: unknown): [Date, boolean] {
    if (v === null || v === undefined) {
        return [new Date(0), false];
    }
    if (v instanceof Date) {
        return [v, true];
    }
    if (typeof v === 'string') {
        const d = new Date(v);
        if (!isNaN(d.getTime())) {
            return [d, true];
        }
    }
    return [new Date(0), false];
}

/**
 * Determine if a value is "truthy" in JSON-logic terms.
 * nil, false, 0, and "" are falsy. Everything else is truthy.
 */
export function isTruthy(value: unknown): boolean {
    if (value === null || value === undefined) {
        return false;
    }
    if (typeof value === 'boolean') {
        return value;
    }
    if (typeof value === 'number') {
        return value !== 0;
    }
    if (typeof value === 'string') {
        return value !== '';
    }
    if (Array.isArray(value)) {
        return value.length > 0;
    }
    if (typeof value === 'object') {
        return Object.keys(value).length > 0;
    }
    return true;
}

/**
 * Compare two values for equality with type coercion.
 * nil == nil is true, nil == anything_else is false.
 */
export function compareEqual(a: unknown, b: unknown): boolean {
    if (a === null || a === undefined) {
        return b === null || b === undefined;
    }
    if (b === null || b === undefined) {
        return false;
    }

    // Try numeric comparison if both can be numbers
    const [aNum, aOk] = toFloat(a);
    const [bNum, bOk] = toFloat(b);
    if (aOk && bOk) {
        return aNum === bNum;
    }

    // String comparison
    return String(a) === String(b);
}

/**
 * Resolve arguments array, handling single values and missing args.
 */
function resolveArgs(args: unknown, expected: number, resolve: ResolveFn, state: EvalState): unknown[] {
    const result: unknown[] = new Array(expected).fill(null);

    if (!Array.isArray(args)) {
        // Single value case (e.g., {"not": true})
        if (expected > 0) {
            result[0] = resolve(args, state);
        }
        return result;
    }

    for (let i = 0; i < expected && i < args.length; i++) {
        result[i] = resolve(args[i], state);
    }
    return result;
}

/**
 * Get variable value from schema definitions or current element context.
 */
function getVar(path: string, state: EvalState): unknown {
    // Empty path returns current element context (for some/all/none)
    if (path === '') {
        return state.currentElement;
    }

    const parts = path.split('.');

    // Check definitions first
    const def = state.schema.definitions[parts[0]];
    if (def) {
        if (parts.length === 1) {
            return def.value;
        }
        // Nested access into the value
        return accessPath(def.value, parts.slice(1));
    }

    // Check derived state
    if (state.schema.state_model?.derived) {
        const derived = state.schema.state_model.derived[parts[0]];
        if (derived) {
            // Note: derived values should already be computed by this point
            // This is a fallback for direct access
            return undefined;
        }
    }

    return undefined;
}

/**
 * Access nested path in an object.
 */
function accessPath(value: unknown, parts: string[]): unknown {
    if (parts.length === 0 || value === null || value === undefined) {
        return value;
    }

    if (typeof value === 'object' && !Array.isArray(value)) {
        const obj = value as Record<string, unknown>;
        const next = obj[parts[0]];
        return accessPath(next, parts.slice(1));
    }

    return undefined;
}

// ============================================================
// Operator implementations
// ============================================================

const operators: Record<string, OperatorFn> = {
    // === Variable Access ===
    'var': (args, state) => {
        const path = typeof args === 'string' ? args : '';
        return getVar(path, state);
    },

    // === Comparison Operators ===
    '==': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        return compareEqual(a[0], a[1]);
    },

    '!=': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        return !compareEqual(a[0], a[1]);
    },

    '>': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return false;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        return aOk && bOk && aNum > bNum;
    },

    '<': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return false;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        return aOk && bOk && aNum < bNum;
    },

    '>=': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return false;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        return aOk && bOk && aNum >= bNum;
    },

    '<=': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return false;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        return aOk && bOk && aNum <= bNum;
    },

    // === Logical Operators ===
    'and': (args, state, resolve) => {
        if (!Array.isArray(args)) {
            return isTruthy(resolve(args, state));
        }
        for (const arg of args) {
            if (!isTruthy(resolve(arg, state))) {
                return false;
            }
        }
        return true;
    },

    'or': (args, state, resolve) => {
        if (!Array.isArray(args)) {
            return isTruthy(resolve(args, state));
        }
        for (const arg of args) {
            if (isTruthy(resolve(arg, state))) {
                return true;
            }
        }
        return false;
    },

    'not': (args, state, resolve) => {
        const a = resolveArgs(args, 1, resolve, state);
        return !isTruthy(a[0]);
    },

    '!': (args, state, resolve) => {
        const a = resolveArgs(args, 1, resolve, state);
        return !isTruthy(a[0]);
    },

    'if': (args, state, resolve) => {
        if (!Array.isArray(args) || args.length < 2) {
            return null;
        }

        // Process condition-then pairs
        for (let i = 0; i + 1 < args.length; i += 2) {
            const condition = resolve(args[i], state);
            if (isTruthy(condition)) {
                return resolve(args[i + 1], state);
            }
        }

        // Else clause (odd number of elements = has else)
        if (args.length % 2 === 1) {
            return resolve(args[args.length - 1], state);
        }

        return null;
    },

    // === Arithmetic Operators ===
    '+': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return null;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        if (!aOk || !bOk) {
            return null;
        }
        return aNum + bNum;
    },

    '-': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return null;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        if (!aOk || !bOk) {
            return null;
        }
        return aNum - bNum;
    },

    '*': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return null;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        if (!aOk || !bOk) {
            return null;
        }
        return aNum * bNum;
    },

    '/': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        if (a[0] === null || a[0] === undefined || a[1] === null || a[1] === undefined) {
            return null;
        }
        const [aNum, aOk] = toFloat(a[0]);
        const [bNum, bOk] = toFloat(a[1]);
        if (!aOk || !bOk || bNum === 0) {
            return null;
        }
        return aNum / bNum;
    },

    // === Date Operators ===
    'before': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        const [aDate, aOk] = parseDate(a[0]);
        const [bDate, bOk] = parseDate(a[1]);
        if (!aOk || !bOk) {
            return false;
        }
        return aDate.getTime() < bDate.getTime();
    },

    'after': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        const [aDate, aOk] = parseDate(a[0]);
        const [bDate, bOk] = parseDate(a[1]);
        if (!aOk || !bOk) {
            return false;
        }
        return aDate.getTime() > bDate.getTime();
    },

    // === Collection Operators ===
    'in': (args, state, resolve) => {
        const a = resolveArgs(args, 2, resolve, state);
        const needle = a[0];
        const haystack = a[1];

        if (needle === null || needle === undefined || haystack === null || haystack === undefined) {
            return false;
        }

        if (Array.isArray(haystack)) {
            for (const item of haystack) {
                if (compareEqual(needle, item)) {
                    return true;
                }
            }
            return false;
        }

        if (typeof haystack === 'string' && typeof needle === 'string') {
            return haystack.includes(needle);
        }

        return false;
    },

    'some': (args, state, resolve) => {
        if (!Array.isArray(args) || args.length < 2) {
            return false;
        }

        // First arg is the array (resolve it)
        const collection = resolve(args[0], state);
        if (!Array.isArray(collection) || collection.length === 0) {
            return false;
        }

        // Second arg is the condition
        const condition = args[1];

        // Check if any element satisfies the condition
        for (const item of collection) {
            const oldContext = state.currentElement;
            state.currentElement = item;
            const result = isTruthy(resolve(condition, state));
            state.currentElement = oldContext;
            if (result) {
                return true;
            }
        }
        return false;
    },

    'all': (args, state, resolve) => {
        if (!Array.isArray(args) || args.length < 2) {
            return false;
        }

        // First arg is the array (resolve it)
        const collection = resolve(args[0], state);
        if (!Array.isArray(collection)) {
            return false;
        }

        // Empty array returns true for "all" (vacuous truth)
        if (collection.length === 0) {
            return true;
        }

        // Second arg is the condition
        const condition = args[1];

        // Check if all elements satisfy the condition
        for (const item of collection) {
            const oldContext = state.currentElement;
            state.currentElement = item;
            const result = isTruthy(resolve(condition, state));
            state.currentElement = oldContext;
            if (!result) {
                return false;
            }
        }
        return true;
    },

    'none': (args, state, resolve) => {
        if (!Array.isArray(args) || args.length < 2) {
            return false;
        }

        // First arg is the array (resolve it)
        const collection = resolve(args[0], state);
        if (!Array.isArray(collection)) {
            return false;
        }

        // Empty array returns true for "none"
        if (collection.length === 0) {
            return true;
        }

        // Second arg is the condition
        const condition = args[1];

        // Check that no element satisfies the condition
        for (const item of collection) {
            const oldContext = state.currentElement;
            state.currentElement = item;
            const result = isTruthy(resolve(condition, state));
            state.currentElement = oldContext;
            if (result) {
                return false;
            }
        }
        return true;
    },
};

/**
 * Apply an operator to its arguments.
 */
export function applyOperator(
    op: string,
    args: unknown,
    state: EvalState,
    resolve: ResolveFn
): unknown {
    const fn = operators[op];
    if (!fn) {
        // Unknown operator - return null
        return null;
    }
    return fn(args, state, resolve);
}
