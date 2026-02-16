/**
 * Main Tenet VM engine.
 * Provides run() and verify() functions for schema evaluation.
 */

import type {
    TenetSchema,
    TenetResult,
    TenetVerifyResult,
    VerifyIssue,
    EvalState,
    Rule,
    Action,
    Definition,
} from './types.js';
import { resolve } from './resolver.js';
import { isTruthy, toFloat, compareEqual } from './operators.js';
import { applyTemporalRouting } from './temporal.js';
import { validateDefinitions, checkAttestations, determineStatus, addError } from './validate.js';

/**
 * Infer type string from a value.
 */
function inferType(value: unknown): Definition['type'] {
    if (typeof value === 'string') return 'string';
    if (typeof value === 'number') return 'number';
    if (typeof value === 'boolean') return 'boolean';
    return 'string'; // default
}

/**
 * Deep clone an object (for immutability).
 */
function deepClone<T>(obj: T): T {
    return JSON.parse(JSON.stringify(obj));
}

/**
 * Apply UI modifications to a definition.
 */
function applyUIModify(state: EvalState, key: string, mods: unknown): void {
    const def = state.schema.definitions[key];
    if (!def) {
        return;
    }

    if (typeof mods !== 'object' || mods === null) {
        return;
    }

    const modMap = mods as Record<string, unknown>;

    // Apply visibility and metadata modifications
    if (typeof modMap['visible'] === 'boolean') {
        def.visible = modMap['visible'];
    }
    if (typeof modMap['ui_class'] === 'string') {
        def.ui_class = modMap['ui_class'];
    }
    if (typeof modMap['ui_message'] === 'string') {
        def.ui_message = modMap['ui_message'];
    }
    if (typeof modMap['required'] === 'boolean') {
        def.required = modMap['required'];
    }

    // Apply numeric constraints
    const [minVal, minOk] = toFloat(modMap['min']);
    if (minOk) {
        def.min = minVal;
    }
    const [maxVal, maxOk] = toFloat(modMap['max']);
    if (maxOk) {
        def.max = maxVal;
    }
    const [stepVal, stepOk] = toFloat(modMap['step']);
    if (stepOk) {
        def.step = stepVal;
    }

    // Apply string constraints
    if (typeof modMap['min_length'] === 'number') {
        def.min_length = modMap['min_length'];
    }
    if (typeof modMap['max_length'] === 'number') {
        def.max_length = modMap['max_length'];
    }
    if (typeof modMap['pattern'] === 'string') {
        def.pattern = modMap['pattern'];
    }
}

/**
 * Set a definition value, with cycle detection.
 */
function setDefinitionValue(state: EvalState, key: string, value: unknown, ruleId: string): void {
    // Cycle detection
    const prevRule = state.fieldsSet.get(key);
    if (prevRule && prevRule !== ruleId) {
        addError(state, key, ruleId, 'cycle_detected', `Potential cycle: field '${key}' set by rule '${prevRule}' and again by rule '${ruleId}'`);
    }
    state.fieldsSet.set(key, ruleId);

    const def = state.schema.definitions[key];
    if (!def) {
        // Create new definition if it doesn't exist
        state.schema.definitions[key] = {
            type: inferType(value),
            value,
            visible: true,
        };
        return;
    }

    def.value = value;
}

/**
 * Apply a rule's action: setting values, modifying UI, or emitting errors.
 */
function applyAction(state: EvalState, action: Action | undefined, ruleId: string, lawRef: string): void {
    if (!action) {
        return;
    }

    // Apply value mutations
    if (action.set) {
        for (const [key, value] of Object.entries(action.set)) {
            // Resolve the value in case it's an expression
            const resolvedValue = resolve(value, state);
            setDefinitionValue(state, key, resolvedValue, ruleId);
        }
    }

    // Apply UI modifications
    if (action.ui_modify) {
        for (const [key, mods] of Object.entries(action.ui_modify)) {
            applyUIModify(state, key, mods);
        }
    }

    // Emit error if specified
    if (action.error_msg) {
        addError(state, '', ruleId, 'runtime_warning', action.error_msg, lawRef);
    }
}

/**
 * Evaluate the logic tree (all active rules in order).
 */
function evaluateLogicTree(state: EvalState): void {
    if (!state.schema.logic_tree) {
        return;
    }

    for (const rule of state.schema.logic_tree) {
        if (!rule || rule.disabled) {
            continue;
        }

        // Evaluate the condition
        const condition = resolve(rule.when, state);
        if (isTruthy(condition)) {
            applyAction(state, rule.then, rule.id, rule.law_ref || '');
        }
    }
}

/**
 * Compute derived state values.
 */
