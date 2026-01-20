# Tenet VM Complete Specification

**Version:** 1.0  
**Last Updated:** January 2026

---

## What is Tenet?

Tenet is a **logic engine** that processes JSON documents according to rules you define. Think of it as a smart form processor that can:

- Validate data against constraints
- Show/hide fields based on conditions
- Calculate values automatically
- Enforce that required signatures are collected
- Apply different rules based on dates

**What Tenet is NOT:**
- It is NOT a database
- It is NOT a signing service (it validates signatures, not creates them)
- It is NOT a blockchain or cryptographic system

---

## How It Works

```
Input (JSON Schema) → [Tenet VM] → Output (Processed JSON)
```

1. You provide a JSON document with field definitions and rules
2. Tenet evaluates all rules in order
3. Tenet returns the document with computed values, validation errors, and a status

---

## Document Status

Every processed document gets one of three statuses:

| Status | Meaning |
|--------|---------|
| `READY` | All validations pass. Document is complete. |
| `INCOMPLETE` | Missing required fields or unsigned attestations. |
| `INVALID` | Type errors, constraint violations, or rule violations. |

---

# Schema Structure

A Tenet schema is a JSON object with these top-level fields:

```json
{
  "$schema": "https://tenet.dev/schema/v1.json",
  "protocol": "Tenet_v1.0",
  "schema_id": "loan_application_v2",
  "version": "2026.01.20",
  "valid_from": "2026-01-01",
  
  "definitions": { },
  "logic_tree": [ ],
  "state_model": { },
  "attestations": { },
  "temporal_map": [ ]
}
```

## Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `$schema` | No | URL for IDE autocompletion |
| `protocol` | No | Version identifier (e.g., "Tenet_v1.0") |
| `schema_id` | No | Unique ID for this schema type |
| `version` | No | Your version number |
| `valid_from` | No | When this schema becomes effective |
| `definitions` | **Yes** | Field definitions (see below) |
| `logic_tree` | No | Rules to evaluate |
| `state_model` | No | Computed/derived values |
| `attestations` | No | Signature requirements |
| `temporal_map` | No | Date-based rule versioning |

---

# Definitions

Definitions are the fields in your document. Each field has a name and properties.

## Basic Structure

```json
"definitions": {
  "field_name": {
    "type": "string",
    "value": null,
    "label": "Field Label",
    "required": true,
    "visible": true
  }
}
```

## Field Types

| Type | Description | Example Value |
|------|-------------|---------------|
| `string` | Text | `"John Doe"` |
| `number` | Numeric value | `42`, `3.14` |
| `boolean` | True or false | `true`, `false` |
| `select` | One of predefined options | `"employed"` |
| `date` | ISO date string | `"2026-01-20"` |
| `currency` | Money value | `1500.00` |
| `attestation` | Legacy signature field (prefer top-level attestations) | `true` |

## Field Properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `type` | string | — | Required. One of the types above. |
| `value` | any | `null` | Current value of the field. |
| `label` | string | — | Human-readable label for UI. |
| `required` | boolean | `false` | Must have a value for READY status. |
| `visible` | boolean | `true` | Whether field is shown in UI. |
| `readonly` | boolean | `false` | Computed field, cannot be edited by user. |
| `options` | string[] | — | Valid values for `select` type. |
| `min` | number | — | Minimum value (for numbers). |
| `max` | number | — | Maximum value (for numbers). |
| `min_length` | integer | — | Minimum string length. |
| `max_length` | integer | — | Maximum string length. |
| `pattern` | string | — | Regex pattern for validation. |
| `step` | number | — | Increment step (for UI). |
| `ui_class` | string | — | CSS class for styling. |
| `ui_message` | string | — | Help text or warning. |

## Example: Complete Definition

