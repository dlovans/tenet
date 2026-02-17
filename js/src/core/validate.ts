/**
 * Definition validation and status determination.
 * Validates types, constraints, and required fields.
 */

import type { EvalState, Definition, ErrorKind, DocStatus, Action } from './types.js';
import { toFloat, parseDate } from './operators.js';

/**
 * Add an error to the state's error list.
 */
export function addError(
    state: EvalState,
    fieldId: string,
    ruleId: string,
    kind: ErrorKind,
    message: string,
    lawRef?: string
): void {
    state.errors.push({
        field_id: fieldId || undefined,
        rule_id: ruleId || undefined,
        kind,
        message,
        law_ref: lawRef || undefined,
    });
}

/**
 * Check if a value is one of the allowed options.
 */
function isValidOption(value: string, options: string[] | undefined): boolean {
    if (!options) {
        return true; // No restrictions
    }
    return options.includes(value);
}

/**
 * Validate numeric constraints (min/max).
 */
function validateNumericConstraints(
    state: EvalState,
    id: string,
    value: number,
    def: Definition
): void {
    if (def.min !== undefined && value < def.min) {
        addError(state, id, '', 'constraint_violation', `Field '${id}' value ${value.toFixed(2)} is below minimum ${def.min.toFixed(2)}`);
    }
    if (def.max !== undefined && value > def.max) {
        addError(state, id, '', 'constraint_violation', `Field '${id}' value ${value.toFixed(2)} exceeds maximum ${def.max.toFixed(2)}`);
    }
}

/**
 * Validate string constraints (length, pattern).
 */
function validateStringConstraints(
    state: EvalState,
    id: string,
    value: string,
    def: Definition
): void {
    if (def.min_length !== undefined && value.length < def.min_length) {
        addError(state, id, '', 'constraint_violation', `Field '${id}' is too short (minimum ${def.min_length} characters)`);
    }
    if (def.max_length !== undefined && value.length > def.max_length) {
        addError(state, id, '', 'constraint_violation', `Field '${id}' is too long (maximum ${def.max_length} characters)`);
    }
    if (def.pattern) {
        try {
            const regex = new RegExp(def.pattern);
            if (!regex.test(value)) {
                addError(state, id, '', 'constraint_violation', `Field '${id}' does not match required pattern`);
            }
        } catch {
            // Invalid regex pattern, skip validation
        }
    }
}

/**
 * Validate a single definition's type and constraints.
 * Array values are allowed — the declared type describes the element type,
 * used by collection operators (some/all/none). Scalar validation is skipped for arrays.
 */
function validateType(state: EvalState, id: string, def: Definition): void {
    const value = def.value;

    // Skip scalar validation for array values (used with some/all/none operators)
    if (Array.isArray(value)) {
        return;
    }

    switch (def.type) {
        case 'string': {
            if (typeof value !== 'string') {
                addError(state, id, '', 'type_mismatch', `Field '${id}' must be a string`);
                return;
            }
            validateStringConstraints(state, id, value, def);
            break;
        }

        case 'number':
        case 'currency': {
            const [numVal, ok] = toFloat(value);
            if (!ok) {
                addError(state, id, '', 'type_mismatch', `Field '${id}' must be a number`);
                return;
            }
            validateNumericConstraints(state, id, numVal, def);
            break;
        }

        case 'boolean': {
            if (typeof value !== 'boolean') {
                addError(state, id, '', 'type_mismatch', `Field '${id}' must be a boolean`);
            }
            break;
        }

        case 'select': {
            if (typeof value !== 'string') {
                addError(state, id, '', 'type_mismatch', `Field '${id}' must be a string`);
                return;
            }
            if (!isValidOption(value, def.options)) {
                addError(state, id, '', 'constraint_violation', `Field '${id}' value '${value}' is not a valid option`);
            }
            break;
        }

        case 'attestation': {
            if (typeof value !== 'boolean') {
                addError(state, id, '', 'type_mismatch', `Attestation '${id}' must be a boolean`);
            }
            break;
        }

        case 'date': {
            const [, ok] = parseDate(value);
            if (!ok) {
                addError(state, id, '', 'type_mismatch', `Field '${id}' must be a valid date`);
            }
            break;
        }
    }
}

/**
 * Validate all definitions for type correctness and required fields.
 */
export function validateDefinitions(state: EvalState): void {
    for (const [id, def] of Object.entries(state.schema.definitions)) {
        if (!def) {
            continue;
        }

        // Check required fields
        if (def.required) {
            if (def.value === undefined || def.value === null) {
                addError(state, id, '', 'missing_required', `Required field '${id}' is missing`);
            } else if ((def.type === 'string' || def.type === 'select') && def.value === '') {
                // Empty string is also considered "missing" for required string/select fields
                addError(state, id, '', 'missing_required', `Required field '${id}' is missing`);
            }
        }

        // Validate type if value is present
        if (def.value !== undefined && def.value !== null) {
            validateType(state, id, def);
        }
    }
}

/**
 * Check attestations for required signatures.
 */
export function checkAttestations(
    state: EvalState,
    applyAction: (action: Action, ruleId: string, lawRef: string) => void
): void {
    // Check legacy attestations in definitions (simple type: attestation)
    for (const [id, def] of Object.entries(state.schema.definitions)) {
        if (!def || def.type !== 'attestation') {
            continue;
        }
        if (def.required && def.value !== true) {
            addError(state, id, '', 'attestation_incomplete', `Required attestation '${id}' not confirmed`);
        }
    }

    // Check rich attestations
    if (!state.schema.attestations) {
        return;
    }

    for (const [id, att] of Object.entries(state.schema.attestations)) {
        if (!att) {
            continue;
        }

        // Process on_sign if signed is true
        if (att.signed && att.on_sign) {
            applyAction(att.on_sign, `attestation_${id}`, att.law_ref || '');
        }

        // Validate required attestations
        if (att.required) {
            if (!att.signed) {
                addError(state, id, '', 'attestation_incomplete', `Required attestation '${id}' not signed`, att.law_ref);
            } else if (!att.evidence || !att.evidence.provider_audit_id) {
                addError(state, id, '', 'attestation_incomplete', `Attestation '${id}' signed but missing evidence`, att.law_ref);
            }
        }
    }
}

/**
 * Determine document status based on ErrorKind.
 * Non-blocking kinds (runtime_warning, cycle_detected, notice) do not affect status.
 */
export function determineStatus(state: EvalState): DocStatus {
    for (const err of state.errors) {
        if (err.kind === 'type_mismatch') return 'INVALID';
    }
    for (const err of state.errors) {
        if (err.kind === 'missing_required' || err.kind === 'attestation_incomplete') {
            return 'INCOMPLETE';
        }
    }
    for (const err of state.errors) {
        if (err.kind === 'constraint_violation') return 'INVALID';
    }
    return 'READY';
}
