/**
 * Static schema linter for Tenet schemas.
 * Analyzes schemas without executing them, catching structural errors,
 * type mismatches, undefined references, and circular dependencies.
 */

import type {
    TenetSchema,
    Definition,
    Rule,
    Action,
    TemporalBranch,
} from './types.js';

// ============================================================
// Public types
// ============================================================

export type LintSeverity = 'error' | 'warning';

export interface LintIssue {
    severity: LintSeverity;
    code: string;
    message: string;
    path?: string;
    field_id?: string;
    rule_id?: string;
}

export interface LintResult {
    valid: boolean;
    issues: LintIssue[];
}

// ============================================================
// Internal types
// ============================================================

type InferredType = 'string' | 'number' | 'boolean' | 'date' | 'unknown';

interface LintContext {
    schema: TenetSchema;
    issues: LintIssue[];
    knownFields: Map<string, InferredType>;
    supportedOps: Set<string>;
}

// ============================================================
// Constants
// ============================================================

const VALID_DEF_TYPES: ReadonlySet<string> = new Set([
    'string', 'number', 'boolean', 'select', 'date', 'attestation', 'currency',
]);

const SUPPORTED_OPS: ReadonlySet<string> = new Set([
    'var', '==', '!=', '>', '<', '>=', '<=',
    'and', 'or', 'not', '!', 'if',
    '+', '-', '*', '/',
    'before', 'after',
    'in', 'some', 'all', 'none',
]);

const ARITHMETIC_OPS: ReadonlySet<string> = new Set(['+', '-', '*', '/']);
const COMPARISON_OPS: ReadonlySet<string> = new Set(['>', '<', '>=', '<=']);

// ============================================================
// Helpers
// ============================================================

function addIssue(
    ctx: LintContext,
    severity: LintSeverity,
    code: string,
    message: string,
    opts?: { path?: string; field_id?: string; rule_id?: string }
): void {
    ctx.issues.push({
        severity,
        code,
        message,
        path: opts?.path,
        field_id: opts?.field_id,
        rule_id: opts?.rule_id,
    });
}

function defTypeToInferred(type: Definition['type']): InferredType {
    switch (type) {
        case 'number':
        case 'currency':
            return 'number';
        case 'string':
        case 'select':
            return 'string';
        case 'boolean':
        case 'attestation':
            return 'boolean';
        case 'date':
            return 'date';
        default:
            return 'unknown';
    }
}

// ============================================================
// Expression type inference
// ============================================================

