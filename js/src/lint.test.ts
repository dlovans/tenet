/**
 * Tests for the lint() static schema analyzer.
 * Organized into: Errors, Warnings, Valid schemas, Edge cases.
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';
import { lint } from './index.js';
import type { TenetSchema } from './index.js';

// Helper: build a minimal valid schema, then override
function base(overrides?: Partial<TenetSchema>): TenetSchema {
    return {
        definitions: {
            name: { type: 'string', value: 'test' },
        },
        ...overrides,
    };
}

// Helper: assert a specific issue code exists
function expectIssue(result: ReturnType<typeof lint>, code: string, severity?: 'error' | 'warning') {
    const found = result.issues.find(i => i.code === code && (!severity || i.severity === severity));
    assert.ok(found, `Expected issue '${code}' (${severity ?? 'any'}) but got: ${JSON.stringify(result.issues.map(i => i.code))}`);
    return found;
}

function expectNoIssue(result: ReturnType<typeof lint>, code: string) {
    const found = result.issues.find(i => i.code === code);
    assert.ok(!found, `Did not expect issue '${code}' but found: ${found?.message}`);
}

// ===========================================================================
// Group 1: Errors (valid: false)
// ===========================================================================

describe('Lint Errors', () => {
    it('E1: no_definitions — empty definitions object', () => {
        const result = lint({ definitions: {} } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'no_definitions', 'error');
    });

    it('E2: invalid_field_type — unknown type', () => {
        const result = lint({
            definitions: {
                field1: { type: 'potato' as any, value: 'x' },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'invalid_field_type', 'error');
    });

    it('E3: duplicate_rule_id — two rules with same id', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: { '==': [1, 1] }, then: { set: { name: 'a' } } },
                { id: 'r1', when: { '==': [2, 2] }, then: { set: { name: 'b' } } },
            ],
        }));
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'duplicate_rule_id', 'error');
    });

    it('E4: empty_rule_id — rule with empty id', () => {
        const result = lint(base({
            logic_tree: [
                { id: '', when: { '==': [1, 1] }, then: { set: { name: 'a' } } },
            ],
        }));
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'empty_rule_id', 'error');
    });

    it('E5: rule_missing_when and rule_missing_then', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: {}, then: { set: { name: 'a' } } },
                { id: 'r2', when: { '==': [1, 1] } } as any,
            ],
        }));
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'rule_missing_when', 'error');
        expectIssue(result, 'rule_missing_then', 'error');
    });

    it('E6: select_missing_options — select without options', () => {
        const result = lint({
            definitions: {
                color: { type: 'select', value: 'red' },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'select_missing_options', 'error');
    });

    it('E7: select_options_empty — options is empty array', () => {
        const result = lint({
            definitions: {
                color: { type: 'select', value: 'red', options: [] },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'select_options_empty', 'error');
    });

    it('E8: min_exceeds_max', () => {
        const result = lint({
            definitions: {
                price: { type: 'number', value: 50, min: 100, max: 50 },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'min_exceeds_max', 'error');
    });

    it('E9: min_length_exceeds_max_length', () => {
        const result = lint({
            definitions: {
                code: { type: 'string', value: 'abc', min_length: 10, max_length: 5 },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'min_length_exceeds_max_length', 'error');
    });

    it('E10: invalid_regex', () => {
        const result = lint({
            definitions: {
                code: { type: 'string', value: 'abc', pattern: '[unclosed' },
            },
        });
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'invalid_regex', 'error');
    });

    it('E11: arithmetic_type_mismatch — string + number', () => {
        const result = lint({
            definitions: {
                label: { type: 'string', value: 'hello' },
                count: { type: 'number', value: 5 },
            },
            logic_tree: [
                {
                    id: 'r1',
                    when: { '==': [1, 1] },
                    then: { set: { result: { '+': [{ var: 'label' }, { var: 'count' }] } } },
                },
            ],
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'arithmetic_type_mismatch', 'error');
    });

    it('E11b: arithmetic_type_mismatch — boolean * number', () => {
        const result = lint({
            definitions: {
                flag: { type: 'boolean', value: true },
                amount: { type: 'number', value: 10 },
            },
            logic_tree: [
                {
                    id: 'r1',
                    when: { '==': [1, 1] },
                    then: { set: { result: { '*': [{ var: 'flag' }, 2] } } },
                },
            ],
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'arithmetic_type_mismatch', 'error');
    });

    it('E12: derived_circular_dependency — A uses B, B uses A', () => {
        const result = lint({
            definitions: {
                input: { type: 'number', value: 1 },
            },
            state_model: {
                inputs: ['input'],
                derived: {
                    a: { eval: { '+': [{ var: 'b' }, 1] } },
                    b: { eval: { '+': [{ var: 'a' }, 1] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'derived_circular_dependency', 'error');
    });

    it('E13: derived_missing_eval — empty eval', () => {
        const result = lint({
            definitions: {
                input: { type: 'number', value: 1 },
            },
            state_model: {
                inputs: ['input'],
                derived: {
                    broken: { eval: {} as any },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'derived_missing_eval', 'error');
    });

    it('E14: attestation_missing_statement', () => {
        const result = lint(base({
            attestations: {
                confirm: { statement: '', required: true },
            },
        }));
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'attestation_missing_statement', 'error');
    });

    it('E15: temporal_branch_missing_version', () => {
        const result = lint(base({
            temporal_map: [
                { valid_range: ['2025-01-01', '2025-12-31'], logic_version: '', status: 'ACTIVE' },
            ],
        }));
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'temporal_branch_missing_version', 'error');
    });

    it('E29: duplicate_id_across_namespaces — definition key and derived field key collide', () => {
        const result = lint({
            definitions: {
                total: { type: 'number', value: 100 },
            },
            state_model: {
                inputs: ['total'],
                derived: {
                    total: { eval: { '+': [{ var: 'total' }, 1] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'duplicate_id_across_namespaces', 'error');
    });
});

// ===========================================================================
// Group 2: Warnings (valid: true)
// ===========================================================================

describe('Lint Warnings', () => {
    it('W16: undefined_variable — var references ghost field', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: { '==': [{ var: 'ghost' }, 1] }, then: { set: { name: 'x' } } },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'undefined_variable', 'warning');
    });

    it('W17: orphaned_logic_version — rule version not in temporal_map', () => {
        const result = lint(base({
            temporal_map: [
                { valid_range: ['2025-01-01', null], logic_version: 'v1', status: 'ACTIVE' },
            ],
            logic_tree: [
                { id: 'r1', logic_version: 'v2', when: { '==': [1, 1] }, then: { set: { name: 'x' } } },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'orphaned_logic_version', 'warning');
    });

    it('W18: ui_modify_undefined_field', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: { '==': [1, 1] }, then: { ui_modify: { ghost: { visible: false } } } },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'ui_modify_undefined_field', 'warning');
    });

    it('W19: set_undefined_field', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: { '==': [1, 1] }, then: { set: { ghost: 'val' } } },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'set_undefined_field', 'warning');
    });

    it('W20: temporal_branch_overlap', () => {
        const result = lint(base({
            temporal_map: [
                { valid_range: ['2025-01-01', '2025-06-30'], logic_version: 'v1', status: 'ACTIVE' },
                { valid_range: ['2025-03-01', '2025-12-31'], logic_version: 'v2', status: 'ACTIVE' },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'temporal_branch_overlap', 'warning');
    });

    it('W21: temporal_branch_zero_length', () => {
        const result = lint(base({
            temporal_map: [
                { valid_range: ['2025-06-01', '2025-06-01'], logic_version: 'v1', status: 'ACTIVE' },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'temporal_branch_zero_length', 'warning');
    });

    it('W22: numeric_constraint_on_non_numeric — min on string', () => {
        const result = lint({
            definitions: {
                label: { type: 'string', value: 'hi', min: 5 },
            },
        });
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'numeric_constraint_on_non_numeric', 'warning');
    });

    it('W23: string_constraint_on_non_string — min_length on number', () => {
        const result = lint({
            definitions: {
                amount: { type: 'number', value: 42, min_length: 3 },
            },
        });
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'string_constraint_on_non_string', 'warning');
    });

    it('W24: multiple_rules_set_field — two rules set same field', () => {
        const result = lint({
            definitions: {
                price: { type: 'number', value: 10 },
                flag: { type: 'boolean', value: true },
            },
            logic_tree: [
                { id: 'r1', when: { '==': [{ var: 'flag' }, true] }, then: { set: { price: 20 } } },
                { id: 'r2', when: { '==': [{ var: 'flag' }, false] }, then: { set: { price: 30 } } },
            ],
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'multiple_rules_set_field', 'warning');
    });

    it('W25: comparison_type_mismatch — string < number', () => {
        const result = lint({
            definitions: {
                label: { type: 'string', value: 'hi' },
                count: { type: 'number', value: 5 },
            },
            logic_tree: [
                { id: 'r1', when: { '<': [{ var: 'label' }, { var: 'count' }] }, then: { set: { label: 'low' } } },
            ],
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'comparison_type_mismatch', 'warning');
    });

    it('W26: set_type_mismatch — arithmetic result assigned to string field', () => {
        const result = lint({
            definitions: {
                label: { type: 'string', value: 'hi' },
                a: { type: 'number', value: 1 },
                b: { type: 'number', value: 2 },
            },
            logic_tree: [
                { id: 'r1', when: { '==': [1, 1] }, then: { set: { label: { '+': [{ var: 'a' }, { var: 'b' }] } } } },
            ],
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'set_type_mismatch', 'warning');
    });

    it('W27: input_references_undefined', () => {
        const result = lint({
            definitions: {
                price: { type: 'number', value: 10 },
            },
            state_model: {
                inputs: ['ghost'],
                derived: {
                    doubled: { eval: { '*': [{ var: 'price' }, 2] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'input_references_undefined', 'warning');
    });

    it('W28: unknown_operator — unrecognized operator', () => {
        const result = lint(base({
            logic_tree: [
                { id: 'r1', when: { 'modulo': [10, 3] }, then: { set: { name: 'x' } } },
            ],
        }));
        assert.strictEqual(result.valid, true);
        expectIssue(result, 'unknown_operator', 'warning');
    });
});

// ===========================================================================
// Group 3: Valid schemas (no issues)
// ===========================================================================

describe('Lint Valid Schemas', () => {
    it('minimal schema — one definition, no rules', () => {
        const result = lint({
            definitions: {
                name: { type: 'string', value: 'test' },
            },
        });
        assert.strictEqual(result.valid, true);
        assert.strictEqual(result.issues.length, 0);
    });

    it('full schema with derived fields, rules, attestations, temporal map', () => {
        const result = lint({
            protocol: 'Test_v1',
            schema_id: 'test-001',
            definitions: {
                income: { type: 'number', value: 50000 },
                deductions: { type: 'number', value: 10000 },
                status_msg: { type: 'string', value: '' },
            },
            state_model: {
                inputs: ['income', 'deductions'],
                derived: {
                    taxable_income: { eval: { '-': [{ var: 'income' }, { var: 'deductions' }] } },
                },
            },
            logic_tree: [
                {
                    id: 'high_income',
                    logic_version: 'v1',
                    when: { '>': [{ var: 'taxable_income' }, 30000] },
                    then: { set: { status_msg: 'high bracket' } },
                },
            ],
            temporal_map: [
                { valid_range: ['2025-01-01', null], logic_version: 'v1', status: 'ACTIVE' as const },
            ],
            attestations: {
                confirm: { statement: 'I confirm this is correct', required: true },
            },
        } as TenetSchema);

        assert.strictEqual(result.valid, true);
        assert.strictEqual(result.issues.length, 0);
    });

    it('number + currency arithmetic — compatible types, no error', () => {
        const result = lint({
            definitions: {
                price: { type: 'currency', value: 100 },
                tax_rate: { type: 'number', value: 0.1 },
            },
            state_model: {
                inputs: ['price', 'tax_rate'],
                derived: {
                    tax: { eval: { '*': [{ var: 'price' }, { var: 'tax_rate' }] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectNoIssue(result, 'arithmetic_type_mismatch');
    });
});

// ===========================================================================
// Group 4: Edge cases
// ===========================================================================

describe('Lint Edge Cases', () => {
    it('JSON string input parses correctly', () => {
        const json = JSON.stringify({
            definitions: { x: { type: 'number', value: 1 } },
        });
        const result = lint(json);
        assert.strictEqual(result.valid, true);
        assert.strictEqual(result.issues.length, 0);
    });

    it('invalid JSON string → parse_error', () => {
        const result = lint('not json {{{');
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'parse_error', 'error');
    });

    it('nested expression inference: all-numeric chain produces no error', () => {
        const result = lint({
            definitions: {
                a: { type: 'number', value: 1 },
                b: { type: 'number', value: 2 },
                c: { type: 'number', value: 3 },
            },
            state_model: {
                inputs: ['a', 'b', 'c'],
                derived: {
                    result: { eval: { '*': [{ '+': [{ var: 'a' }, { var: 'b' }] }, { var: 'c' }] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, true);
        expectNoIssue(result, 'arithmetic_type_mismatch');
    });

    it('deep cycle: A→B→C→A', () => {
        const result = lint({
            definitions: {
                input: { type: 'number', value: 1 },
            },
            state_model: {
                inputs: ['input'],
                derived: {
                    a: { eval: { '+': [{ var: 'b' }, 1] } },
                    b: { eval: { '+': [{ var: 'c' }, 1] } },
                    c: { eval: { '+': [{ var: 'a' }, 1] } },
                },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'derived_circular_dependency', 'error');
    });

    it('unknown types do not trigger false errors', () => {
        // An if-expression returning unknown type used in arithmetic — no error
        const result = lint({
            definitions: {
                flag: { type: 'boolean', value: true },
                a: { type: 'number', value: 1 },
            },
            logic_tree: [
                {
                    id: 'r1',
                    when: { '==': [1, 1] },
                    then: {
                        set: {
                            result: {
                                '+': [
                                    { var: 'a' },
                                    // if-expression: type inferred from first then-branch (number)
                                    { 'if': [{ var: 'flag' }, 10, 20] },
                                ],
                            },
                        },
                    },
                },
            ],
        } as TenetSchema);
        // The if returns number (inferred from first then-branch=10), so no error
        expectNoIssue(result, 'arithmetic_type_mismatch');
    });

    it('warnings-only → valid: true', () => {
        const result = lint({
            definitions: {
                label: { type: 'string', value: 'hi', min: 5 },
            },
        });
        assert.strictEqual(result.valid, true);
        assert.ok(result.issues.length > 0, 'Expected at least one warning');
        assert.ok(result.issues.every(i => i.severity === 'warning'));
    });

    it('derived field not in definitions gets registered via inference', () => {
        const result = lint({
            definitions: {
                a: { type: 'number', value: 10 },
                b: { type: 'number', value: 20 },
            },
            state_model: {
                inputs: ['a', 'b'],
                derived: {
                    total: { eval: { '+': [{ var: 'a' }, { var: 'b' }] } },
                },
            },
            logic_tree: [
                {
                    id: 'r1',
                    when: { '>': [{ var: 'total' }, 25] },
                    then: { set: { a: 0 } },
                },
            ],
        } as TenetSchema);
        // 'total' is derived → should be known. No undefined_variable warning.
        expectNoIssue(result, 'undefined_variable');
    });

    it('self-referencing derived field is not flagged as cycle', () => {
        // self-references are filtered out in cycle detection (ref !== name)
        const result = lint({
            definitions: {
                input: { type: 'number', value: 1 },
            },
            state_model: {
                inputs: ['input'],
                derived: {
                    acc: { eval: { '+': [{ var: 'acc' }, { var: 'input' }] } },
                },
            },
        } as TenetSchema);
        expectNoIssue(result, 'derived_circular_dependency');
    });

    it('E29: rule id collides with attestation key', () => {
        const result = lint({
            definitions: {
                name: { type: 'string', value: 'test' },
            },
            logic_tree: [
                { id: 'confirm', when: { '==': [1, 1] }, then: { set: { name: 'ok' } } },
            ],
            attestations: {
                confirm: { statement: 'I confirm', required: true },
            },
        } as TenetSchema);
        assert.strictEqual(result.valid, false);
        expectIssue(result, 'duplicate_id_across_namespaces', 'error');
    });
});