```json
"definitions": {
  "applicant_name": {
    "type": "string",
    "value": null,
    "label": "Full Legal Name",
    "required": true,
    "visible": true,
    "min_length": 2,
    "max_length": 100
  },
  "annual_income": {
    "type": "number",
    "value": null,
    "label": "Annual Income ($)",
    "required": true,
    "min": 0,
    "max": 10000000
  },
  "employment_status": {
    "type": "select",
    "value": null,
    "label": "Employment Status",
    "required": true,
    "options": ["employed", "self_employed", "unemployed", "retired"]
  },
  "application_date": {
    "type": "date",
    "value": null,
    "label": "Date of Application"
  },
  "loan_approved": {
    "type": "boolean",
    "value": false,
    "readonly": true
  }
}
```

---

# Logic Tree (Rules)

The logic tree contains rules that fire when conditions are met.

## Rule Structure

```json
"logic_tree": [
  {
    "id": "rule_unique_id",
    "law_ref": "Optional: Legal Citation",
    "when": { /* condition */ },
    "then": { /* action */ },
    "disabled": false
  }
]
```

## Rule Properties

| Property | Required | Description |
|----------|----------|-------------|
| `id` | Yes | Unique identifier for this rule |
| `law_ref` | No | Legal citation (e.g., "26 U.S.C. § 61") |
| `when` | Yes | JSON-logic condition |
| `then` | Yes | Actions to take if condition is true |
| `disabled` | No | Skip this rule if true |

## Actions (the `then` block)

| Action | Description |
|--------|-------------|
| `set` | Set field values |
| `ui_modify` | Change field visibility or requirements |
| `error_msg` | Add a validation error |

## Example Rules

### Set a value based on condition

```json
{
  "id": "calculate_tax_bracket",
  "when": {"<": [{"var": "annual_income"}, 50000]},
  "then": {
    "set": {"tax_bracket": "low"}
  }
}
```

### Show/hide fields dynamically

```json
{
  "id": "show_business_fields",
  "when": {"==": [{"var": "employment_status"}, "self_employed"]},
  "then": {
    "ui_modify": {
      "business_name": {"visible": true, "required": true},
      "business_ein": {"visible": true, "required": true}
    }
  }
}
```

### Add validation error with legal reference

```json
{
  "id": "unemployed_denial",
  "law_ref": "Lending Standards Act § 4.2",
  "when": {"==": [{"var": "employment_status"}, "unemployed"]},
  "then": {
    "set": {"loan_approved": false},
    "error_msg": "Applicant must be employed per Lending Standards Act § 4.2"
  }
}
```

---

# Operators (JSON-Logic)

Conditions and expressions use JSON-logic syntax.

## Reading a Field Value

```json
{"var": "field_name"}
```

Nested access: `{"var": "user.address.city"}`

## Comparison Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `==` | Equals | `{"==": [{"var": "status"}, "active"]}` |
| `!=` | Not equals | `{"!=": [{"var": "status"}, "deleted"]}` |
| `<` | Less than | `{"<": [{"var": "age"}, 18]}` |
| `>` | Greater than | `{">": [{"var": "amount"}, 1000]}` |
| `<=` | Less or equal | `{"<=": [{"var": "score"}, 100]}` |
| `>=` | Greater or equal | `{">=": [{"var": "income"}, 30000]}` |

## Logical Operators

| Operator | Example |
|----------|---------|
| `and` | `{"and": [cond1, cond2]}` |
| `or` | `{"or": [cond1, cond2]}` |
| `not` | `{"not": condition}` |

## Arithmetic Operators

| Operator | Example |
|----------|---------|
| `+` | `{"+": [{"var": "a"}, {"var": "b"}]}` |
| `-` | `{"-": [{"var": "total"}, {"var": "discount"}]}` |
| `*` | `{"*": [{"var": "price"}, {"var": "quantity"}]}` |
| `/` | `{"/": [{"var": "sum"}, {"var": "count"}]}` |

## Conditional (if-then-else)

```json
{"if": [
  condition,
  value_if_true,
  value_if_false
]}
```

Example:
```json
{"if": [
  {">=": [{"var": "score"}, 700]},
  "approved",
  "denied"
]}
```

## Collection Operators

| Operator | Description |
|----------|-------------|
| `in` | Check if value is in array |
| `some` | True if any element matches |
| `all` | True if all elements match |
| `none` | True if no elements match |

---

# State Model (Computed Values)