function inferExprType(expr: unknown, ctx: LintContext, path: string): InferredType {
    if (expr === null || expr === undefined) return 'unknown';
    if (typeof expr === 'number') return 'number';
    if (typeof expr === 'string') return 'string';
    if (typeof expr === 'boolean') return 'boolean';
    if (Array.isArray(expr)) return 'unknown';

    if (typeof expr !== 'object') return 'unknown';

    const obj = expr as Record<string, unknown>;
    const keys = Object.keys(obj);
    if (keys.length !== 1) return 'unknown';

    const op = keys[0];
    const args = obj[op];

    // Check for unknown operators
    if (!ctx.supportedOps.has(op)) {
        addIssue(ctx, 'warning', 'unknown_operator', `Unknown operator '${op}'`, { path });
        return 'unknown';
    }

    if (op === 'var') {
        const varName = typeof args === 'string' ? args : '';
        const rootName = varName.split('.')[0];
        const fieldType = ctx.knownFields.get(rootName);
        if (fieldType !== undefined) {
            return fieldType;
        }
        if (rootName && !ctx.knownFields.has(rootName)) {
            addIssue(ctx, 'warning', 'undefined_variable', `Variable '${rootName}' is not defined in definitions or derived fields`, {
                path,
                field_id: rootName,
            });
        }
        return 'unknown';
    }

    if (ARITHMETIC_OPS.has(op)) {
        const operands = Array.isArray(args) ? args : [args];
        for (let i = 0; i < operands.length; i++) {
            const operandType = inferExprType(operands[i], ctx, `${path}.${op}[${i}]`);
            if (operandType !== 'unknown' && operandType !== 'number') {
                addIssue(ctx, 'error', 'arithmetic_type_mismatch',
                    `Arithmetic operator '${op}' used with ${operandType} operand (expected number)`, {
                        path: `${path}.${op}[${i}]`,
                    });
            }
        }
        return 'number';
    }

    if (COMPARISON_OPS.has(op)) {
        const operands = Array.isArray(args) ? args : [args];
        if (operands.length >= 2) {
            const leftType = inferExprType(operands[0], ctx, `${path}.${op}[0]`);
            const rightType = inferExprType(operands[1], ctx, `${path}.${op}[1]`);
            if (leftType !== 'unknown' && rightType !== 'unknown' && leftType !== rightType) {
                addIssue(ctx, 'warning', 'comparison_type_mismatch',
                    `Comparison operator '${op}' between ${leftType} and ${rightType}`, {
                        path,
                    });
            }
        }
        return 'boolean';
    }

    if (op === '==' || op === '!=') {
        // Infer children for side effects (undefined_variable etc) but don't flag type mismatch
        const operands = Array.isArray(args) ? args : [args];
        for (let i = 0; i < operands.length; i++) {
            inferExprType(operands[i], ctx, `${path}.${op}[${i}]`);
        }
        return 'boolean';
    }

    if (op === 'and' || op === 'or') {
        const operands = Array.isArray(args) ? args : [args];
        for (let i = 0; i < operands.length; i++) {
            inferExprType(operands[i], ctx, `${path}.${op}[${i}]`);
        }
        return 'boolean';
    }

    if (op === 'not' || op === '!') {
        inferExprType(args, ctx, `${path}.${op}`);
        return 'boolean';
    }

    if (op === 'if') {
        const branches = Array.isArray(args) ? args : [];
        // Walk all branches for side effects
        for (let i = 0; i < branches.length; i++) {
            inferExprType(branches[i], ctx, `${path}.if[${i}]`);
        }
        // Return type of first then-branch if exists
        if (branches.length >= 2) {
            return inferExprType(branches[1], ctx, `${path}.if[1]`);
        }
        return 'unknown';
    }

    if (op === 'before' || op === 'after') {
        const operands = Array.isArray(args) ? args : [args];
        for (let i = 0; i < operands.length; i++) {
            inferExprType(operands[i], ctx, `${path}.${op}[${i}]`);
        }
        return 'boolean';
    }

    if (op === 'in' || op === 'some' || op === 'all' || op === 'none') {
        const operands = Array.isArray(args) ? args : [args];
        for (let i = 0; i < operands.length; i++) {
            inferExprType(operands[i], ctx, `${path}.${op}[${i}]`);
        }
        return 'boolean';
    }

    return 'unknown';
}

// ============================================================
// Var reference extraction (for cycle detection)
// ============================================================

function collectVarRefs(node: unknown, refs: Set<string>): void {
    if (node === null || node === undefined) return;
    if (typeof node !== 'object') return;

    if (Array.isArray(node)) {
        for (const item of node) {
            collectVarRefs(item, refs);
        }
        return;
    }

    const obj = node as Record<string, unknown>;
    const keys = Object.keys(obj);
    if (keys.length === 1 && keys[0] === 'var') {
        const varName = typeof obj['var'] === 'string' ? obj['var'] : '';
        const rootName = varName.split('.')[0];
        if (rootName) refs.add(rootName);
        return;
    }

    for (const key of keys) {
        collectVarRefs(obj[key], refs);
    }
}

// ============================================================
// Check passes
// ============================================================

