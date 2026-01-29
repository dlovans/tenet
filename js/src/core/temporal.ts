/**
 * Temporal branch selection and rule pruning.
 * Routes logic based on effective dates for bitemporal support.
 */

import type { EvalState, TemporalBranch } from './types.js';
import { addError } from './validate.js';

/**
 * Validate temporal_map for configuration errors.
 * Checks for same start/end dates and overlapping ranges.
 */
function validateTemporalMap(state: EvalState): void {
    const { schema } = state;

    if (!schema.temporal_map || schema.temporal_map.length === 0) {
        return;
    }

    for (let i = 0; i < schema.temporal_map.length; i++) {
        const branch = schema.temporal_map[i];
        if (!branch) {
            continue;
        }

        const start = branch.valid_range[0];
        const end = branch.valid_range[1];

        // Check for same start/end date (invalid zero-length range)
        if (start && end && start === end) {
            addError(
                state,
                '',
                '',
                `Temporal branch ${i} has same start and end date '${start}' (invalid range)`
            );
        }

        // Check for overlapping with previous branch
        if (i > 0) {
            const prev = schema.temporal_map[i - 1];
            if (prev) {
                const prevEnd = prev.valid_range[1]
                    ? new Date(prev.valid_range[1]).getTime()
                    : Infinity;
                const currStart = start
                    ? new Date(start).getTime()
                    : -Infinity;

                if (currStart <= prevEnd) {
                    addError(
                        state,
                        '',
                        '',
                        `Temporal branch ${i} overlaps with branch ${i - 1} (ranges must not overlap)`
                    );
                }
            }
        }
    }
}

/**
 * Find the active temporal branch for a given effective date.
 * Returns undefined if no branch matches (uses default/unversioned logic).
 */
export function selectBranch(state: EvalState): TemporalBranch | undefined {
    const { schema, effectiveDate } = state;

    if (!schema.temporal_map || schema.temporal_map.length === 0) {
        return undefined;
    }

    for (const branch of schema.temporal_map) {
        if (!branch || !branch.valid_range[0]) {
            continue;
        }

        const start = new Date(branch.valid_range[0]);
        if (isNaN(start.getTime())) {
            continue;
        }

        // Check if effectiveDate is at or after start
        if (effectiveDate < start) {
            continue;
        }

        // Check end date (null = open-ended)
        if (branch.valid_range[1]) {
            const end = new Date(branch.valid_range[1]);
            if (!isNaN(end.getTime()) && effectiveDate > end) {
                continue;
            }
        }

        return branch;
    }

    return undefined;
}

/**
 * Mark rules as disabled if they don't belong to the active branch.
 * Rules without a logic_version are always active (unversioned rules).
 */
export function pruneRules(state: EvalState, activeBranch: TemporalBranch | undefined): void {
    if (!activeBranch || !state.schema.logic_tree) {
        return;
    }

    const activeVersion = activeBranch.logic_version;

    for (const rule of state.schema.logic_tree) {
        if (!rule) {
            continue;
        }

        // Rules without a version are always active
        if (!rule.logic_version) {
            continue;
        }

        // Disable rules that don't match the active version
        if (rule.logic_version !== activeVersion) {
            rule.disabled = true;
        }
    }
}

/**
 * Select temporal branch and prune inactive rules.
 * Call this at the start of Run().
 */
export function applyTemporalRouting(state: EvalState): void {
    // Validate temporal_map configuration
    validateTemporalMap(state);

    const branch = selectBranch(state);
    if (branch) {
        pruneRules(state, branch);
    }
}