The state model defines fields that are **automatically calculated** from other fields.

```json
"state_model": {
  "inputs": ["loan_amount", "annual_income"],
  "derived": {
    "debt_to_income_ratio": {
      "eval": {"/": [{"var": "loan_amount"}, {"var": "annual_income"}]}
    },
    "max_loan_eligible": {
      "eval": {"*": [{"var": "annual_income"}, 4]}
    }
  }
}
```

- `inputs`: Fields that trigger recalculation when changed
- `derived`: Computed fields with their formulas
- Derived fields are automatically marked `readonly: true`

---

# Attestations (Signatures)

**Important:** Tenet does NOT create signatures. It only validates that signatures have been collected.

## How Attestations Work

1. Your application collects a signature (via DocuSign, manual sign, etc.)
2. Your application updates the schema with `signed: true` and evidence
3. Tenet validates that required attestations have evidence

## Attestation Structure

```json
"attestations": {
  "compliance_officer_sign": {
    "law_ref": "OSHA 1910.132(d)",
    "statement": "I certify that all safety equipment has been inspected.",
    "required_role": "Safety_Officer",
    "provider": "DocuSign",
    "required": true,
    "signed": false,
    "evidence": null,
    "on_sign": {
      "set": {"inspection_complete": true}
    }
  }
}
```

## Attestation Fields

| Field | Required | Description |
|-------|----------|-------------|
| `statement` | Yes | What the signer is attesting to |
| `law_ref` | No | Legal citation requiring this signature |
| `required_role` | No | Who can sign (for your app to enforce) |
| `provider` | No | Signing method ("DocuSign", "Manual", etc.) |
| `required` | No | Must be signed for READY status |
| `signed` | — | Set by your app when signature collected |
| `evidence` | — | Proof of signature (set by your app) |
| `on_sign` | No | Actions to run after signing |

## Evidence Object

When a signature is collected, your app should populate:

```json
"evidence": {
  "provider_audit_id": "ds_abc123",
  "timestamp": "2026-01-20T14:30:00Z",
  "signer_id": "john.smith@company.com"
}
```

## Validation

For a required attestation, Tenet checks:

1. `signed` must be `true`
2. `evidence` must exist
3. `evidence.provider_audit_id` must not be empty

If any check fails, the document status is `INCOMPLETE`.

---

# Temporal Map (Date-Based Versioning)

When laws change, you can version your rules by date.

```json
"temporal_map": [
  {
    "valid_range": ["2025-01-01", "2025-12-31"],
    "logic_version": "v2025",
    "status": "ARCHIVED"
  },
  {
    "valid_range": ["2026-01-01", null],
    "logic_version": "v2026",
    "status": "ACTIVE"
  }
]
```

- `valid_range`: [start_date, end_date] — null means open-ended
- `logic_version`: Identifier for which rules apply
- `status`: "ACTIVE" or "ARCHIVED"

Rules reference their version:

```json
{
  "id": "tax_rate_2026",
  "logic_version": "v2026",
  "when": {">=": [{"var": "income"}, 100000]},
  "then": {"set": {"tax_rate": 0.35}}
}
```

---

# Complete Example: Loan Application