function checkDefinitions(ctx: LintContext): void {
    const { schema } = ctx;

    // E1: no definitions
    if (!schema.definitions || Object.keys(schema.definitions).length === 0) {
        addIssue(ctx, 'error', 'no_definitions', 'Schema has no definitions');
        return;
    }

    for (const [id, def] of Object.entries(schema.definitions)) {
        if (!def) continue;

        // E2: invalid field type
        if (!VALID_DEF_TYPES.has(def.type)) {
            addIssue(ctx, 'error', 'invalid_field_type',
                `Field '${id}' has invalid type '${def.type}'`, { field_id: id });
        }

        // E6/E7: select validations
        if (def.type === 'select') {
            if (def.options === undefined) {
                addIssue(ctx, 'error', 'select_missing_options',
                    `Select field '${id}' is missing options`, { field_id: id });
            } else if (Array.isArray(def.options) && def.options.length === 0) {
                addIssue(ctx, 'error', 'select_options_empty',
                    `Select field '${id}' has empty options array`, { field_id: id });
            }
        }

        // E8: min > max
        if (def.min !== undefined && def.max !== undefined && def.min > def.max) {
            addIssue(ctx, 'error', 'min_exceeds_max',
                `Field '${id}' has min (${def.min}) greater than max (${def.max})`, { field_id: id });
        }

        // E9: min_length > max_length
        if (def.min_length !== undefined && def.max_length !== undefined && def.min_length > def.max_length) {
            addIssue(ctx, 'error', 'min_length_exceeds_max_length',
                `Field '${id}' has min_length (${def.min_length}) greater than max_length (${def.max_length})`, { field_id: id });
        }

        // E10: invalid regex
        if (def.pattern !== undefined) {
            try {
                new RegExp(def.pattern);
            } catch {
                addIssue(ctx, 'error', 'invalid_regex',
                    `Field '${id}' has invalid regex pattern '${def.pattern}'`, { field_id: id });
            }
        }

        // W22: numeric constraint on non-numeric
        const inferred = defTypeToInferred(def.type);
        if ((def.min !== undefined || def.max !== undefined) && inferred !== 'number' && inferred !== 'unknown') {
            addIssue(ctx, 'warning', 'numeric_constraint_on_non_numeric',
                `Field '${id}' has min/max constraints but type is '${def.type}'`, { field_id: id });
        }

        // W23: string constraint on non-string
        if ((def.min_length !== undefined || def.max_length !== undefined || def.pattern !== undefined) &&
            inferred !== 'string' && inferred !== 'unknown') {
            addIssue(ctx, 'warning', 'string_constraint_on_non_string',
                `Field '${id}' has string constraints but type is '${def.type}'`, { field_id: id });
        }

        // Populate knownFields
        ctx.knownFields.set(id, defTypeToInferred(def.type));
    }
}

function checkRules(ctx: LintContext): void {
    const { schema } = ctx;
    if (!schema.logic_tree || schema.logic_tree.length === 0) return;

    const seenIds = new Set<string>();

    for (let i = 0; i < schema.logic_tree.length; i++) {
        const rule = schema.logic_tree[i];
        if (!rule) continue;

        const path = `logic_tree[${i}]`;

        // E4: empty rule id
        if (!rule.id || rule.id.trim() === '') {
            addIssue(ctx, 'error', 'empty_rule_id',
                `Rule at ${path} has empty or missing id`, { path });
        } else {
            // E3: duplicate rule id
            if (seenIds.has(rule.id)) {
                addIssue(ctx, 'error', 'duplicate_rule_id',
                    `Duplicate rule id '${rule.id}'`, { path, rule_id: rule.id });
            }
            seenIds.add(rule.id);
        }

        // E5: missing when/then
        if (!rule.when || (typeof rule.when === 'object' && Object.keys(rule.when).length === 0)) {
            addIssue(ctx, 'error', 'rule_missing_when',
                `Rule '${rule.id || i}' is missing a 'when' condition`, { path, rule_id: rule.id });
        }
        if (!rule.then) {
            addIssue(ctx, 'error', 'rule_missing_then',
                `Rule '${rule.id || i}' is missing a 'then' action`, { path, rule_id: rule.id });
        }
    }
}

