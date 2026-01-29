/**
 * Definition validation and status determination.
 * Validates types, constraints, and required fields.
 */

import type { EvalState, Definition, ValidationError, DocStatus } from './types.js';
import { toFloat, parseDate } from './operators.js';

/**
 * Add an error to the state's error list.
 */
export function addError(
    state: EvalState,
    fieldId: string,
    ruleId: string,
    message: string,
    lawRef?: string
): void {
    state.errors.push({
        field_id: fieldId || undefined,
        rule_id: ruleId || undefined,
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
        addError(state, id, '', `Field '${id}' value ${value.toFixed(2)} is below minimum ${def.min.toFixed(2)}`);
    }
    if (def.max !== undefined && value > def.max) {
        addError(state, id, '', `Field '${id}' value ${value.toFixed(2)} exceeds maximum ${def.max.toFixed(2)}`);
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
        addError(state, id, '', `Field '${id}' is too short (minimum ${def.min_length} characters)`);
    }
    if (def.max_length !== undefined && value.length > def.max_length) {
        addError(state, id, '', `Field '${id}' is too long (maximum ${def.max_length} characters)`);
    }
    if (def.pattern) {
        try {
            const regex = new RegExp(def.pattern);
            if (!regex.test(value)) {
                addError(state, id, '', `Field '${id}' does not match required pattern`);
            }
        } catch {
            // Invalid regex pattern, skip validation
        }
    }
}

/**
 * Validate a single definition's type and constraints.
 */
function validateType(state: EvalState, id: string, def: Definition): void {
    const value = def.value;

    switch (def.type) {
        case 'string': {
            if (typeof value !== 'string') {
                addError(state, id, '', `Field '${id}' must be a string`);
                return;
            }
            validateStringConstraints(state, id, value, def);
            break;
        }

        case 'number':
        case 'currency': {
            const [numVal, ok] = toFloat(value);
            if (!ok) {
                addError(state, id, '', `Field '${id}' must be a number`);
                return;
            }
            validateNumericConstraints(state, id, numVal, def);
            break;
        }

        case 'boolean': {
            if (typeof value !== 'boolean') {
                addError(state, id, '', `Field '${id}' must be a boolean`);
            }
            break;
        }

        case 'select': {
            if (typeof value !== 'string') {
                addError(state, id, '', `Field '${id}' must be a string`);
                return;
            }
            if (!isValidOption(value, def.options)) {
                addError(state, id, '', `Field '${id}' value '${value}' is not a valid option`);
            }
            break;
        }

        case 'attestation': {
            if (typeof value !== 'boolean') {
                addError(state, id, '', `Attestation '${id}' must be a boolean`);
            }
            break;
        }

        case 'date': {
            const [, ok] = parseDate(value);
            if (!ok) {
                addError(state, id, '', `Field '${id}' must be a valid date`);
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
        if (def.required && (def.value === undefined || def.value === null)) {
            addError(state, id, '', `Required field '${id}' is missing`);
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
export function checkAttestations(state: EvalState, applyAction: (action: any, ruleId: string, lawRef: string) => void): void {
    // Check legacy attestations in definitions (simple type: attestation)
    for (const [id, def] of Object.entries(state.schema.definitions)) {
        if (!def || def.type !== 'attestation') {
            continue;
        }
        if (def.required && def.value !== true) {
            addError(state, id, '', `Required attestation '${id}' not confirmed`);
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
                addError(state, id, '', `Required attestation '${id}' not signed`, att.law_ref);
            } else if (!att.evidence || !att.evidence.provider_audit_id) {
                addError(state, id, '', `Attestation '${id}' signed but missing evidence`, att.law_ref);
            }
        }
    }
}

/**
 * Determine document status based on validation errors.
 */
export function determineStatus(state: EvalState): DocStatus {
    let hasTypeErrors = false;
    let hasMissingRequired = false;
    let hasMissingAttestations = false;

    for (const err of state.errors) {
        const msg = err.message;
        if (msg.includes('must be a')) {
            hasTypeErrors = true;
        } else if (msg.includes('missing') || msg.includes('Required field')) {
            hasMissingRequired = true;
        } else if (msg.includes('attestation')) {
            hasMissingAttestations = true;
        }
    }

    if (hasTypeErrors) {
        return 'INVALID';
    }
    if (hasMissingRequired || hasMissingAttestations) {
        return 'INCOMPLETE';
    }
    return 'READY';
}