function computeDerived(state: EvalState): void {
    if (!state.schema.state_model?.derived) {
        return;
    }

    for (const [name, derivedDef] of Object.entries(state.schema.state_model.derived)) {
        if (!derivedDef?.eval) {
            continue;
        }

        // Evaluate the expression
        const value = resolve(derivedDef.eval, state);

        // Preserve existing definition metadata if present
        const existing = state.schema.definitions[name];
        if (existing) {
            existing.value = value;
            existing.readonly = true;
            if (existing.visible === undefined) {
                existing.visible = true;
            }
        } else {
            state.schema.definitions[name] = {
                type: inferType(value),
                value,
                readonly: true,
                visible: true,
            };
        }
    }
}

/**
 * Run the Tenet VM on a schema.
 *
 * @param schema - The schema object
 * @param effectiveDate - Effective date for temporal routing (defaults to now)
 * @returns The transformed schema with computed state, errors, and status
 */
export function run(
    schema: TenetSchema | string,
    effectiveDate: Date | string = new Date()
): TenetResult {
    try {
        // Parse if string
        const parsedSchema: TenetSchema = typeof schema === 'string'
            ? JSON.parse(schema)
            : deepClone(schema);

        // Parse date
        const date = effectiveDate instanceof Date
            ? effectiveDate
            : new Date(effectiveDate);

        // Initialize default visibility for definitions
        for (const def of Object.values(parsedSchema.definitions)) {
            if (def && def.visible === undefined) {
                def.visible = true;
            }
        }

        // Create evaluation state
        const state: EvalState = {
            schema: parsedSchema,
            effectiveDate: date,
            fieldsSet: new Map(),
            errors: [],
            derivedInProgress: new Set(),
        };

        // 1. Select temporal branch and prune inactive rules
        applyTemporalRouting(state);

        // 2. Compute derived state (so logic tree can use derived values)
        computeDerived(state);

        // 3. Evaluate logic tree
        evaluateLogicTree(state);

        // 4. Re-compute derived state (in case logic modified inputs)
        computeDerived(state);

        // 5. Validate definitions
        validateDefinitions(state);

        // 6. Check attestations
        checkAttestations(state, (action, ruleId, lawRef) => {
            applyAction(state, action, ruleId, lawRef);
        });

        // 7. Determine status and attach errors
        state.schema.errors = state.errors.length > 0 ? state.errors : undefined;
        state.schema.status = determineStatus(state);

        return { result: state.schema };
    } catch (error) {
        return { error: String(error) };
    }
}

/**
 * Get visible, editable fields from a schema.
 */
function getVisibleEditableFields(schema: TenetSchema): Set<string> {
    const result = new Set<string>();
    for (const [id, def] of Object.entries(schema.definitions)) {
        if (def && def.visible && !def.readonly) {
            result.add(id);
        }
    }
    return result;
}

/**
 * Get a sorted, comma-joined string of visible field IDs for convergence detection.
 */
function getVisibleFieldIds(schema: TenetSchema): string {
    return Object.entries(schema.definitions)
        .filter(([, def]) => def?.visible)
        .map(([id]) => id)
        .sort()
        .join(',');
}

/**
 * Validate that the final state matches expected values.
 * Collects ALL issues instead of bailing on the first — the UI needs the complete picture.
 */
function validateFinalState(
    newSchema: TenetSchema,
    resultSchema: TenetSchema
): TenetVerifyResult {
    const issues: VerifyIssue[] = [];

    // Check for unknown/injected fields in newSchema that don't exist in result
    for (const id of Object.keys(newSchema.definitions)) {
        if (!(id in resultSchema.definitions)) {
            issues.push({
                code: 'unknown_field',
                field_id: id,
                message: `Field '${id}' does not exist in the schema`,
            });
        }
    }

    // Compare computed (readonly) values
    for (const [id, resultDef] of Object.entries(resultSchema.definitions)) {
        if (!resultDef?.readonly) {
            continue;
        }

        const newDef = newSchema.definitions[id];
        if (!newDef) {
            issues.push({
                code: 'computed_mismatch',
                field_id: id,
                message: `Computed field '${id}' is missing from the submitted document`,
                expected: resultDef.value,
            });
            continue;
        }

        if (!compareEqual(newDef.value, resultDef.value)) {
            issues.push({
                code: 'computed_mismatch',
                field_id: id,
                message: `Computed field '${id}' was modified`,
                expected: resultDef.value,
                claimed: newDef.value,
            });
        }
    }

    // Verify attestations are fulfilled
    if (resultSchema.attestations) {
        for (const [id, resultAtt] of Object.entries(resultSchema.attestations)) {
            if (!resultAtt?.required) {
                continue;
            }

            const newAtt = newSchema.attestations?.[id];
            if (!newAtt) {
                continue;
            }

            if (!newAtt.signed) {
                issues.push({
                    code: 'attestation_unsigned',
                    field_id: id,
                    message: `Required attestation '${id}' has not been signed`,
                });
                continue; // No point checking evidence if unsigned
            }

            if (!newAtt.evidence?.provider_audit_id) {
                issues.push({
                    code: 'attestation_no_evidence',
                    field_id: id,
                    message: `Attestation '${id}' is signed but missing proof of signing`,
                });
            }

            if (!newAtt.evidence?.timestamp) {
                issues.push({
                    code: 'attestation_no_timestamp',
                    field_id: id,
                    message: `Attestation '${id}' is signed but missing a timestamp`,
                });
            }
        }
    }

    // Verify status matches
    if (newSchema.status !== resultSchema.status) {
        issues.push({
            code: 'status_mismatch',
            message: 'The document status does not match what was computed',
            expected: resultSchema.status,
            claimed: newSchema.status,
        });
    }

    return {
        valid: issues.length === 0,
        status: resultSchema.status,
        issues: issues.length > 0 ? issues : undefined,
        schema: resultSchema,
    };
}