```json
{
  "$schema": "https://tenet.dev/schema/v1.json",
  "protocol": "Tenet_v1.0",
  "schema_id": "loan_application",
  "version": "2026.01",

  "definitions": {
    "applicant_name": {
      "type": "string",
      "value": null,
      "label": "Full Legal Name",
      "required": true
    },
    "annual_income": {
      "type": "number",
      "value": null,
      "label": "Annual Income ($)",
      "required": true,
      "min": 0
    },
    "loan_amount": {
      "type": "number",
      "value": null,
      "label": "Requested Loan Amount ($)",
      "required": true,
      "min": 1000,
      "max": 500000
    },
    "employment_status": {
      "type": "select",
      "value": null,
      "label": "Employment Status",
      "required": true,
      "options": ["employed", "self_employed", "retired", "unemployed"]
    },
    "debt_to_income": {
      "type": "number",
      "value": null,
      "label": "Debt-to-Income Ratio",
      "readonly": true
    },
    "approval_status": {
      "type": "select",
      "value": "pending",
      "options": ["pending", "approved", "denied", "review"],
      "readonly": true
    },
    "business_name": {
      "type": "string",
      "value": null,
      "label": "Business Name",
      "visible": false,
      "required": false
    }
  },

  "logic_tree": [
    {
      "id": "show_business_fields",
      "when": {"==": [{"var": "employment_status"}, "self_employed"]},
      "then": {
        "ui_modify": {
          "business_name": {"visible": true, "required": true}
        }
      }
    },
    {
      "id": "deny_unemployed",
      "law_ref": "Fair Lending Act § 12.3",
      "when": {"==": [{"var": "employment_status"}, "unemployed"]},
      "then": {
        "set": {"approval_status": "denied"},
        "error_msg": "Applicant must have income source per Fair Lending Act § 12.3"
      }
    },
    {
      "id": "auto_approve_low_dti",
      "when": {"and": [
        {"<": [{"var": "debt_to_income"}, 0.3]},
        {"!=": [{"var": "employment_status"}, "unemployed"]}
      ]},
      "then": {
        "set": {"approval_status": "approved"}
      }
    },
    {
      "id": "high_dti_review",
      "when": {">=": [{"var": "debt_to_income"}, 0.43]},
      "then": {
        "set": {"approval_status": "review"},
        "error_msg": "DTI exceeds 43% guideline, manual review required"
      }
    }
  ],

  "state_model": {
    "inputs": ["loan_amount", "annual_income"],
    "derived": {
      "debt_to_income": {
        "eval": {"/": [{"var": "loan_amount"}, {"var": "annual_income"}]}
      }
    }
  },

  "attestations": {
    "applicant_signature": {
      "statement": "I certify that all information provided is true and accurate.",
      "required": true,
      "signed": false,
      "evidence": null
    }
  }
}
```

---

# API Usage

## Go

```go
import "github.com/tenet-vm/tenet/pkg/tenet"

// Execute schema
result, err := tenet.Run(jsonString, time.Now())

// Verify a completed document against base schema
valid, err := tenet.Verify(completedDoc, baseSchema)
```

## JavaScript

```javascript
import { init, run, verify, lint } from '@tenet-vm/tenet';

// Initialize WASM (required for run/verify)
await init('./tenet.wasm');

// Execute schema
const result = run(schema, new Date());

// Verify transformation
const valid = verify(completedDoc, baseSchema);

// Lint (no init required - pure TypeScript)
const issues = lint(schema);
```

## CLI

```bash
# Run a schema
tenet run -file schema.json -date 2026-01-20

# Verify a completed document
tenet verify -new completed.json -base schema.json

# Static analysis
tenet lint -file schema.json
```

---

# Glossary

| Term | Definition |
|------|------------|
| **Schema** | A JSON document defining fields, rules, and validations |
| **Definition** | A single field with its type, value, and constraints |
| **Logic Tree** | Ordered list of rules to evaluate |
| **Rule** | A condition-action pair: "when X, then do Y" |
| **Attestation** | A signature requirement with evidence tracking |
| **Derived Field** | A computed value calculated from other fields |
| **Temporal Map** | Date-based routing for rule versioning |
| **READY** | Document status when all validations pass |
| **INCOMPLETE** | Document status when required items are missing |
| **INVALID** | Document status when constraints are violated |

---

# FAQ

## Does Tenet create digital signatures?
**No.** Tenet validates that signatures were collected by YOUR application. Your app integrates with DocuSign, Adobe Sign, or other providers, then updates the schema with `signed: true` and evidence.

## Does Tenet store data?
**No.** Tenet is a pure logic engine. It takes JSON in, processes it, and returns JSON out. Your application handles storage.

## Can Tenet run in a browser?
**Yes.** The JavaScript/WASM version runs entirely client-side.

## What happens if a rule sets the same field twice?
Tenet will warn you. Use the `lint` command to detect potential conflicts before running.

## How do I handle law changes?
Use the `temporal_map` to version your rules by effective date. Tenet automatically selects the correct rules based on the date you provide.
