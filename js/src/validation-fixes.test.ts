/**
 * Tests for validation fixes:
 * - Issue 1: Empty string passes required validation
 * - Issue 2: Derived fields shadowed by definitions
 * - Issue 3: Execution order - logic before derived
 */

import { describe, it } from 'node:test';
import assert from 'node:assert';
import { run } from './index.js';

// ===========================================================================
// Issue 1: Empty String Required Validation
// ===========================================================================

describe('Issue 1: Empty String Required Validation', () => {
    it('should treat empty string as missing for required string fields', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                name: { type: 'string' as const, value: '', required: true },
            },
        };

        const result = run(schema);
        assert.ok(!result.error, `Run failed: ${result.error}`);
        assert.ok(result.result);

        // Status should be INCOMPLETE because empty string is "missing"
        assert.strictEqual(
            result.result.status,
            'INCOMPLETE',
            `Expected status INCOMPLETE for empty required string, got ${result.result.status}`
        );

        // Should have an error for the missing field
        assert.ok(result.result.errors && result.result.errors.length > 0, 'Expected error for empty required string');
    });

    it('should accept empty string for non-required fields', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                notes: { type: 'string' as const, value: '', required: false },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Status should be READY because field is not required
        assert.strictEqual(result.result.status, 'READY');
    });

    it('should accept zero for required number fields', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                quantity: { type: 'number' as const, value: 0, required: true },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Status should be READY (0 is a valid value for required number)
        assert.strictEqual(result.result.status, 'READY');
    });

    it('should catch empty allergy_note in survey schema', () => {
        const schema = {
            protocol: 'CoffeePreferenceSurvey_v1',
            schema_id: 'coffee-pref-001',
            definitions: {
                respondent_name: {
                    type: 'string' as const,
                    label: 'Your Name',
                    required: true,
                    value: 'Jane Doe',
                },
                allergy_note: {
                    type: 'string' as const,
                    label: 'Please describe your allergy',
                    required: true,
                    visible: true,
                    value: '',
                },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Should be INCOMPLETE because allergy_note is required but empty
        assert.strictEqual(result.result.status, 'INCOMPLETE');
    });
});

// ===========================================================================
// Issue 2: Derived Fields Shadowing
// ===========================================================================

describe('Issue 2: Derived Fields Take Precedence', () => {
    it('should compute derived value even when field exists in definitions', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                gross: { type: 'number' as const, value: 100 },
                tax: { type: 'number' as const, value: null, readonly: true },
            },
            state_model: {
                inputs: ['gross'],
                derived: {
                    tax: { eval: { '*': [{ var: 'gross' }, 0.1] } },
                },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Tax should be computed as 10 (100 * 0.1)
        const taxDef = result.result.definitions.tax;
        assert.ok(taxDef, 'Expected tax definition to exist');
        assert.strictEqual(taxDef.value, 10, `Expected tax = 10, got ${taxDef.value}`);
    });

    it('should allow logic tree to use derived values', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                gross: { type: 'number' as const, value: 100 },
            },
            state_model: {
                inputs: ['gross'],
                derived: {
                    tax: { eval: { '*': [{ var: 'gross' }, 0.1] } },
                },
            },
            logic_tree: [
                {
                    id: 'check_tax',
                    when: { '>': [{ var: 'tax' }, 5] },
                    then: { set: { high_tax: true } },
                },
            ],
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // high_tax should be set because tax (10) > 5
        const highTaxDef = result.result.definitions.high_tax;
        assert.ok(highTaxDef, 'Expected high_tax definition to be created');
        assert.strictEqual(highTaxDef.value, true, `Expected high_tax = true, got ${highTaxDef.value}`);
    });
});

// ===========================================================================
// Issue 3: Execution Order (Logic Before Derived)
// ===========================================================================