function checkAttestations(ctx: LintContext): void {
    const { schema } = ctx;
    if (!schema.attestations) return;

    for (const [id, att] of Object.entries(schema.attestations)) {
        if (!att) continue;

        // E14: missing statement
        if (!att.statement || att.statement.trim() === '') {
            addIssue(ctx, 'error', 'attestation_missing_statement',
                `Attestation '${id}' is missing a statement`, { field_id: id });
        }
    }
}

function checkTemporalMap(ctx: LintContext): void {
    const { schema } = ctx;
    if (!schema.temporal_map || schema.temporal_map.length === 0) return;

    for (let i = 0; i < schema.temporal_map.length; i++) {
        const branch = schema.temporal_map[i];
        if (!branch) continue;

        const path = `temporal_map[${i}]`;

        // E15: missing logic_version
        if (!branch.logic_version || branch.logic_version.trim() === '') {
            addIssue(ctx, 'error', 'temporal_branch_missing_version',
                `Temporal branch at ${path} is missing logic_version`, { path });
        }

        // W21: zero-length range
        const [start, end] = branch.valid_range;
        if (start && end && start === end) {
            addIssue(ctx, 'warning', 'temporal_branch_zero_length',
                `Temporal branch at ${path} has zero-length range (start == end)`, { path });
        }

        // W20: overlapping ranges with adjacent branches
        if (i > 0) {
            const prev = schema.temporal_map[i - 1];
            if (prev) {
                const prevEnd = prev.valid_range[1]
                    ? new Date(prev.valid_range[1]).getTime()
                    : Infinity;
                const currStart = start
                    ? new Date(start).getTime()
                    : -Infinity;

                if (currStart <= prevEnd && !isNaN(prevEnd) && !isNaN(currStart)) {
                    addIssue(ctx, 'warning', 'temporal_branch_overlap',
                        `Temporal branch ${i} overlaps with branch ${i - 1}`, { path });
                }
            }
        }
    }

    // W17: orphaned logic versions (rule versions not in temporal map)
    if (schema.logic_tree) {
        const temporalVersions = new Set(
            schema.temporal_map
                .filter((b): b is TemporalBranch => !!b)
                .map(b => b.logic_version)
        );

        for (let i = 0; i < schema.logic_tree.length; i++) {
            const rule = schema.logic_tree[i];
            if (!rule || !rule.logic_version) continue;

            if (!temporalVersions.has(rule.logic_version)) {
                addIssue(ctx, 'warning', 'orphaned_logic_version',
                    `Rule '${rule.id}' references logic_version '${rule.logic_version}' which is not in temporal_map`, {
                        path: `logic_tree[${i}]`,
                        rule_id: rule.id,
                    });
            }
        }
    }
}

function checkDerived(ctx: LintContext): void {
    const { schema } = ctx;
    if (!schema.state_model?.derived) return;

    const derived = schema.state_model.derived;

    for (const [name, def] of Object.entries(derived)) {
        if (!def) continue;

        // E13: missing eval
        if (!def.eval || (typeof def.eval === 'object' && Object.keys(def.eval).length === 0)) {
            addIssue(ctx, 'error', 'derived_missing_eval',
                `Derived field '${name}' has no eval expression`, { field_id: name });
        }
    }

    // W27: inputs reference undefined definitions
    if (schema.state_model.inputs) {
        for (const input of schema.state_model.inputs) {
            if (!ctx.knownFields.has(input)) {
                addIssue(ctx, 'warning', 'input_references_undefined',
                    `state_model.inputs references undefined field '${input}'`, { field_id: input });
            }
        }
    }

    // Register derived fields in knownFields (infer type from eval expression)
    for (const [name, def] of Object.entries(derived)) {
        if (!def?.eval) continue;
        // If the derived field already exists as a definition, keep that type
        if (!ctx.knownFields.has(name)) {
            const inferredType = inferExprType(def.eval, ctx, `state_model.derived.${name}.eval`);
            ctx.knownFields.set(name, inferredType);
        }
    }

    // E12: cycle detection via three-color DFS
    checkDerivedCycles(ctx);
}

