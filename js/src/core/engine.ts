/**
 * Main Tenet VM engine.
 * Provides run() and verify() functions for schema evaluation.
 */

import type {
    TenetSchema,
    TenetResult,
    TenetVerifyResult,
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
        addError(state, key, ruleId, `Potential cycle: field '${key}' set by rule '${prevRule}' and again by rule '${ruleId}'`);
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
        addError(state, '', ruleId, action.error_msg, lawRef);
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

        // Store the computed value as a definition (readonly)
        state.schema.definitions[name] = {
            type: inferType(value),
            value,
            readonly: true,
            visible: true,
        };
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
 * Count visible fields in a schema.
 */
function countVisibleFields(schema: TenetSchema): number {
    let count = 0;
    for (const def of Object.values(schema.definitions)) {
        if (def?.visible) {
            count++;
        }
    }
    return count;
}

/**
 * Validate that the final state matches expected values.
 */
function validateFinalState(
    newSchema: TenetSchema,
    resultSchema: TenetSchema
): [boolean, string | undefined] {
    // Check for unknown/injected fields in newSchema that don't exist in result
    for (const id of Object.keys(newSchema.definitions)) {
        if (!(id in resultSchema.definitions)) {
            return [false, `Unknown field '${id}' not in schema`];
        }
    }

    // Compare computed (readonly) values
    for (const [id, resultDef] of Object.entries(resultSchema.definitions)) {
        if (!resultDef?.readonly) {
            continue;
        }

        const newDef = newSchema.definitions[id];
        if (!newDef) {
            return [false, `Computed field '${id}' missing in submitted document`];
        }

        if (!compareEqual(newDef.value, resultDef.value)) {
            return [false, `Computed field '${id}' mismatch: claimed ${JSON.stringify(newDef.value)}, expected ${JSON.stringify(resultDef.value)}`];
        }
    }

    // Verify attestations are fulfilled
    if (resultSchema.attestations) {
        for (const [id, resultAtt] of Object.entries(resultSchema.attestations)) {
            if (!resultAtt) {
                continue;
            }

            const newAtt = newSchema.attestations?.[id];
            if (!newAtt) {
                continue;
            }

            if (resultAtt.required) {
                if (!newAtt.signed) {
                    return [false, `Required attestation '${id}' not signed`];
                }
                if (!newAtt.evidence?.provider_audit_id) {
                    return [false, `Attestation '${id}' missing evidence`];
                }
                if (!newAtt.evidence?.timestamp) {
                    return [false, `Attestation '${id}' missing timestamp`];
                }
            }
        }
    }

    // Verify status matches
    if (newSchema.status !== resultSchema.status) {
        return [false, `Status mismatch: claimed ${newSchema.status}, expected ${resultSchema.status}`];
    }

    return [true, undefined];
}

/**
 * Verify that a completed document was correctly derived from a base schema.
 * Simulates the user's journey by iteratively copying visible field values and re-running.
 *
 * @param newSchema - The completed/submitted schema
 * @param oldSchema - The original base schema
 * @param maxIterations - Maximum replay iterations (default: 100)
 * @returns Whether the transformation was valid
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
        let previousVisibleCount = -1;

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
                return { valid: false, error: `Run failed (iteration ${iteration}): ${runResult.error}` };
            }

            const resultSchema = runResult.result!;

            // Count visible fields after run
            const currentVisibleCount = countVisibleFields(resultSchema);

            // Check for convergence
            if (currentVisibleCount === previousVisibleCount) {
                // Converged - now validate the final state
                const [valid, error] = validateFinalState(parsedNewSchema, resultSchema);
                return { valid, error };
            }

            previousVisibleCount = currentVisibleCount;
            currentSchema = resultSchema;
        }

        return { valid: false, error: `Verification did not converge after ${maxIterations} iterations` };
    } catch (error) {
        return { valid: false, error: String(error) };
    }
}