describe('Issue 3: Logic Can Use Derived Values', () => {
    it('should compute derived values before logic tree evaluation', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                income: { type: 'number' as const, value: 50000 },
                deductions: { type: 'number' as const, value: 10000 },
            },
            state_model: {
                inputs: ['income', 'deductions'],
                derived: {
                    taxable_income: { eval: { '-': [{ var: 'income' }, { var: 'deductions' }] } },
                },
            },
            logic_tree: [
                {
                    id: 'high_income_bracket',
                    when: { '>': [{ var: 'taxable_income' }, 30000] },
                    then: { set: { tax_bracket: 'high' } },
                },
            ],
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // taxable_income should be 40000 (50000 - 10000)
        const taxableIncomeDef = result.result.definitions.taxable_income;
        assert.ok(taxableIncomeDef, 'Expected taxable_income definition');
        assert.strictEqual(taxableIncomeDef.value, 40000);

        // tax_bracket should be "high" because taxable_income (40000) > 30000
        const taxBracketDef = result.result.definitions.tax_bracket;
        assert.ok(taxBracketDef, 'Expected tax_bracket to be set by logic rule');
        assert.strictEqual(taxBracketDef.value, 'high');
    });

    it('should calculate effective tax rate in tax calculator schema', () => {
        const schema = {
            protocol: 'IncomeTaxCalculator_v1',
            schema_id: 'tax-calc-001',
            definitions: {
                gross_annual_income: { type: 'currency' as const, value: 85000 },
                filing_status: {
                    type: 'select' as const,
                    options: ['single', 'married_joint', 'married_separate'],
                    value: 'single',
                },
                standard_deduction: { type: 'currency' as const, readonly: true, value: null },
                taxable_income: { type: 'currency' as const, readonly: true, value: null },
                effective_tax_rate: { type: 'number' as const, readonly: true, value: null },
            },
            state_model: {
                inputs: ['gross_annual_income', 'filing_status'],
                derived: {
                    standard_deduction: {
                        eval: {
                            if: [
                                { '==': [{ var: 'filing_status' }, 'single'] },
                                14600,
                                { '==': [{ var: 'filing_status' }, 'married_joint'] },
                                29200,
                                21900,
                            ],
                        },
                    },
                    taxable_income: {
                        eval: {
                            '-': [{ var: 'gross_annual_income' }, { var: 'standard_deduction' }],
                        },
                    },
                },
            },
            logic_tree: [
                {
                    id: 'calc_effective_rate',
                    when: { '>': [{ var: 'taxable_income' }, 0] },
                    then: {
                        set: {
                            effective_tax_rate: {
                                '/': [
                                    { '*': [{ var: 'taxable_income' }, 0.22] },
                                    { var: 'gross_annual_income' },
                                ],
                            },
                        },
                    },
                },
            ],
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Check standard_deduction is 14600 for single
        const stdDedDef = result.result.definitions.standard_deduction;
        assert.ok(stdDedDef, 'Expected standard_deduction definition');
        assert.strictEqual(stdDedDef.value, 14600);

        // Check taxable_income is 70400 (85000 - 14600)
        const taxableIncomeDef = result.result.definitions.taxable_income;
        assert.ok(taxableIncomeDef, 'Expected taxable_income definition');
        assert.strictEqual(taxableIncomeDef.value, 70400);

        // Check effective_tax_rate is calculated (70400 * 0.22 / 85000 = ~0.182)
        const effectiveRateDef = result.result.definitions.effective_tax_rate;
        assert.ok(effectiveRateDef, 'Expected effective_tax_rate to be calculated');
        const effectiveVal = effectiveRateDef.value as number;
        assert.ok(effectiveVal > 0.18 && effectiveVal < 0.19, `Expected effective_tax_rate around 0.182, got ${effectiveVal}`);
    });
});

// ===========================================================================
// Edge Cases & Complex Scenarios
// ===========================================================================