function checkDerivedCycles(ctx: LintContext): void {
    const { schema } = ctx;
    if (!schema.state_model?.derived) return;

    const derived = schema.state_model.derived;
    const derivedNames = new Set(Object.keys(derived));

    // Build adjacency list (only edges to other derived fields)
    const graph = new Map<string, Set<string>>();
    for (const [name, def] of Object.entries(derived)) {
        if (!def?.eval) continue;
        const refs = new Set<string>();
        collectVarRefs(def.eval, refs);
        // Only keep edges to other derived fields
        const edges = new Set<string>();
        for (const ref of refs) {
            if (derivedNames.has(ref) && ref !== name) {
                edges.add(ref);
            }
        }
        graph.set(name, edges);
    }

    // Three-color DFS: 0=white, 1=gray, 2=black
    const color = new Map<string, number>();
    const parent = new Map<string, string>();

    for (const name of derivedNames) {
        color.set(name, 0);
    }

    function dfs(node: string): string[] | null {
        color.set(node, 1); // gray
        const edges = graph.get(node);
        if (edges) {
            for (const neighbor of edges) {
                const neighborColor = color.get(neighbor) ?? 0;
                if (neighborColor === 1) {
                    // Found a cycle — reconstruct path
                    const cycle = [neighbor, node];
                    let cur = node;
                    while (cur !== neighbor) {
                        const p = parent.get(cur);
                        if (!p || p === neighbor) break;
                        cycle.push(p);
                        cur = p;
                    }
                    cycle.reverse();
                    return cycle;
                }
                if (neighborColor === 0) {
                    parent.set(neighbor, node);
                    const result = dfs(neighbor);
                    if (result) return result;
                }
            }
        }
        color.set(node, 2); // black
        return null;
    }

    for (const name of derivedNames) {
        if (color.get(name) === 0) {
            const cycle = dfs(name);
            if (cycle) {
                const cyclePath = cycle.join(' -> ');
                addIssue(ctx, 'error', 'derived_circular_dependency',
                    `Circular dependency in derived fields: ${cyclePath}`, {
                        field_id: cycle[0],
                    });
                return; // Report first cycle only
            }
        }
    }
}

function checkExpressions(ctx: LintContext): void {
    const { schema } = ctx;

    // Track which fields are set by which rules (for W24)
    const fieldsSetBy = new Map<string, string[]>();

    if (schema.logic_tree) {
        for (let i = 0; i < schema.logic_tree.length; i++) {
            const rule = schema.logic_tree[i];
            if (!rule) continue;

            const rulePath = `logic_tree[${i}]`;

            // Walk the when expression (E11 arithmetic_type_mismatch, W25 comparison, W16 undefined_variable, W28 unknown_operator)
            if (rule.when) {
                inferExprType(rule.when, ctx, `${rulePath}.when`);
            }

            // Walk the then action
            if (rule.then) {
                checkAction(ctx, rule.then, rulePath, rule.id, fieldsSetBy);
            }
        }
    }

    // W24: multiple rules set the same field
    for (const [field, ruleIds] of fieldsSetBy) {
        if (ruleIds.length > 1) {
            addIssue(ctx, 'warning', 'multiple_rules_set_field',
                `Field '${field}' is set by multiple rules: ${ruleIds.join(', ')}`, {
                    field_id: field,
                });
        }
    }
}