/**
 * Verify that a completed document was correctly derived from a base schema.
 * Simulates the user's journey by iteratively copying visible field values and re-running.
 *
 * Returns a structured result with all issues found (not just the first).
 * Crash-safe: catches any unexpected error and returns it as an internal_error issue.
 *
 * @param newSchema - The completed/submitted schema
 * @param oldSchema - The original base schema
 * @param maxIterations - Maximum replay iterations (default: 100)
 * @returns Structured verification result
 */
export function verify(
    newSchema: TenetSchema | string,
    oldSchema: TenetSchema | string,
    maxIterations: number = 100
): TenetVerifyResult {
    try {
        // Parse schemas
        const parsedNewSchema: TenetSchema = typeof newSchema === 'string'
            ? JSON.parse(newSchema)
            : deepClone(newSchema);

        const parsedOldSchema: TenetSchema = typeof oldSchema === 'string'
            ? JSON.parse(oldSchema)
            : deepClone(oldSchema);

        // Extract effective date from newSchema
        let effectiveDate = new Date();
        if (parsedNewSchema.valid_from) {
            const parsed = new Date(parsedNewSchema.valid_from);
            if (!isNaN(parsed.getTime())) {
                effectiveDate = parsed;
            }
        }

        // Start with base schema
        let currentSchema = parsedOldSchema;
        let previousVisibleIds = '';

        for (let iteration = 0; iteration < maxIterations; iteration++) {
            // Count visible editable fields before copying
            const visibleEditable = getVisibleEditableFields(currentSchema);

            // Copy values from newSchema for visible, editable fields
            for (const fieldId of visibleEditable) {
                const newDef = parsedNewSchema.definitions[fieldId];
                const currentDef = currentSchema.definitions[fieldId];
                if (newDef && currentDef) {
                    currentDef.value = newDef.value;
                }
            }

            // Copy attestation states
            if (currentSchema.attestations) {
                for (const [attId, currentAtt] of Object.entries(currentSchema.attestations)) {
                    if (!currentAtt) continue;
                    const newAtt = parsedNewSchema.attestations?.[attId];
                    if (newAtt) {
                        currentAtt.signed = newAtt.signed;
                        currentAtt.evidence = newAtt.evidence;
                    }
                }
            }

            // Run the schema
            const runResult = run(currentSchema, effectiveDate);
            if (runResult.error) {
                return {
                    valid: false,
                    issues: [{
                        code: 'internal_error',
                        message: `VM run failed at iteration ${iteration}`,
                    }],
                    error: `Run failed (iteration ${iteration}): ${runResult.error}`,
                };
            }

            const resultSchema = runResult.result!;

            // Get visible field IDs after run
            const currentVisibleIds = getVisibleFieldIds(resultSchema);

            // Check for convergence using set comparison
            if (currentVisibleIds === previousVisibleIds) {
                // Converged - now validate the final state and return full result
                return validateFinalState(parsedNewSchema, resultSchema);
            }

            previousVisibleIds = currentVisibleIds;
            currentSchema = resultSchema;
        }

        return {
            valid: false,
            issues: [{
                code: 'convergence_failed',
                message: `Document did not converge after ${maxIterations} iterations`,
            }],
        };
    } catch (error) {
        return {
            valid: false,
            issues: [{
                code: 'internal_error',
                message: `Unexpected error during verification`,
            }],
            error: String(error),
        };
    }
}