describe('Edge Cases', () => {
    it('should handle chained derived fields', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                base: { type: 'number' as const, value: 100 },
            },
            state_model: {
                inputs: ['base'],
                derived: {
                    level1: { eval: { '*': [{ var: 'base' }, 2] } },
                    level2: { eval: { '*': [{ var: 'level1' }, 3] } },
                },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // level1 = 100 * 2 = 200
        const level1Def = result.result.definitions.level1;
        assert.ok(level1Def);
        assert.strictEqual(level1Def.value, 200);

        // level2 = 200 * 3 = 600
        const level2Def = result.result.definitions.level2;
        assert.ok(level2Def);
        assert.strictEqual(level2Def.value, 600);
    });

    it('should re-compute derived after logic modifies inputs', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                discount_eligible: { type: 'boolean' as const, value: false },
                base_price: { type: 'number' as const, value: 100 },
            },
            state_model: {
                inputs: ['discount_eligible', 'base_price'],
                derived: {
                    final_price: {
                        eval: {
                            if: [
                                { var: 'discount_eligible' },
                                { '*': [{ var: 'base_price' }, 0.9] },
                                { var: 'base_price' },
                            ],
                        },
                    },
                },
            },
            logic_tree: [
                {
                    id: 'apply_discount',
                    when: { '>': [{ var: 'base_price' }, 50] },
                    then: { set: { discount_eligible: true } },
                },
            ],
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // discount_eligible should be true
        const discountDef = result.result.definitions.discount_eligible;
        assert.ok(discountDef);
        assert.strictEqual(discountDef.value, true);

        // final_price should be 90 (100 * 0.9) because discount_eligible was set by logic
        const finalPriceDef = result.result.definitions.final_price;
        assert.ok(finalPriceDef);
        assert.strictEqual(finalPriceDef.value, 90);
    });

    it('should catch multiple required empty strings', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                first_name: { type: 'string' as const, value: '', required: true },
                last_name: { type: 'string' as const, value: '', required: true },
                nickname: { type: 'string' as const, value: '', required: false },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // Status should be INCOMPLETE
        assert.strictEqual(result.result.status, 'INCOMPLETE');

        // Should have errors for both first_name and last_name
        assert.ok(result.result.errors && result.result.errors.length >= 2, 'Expected at least 2 errors');
    });

    it('should use derived value when definition has null value', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                input_a: { type: 'number' as const, value: 10 },
                input_b: { type: 'number' as const, value: 20 },
                result: { type: 'number' as const, value: null, readonly: true },
            },
            state_model: {
                inputs: ['input_a', 'input_b'],
                derived: {
                    result: { eval: { '+': [{ var: 'input_a' }, { var: 'input_b' }] } },
                },
            },
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // result should be 30 (10 + 20), not null
        const resultDef = result.result.definitions.result;
        assert.ok(resultDef);
        assert.strictEqual(resultDef.value, 30);
    });

    it('should compare two derived values in logic', () => {
        const schema = {
            protocol: 'Test_v1',
            schema_id: 'test',
            definitions: {
                price_a: { type: 'number' as const, value: 100 },
                price_b: { type: 'number' as const, value: 80 },
            },
            state_model: {
                inputs: ['price_a', 'price_b'],
                derived: {
                    discounted_a: { eval: { '*': [{ var: 'price_a' }, 0.8] } },
                    discounted_b: { eval: { '*': [{ var: 'price_b' }, 0.9] } },
                },
            },
            logic_tree: [
                {
                    id: 'compare_prices',
                    when: { '>': [{ var: 'discounted_a' }, { var: 'discounted_b' }] },
                    then: { set: { best_deal: 'B' } },
                },
            ],
        };

        const result = run(schema);
        assert.ok(!result.error);
        assert.ok(result.result);

        // discounted_a = 100 * 0.8 = 80
        // discounted_b = 80 * 0.9 = 72
        // 80 > 72, so best_deal = "B"
        const bestDealDef = result.result.definitions.best_deal;
        assert.ok(bestDealDef, 'Expected best_deal to be set');
        assert.strictEqual(bestDealDef.value, 'B');
    });
});