function checkAction(
    ctx: LintContext,
    action: Action,
    rulePath: string,
    ruleId: string,
    fieldsSetBy: Map<string, string[]>
): void {
    if (action.set) {
        for (const [field, expr] of Object.entries(action.set)) {
            // W19: set targets a field not in definitions or derived
            if (!ctx.knownFields.has(field)) {
                addIssue(ctx, 'warning', 'set_undefined_field',
                    `Rule '${ruleId}' sets undefined field '${field}'`, {
                        path: `${rulePath}.then.set.${field}`,
                        field_id: field,
                        rule_id: ruleId,
                    });
            }

            // Track for W24
            const existing = fieldsSetBy.get(field);
            if (existing) {
                existing.push(ruleId);
            } else {
                fieldsSetBy.set(field, [ruleId]);
            }

            // W26: set type mismatch — infer expression type and compare to field type
            const exprType = inferExprType(expr, ctx, `${rulePath}.then.set.${field}`);
            const fieldType = ctx.knownFields.get(field);
            if (fieldType && fieldType !== 'unknown' && exprType !== 'unknown' && exprType !== fieldType) {
                addIssue(ctx, 'warning', 'set_type_mismatch',
                    `Rule '${ruleId}' sets '${field}' (${fieldType}) to a ${exprType} expression`, {
                        path: `${rulePath}.then.set.${field}`,
                        field_id: field,
                        rule_id: ruleId,
                    });
            }
        }
    }

    if (action.ui_modify) {
        for (const field of Object.keys(action.ui_modify)) {
            // W18: ui_modify targets undefined field
            if (!ctx.knownFields.has(field)) {
                addIssue(ctx, 'warning', 'ui_modify_undefined_field',
                    `Rule '${ruleId}' modifies UI of undefined field '${field}'`, {
                        path: `${rulePath}.then.ui_modify.${field}`,
                        field_id: field,
                        rule_id: ruleId,
                    });
            }
        }
    }
}

function checkFieldConflicts(ctx: LintContext): void {
    // W24 is handled in checkExpressions
}

function checkDuplicateIds(ctx: LintContext): void {
    const { schema } = ctx;

    // Collect IDs across namespaces
    const seen = new Map<string, string[]>();

    function track(id: string, namespace: string): void {
        const existing = seen.get(id);
        if (existing) {
            existing.push(namespace);
        } else {
            seen.set(id, [namespace]);
        }
    }

    // Definition keys
    if (schema.definitions) {
        for (const id of Object.keys(schema.definitions)) {
            track(id, 'definition');
        }
    }

    // Rule IDs
    if (schema.logic_tree) {
        for (const rule of schema.logic_tree) {
            if (rule?.id) {
                track(rule.id, 'rule');
            }
        }
    }

    // Attestation keys
    if (schema.attestations) {
        for (const id of Object.keys(schema.attestations)) {
            track(id, 'attestation');
        }
    }

    // Derived field keys
    if (schema.state_model?.derived) {
        for (const id of Object.keys(schema.state_model.derived)) {
            track(id, 'derived');
        }
    }

    // Temporal branch logic_versions
    if (schema.temporal_map) {
        for (const branch of schema.temporal_map) {
            if (branch?.logic_version) {
                track(branch.logic_version, 'temporal_version');
            }
        }
    }

    // E29: flag any ID appearing in more than one namespace
    for (const [id, namespaces] of seen) {
        const unique = [...new Set(namespaces)];
        if (unique.length > 1) {
            addIssue(ctx, 'error', 'duplicate_id_across_namespaces',
                `ID '${id}' appears in multiple namespaces: ${unique.join(', ')}`, {
                    field_id: id,
                });
        }
    }
}

// ============================================================
// Public API
// ============================================================

export function lint(schema: TenetSchema | string): LintResult {
    // Parse
    let parsed: TenetSchema;
    try {
        parsed = typeof schema === 'string' ? JSON.parse(schema) : schema;
    } catch (e) {
        return {
            valid: false,
            issues: [{
                severity: 'error',
                code: 'parse_error',
                message: `Failed to parse schema: ${e instanceof Error ? e.message : String(e)}`,
            }],
        };
    }

    const ctx: LintContext = {
        schema: parsed,
        issues: [],
        knownFields: new Map(),
        supportedOps: new Set(SUPPORTED_OPS),
    };

    // Run check pipeline in order
    checkDefinitions(ctx);
    checkRules(ctx);
    checkAttestations(ctx);
    checkTemporalMap(ctx);
    checkDerived(ctx);       // Must run before checkExpressions — updates knownFields for derived
    checkExpressions(ctx);
    checkFieldConflicts(ctx);
    checkDuplicateIds(ctx);

    const hasErrors = ctx.issues.some(i => i.severity === 'error');

    return {
        valid: !hasErrors,
        issues: ctx.issues,
    };
}
