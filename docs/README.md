# Tenet Documentation

Tenet is a declarative logic VM for JSON schemas. It handles temporal routing, reactive state computation, and validation.

## Table of Contents

- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Schema Reference](#schema-reference)
- [Operators](#operators)
- [API Reference](#api-reference)
- [JavaScript / TypeScript](#javascript--typescript)
- [Examples](#examples)

---

## Quick Start

### Installation

```bash
# Go
go get github.com/dlovans/tenet/pkg/tenet

# npm (browser + Node.js) — pure TypeScript, no WASM
npm install @dlovans/tenet-core

# Build CLI
go build -o tenet ./cmd/tenet
```

### Minimal Example

```json
{
  "definitions": {
    "amount": {
      "type": "number",
      "value": 500,
      "min": 0,
      "max": 10000
    }
  }
}
```

```bash
echo '{"definitions":{"amount":{"type":"number","value":500}}}' | ./tenet run -date 2025-01-01
```

---

## Core Concepts

### 1. Definitions
The data layer. Each definition is a typed field with value and constraints.

```json
{
  "definitions": {
    "loan_amount": {
      "type": "number",
      "value": 50000,
      "required": true,
      "min": 1000,
      "max": 500000
    }
  }
}
```

### 2. Logic Tree
Reactive if-then rules. When conditions are met, actions fire.

```json
{
  "logic_tree": [
    {
      "id": "rule_high_amount",
      "law_ref": "Lending Act §4.2",
      "when": {">": [{"var": "loan_amount"}, 100000]},
      "then": {
        "set": {"requires_review": true},
        "error_msg": "High-value loans require manual review."
      }
    }
  ]
}
```

### 3. Derived State
Computed values that update reactively.

```json
{
  "state_model": {
    "inputs": ["income", "loan_amount"],
    "derived": {
      "debt_ratio": {
        "eval": {"/": [{"var": "loan_amount"}, {"var": "income"}]}
      }
    }
  }
}
```

### 4. Temporal Routing
Version logic based on effective dates.

```json
{
  "temporal_map": [
    {
      "valid_range": ["2024-01-01", "2024-12-31"],
      "logic_version": "v1.0",
      "status": "ARCHIVED"
    },
    {
      "valid_range": ["2025-01-01", null],
      "logic_version": "v2.0",
      "status": "ACTIVE"
    }
  ]
}
```

---

## Schema Reference

### Root Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `definitions` | object | **Yes** | Field definitions |
| `logic_tree` | array | No | Reactive rules |
| `temporal_map` | array | No | Version routing |
| `state_model` | object | No | Derived values |
| `protocol` | string | No | Protocol identifier |
| `schema_id` | string | No | Schema identifier |
| `version` | string | No | Schema version |
| `valid_from` | string | No | Effective date (ISO 8601) |

### Definition Object

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `string`, `number`, `boolean`, `select`, `date`, `attestation`, `currency` |
| `value` | any | Current value (null = unset) |
| `required` | boolean | Is this field required? |
| `visible` | boolean | UI visibility (defaults to `true` when not specified) |
| `label` | string | Human-readable label |
| `options` | array | Options for `select` type |
| `min` | number | Minimum value (numbers) |
| `max` | number | Maximum value (numbers) |
| `step` | number | UI increment hint |
| `min_length` | integer | Minimum string length |
| `max_length` | integer | Maximum string length |
| `pattern` | string | Regex pattern for validation |

### Rule Object

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique rule identifier |
| `law_ref` | string | Legal citation |
| `when` | object | JSON-logic condition |
| `then` | object | Action to execute |
| `logic_version` | string | Temporal branch (optional) |

### Action Object

| Field | Type | Description |
|-------|------|-------------|
| `set` | object | Values to set in definitions |
| `ui_modify` | object | UI metadata changes |
| `error_msg` | string | Validation error message |

### ui_modify Fields

```json
{
  "ui_modify": {
    "field_name": {
      "visible": true,
      "required": true,
      "min": 0,
      "max": 1000000,
      "ui_class": "highlight",
      "ui_message": "This field is now required"
    }
  }
}
```

---

## Operators

### Variable Access
```json
{"var": "field_name"}
{"var": "nested.path.value"}
```

### Comparison
```json
{"==": [{"var": "status"}, "active"]}
{"!=": [{"var": "tier"}, "free"]}
{">": [{"var": "amount"}, 1000]}
{"<": [{"var": "age"}, 18]}
{">=": [{"var": "score"}, 700]}
{"<=": [{"var": "debt_ratio"}, 0.43]}
```

### Logic
```json
{"and": [condition1, condition2]}
{"or": [condition1, condition2]}
{"not": condition}
{"!": condition}
```

### Conditional
```json
{"if": [condition, then_value, else_value]}
{"if": [cond1, val1, cond2, val2, else_val]}
```

### Arithmetic
```json
{"+": [{"var": "a"}, {"var": "b"}]}
{"-": [{"var": "total"}, {"var": "discount"}]}
{"*": [{"var": "price"}, {"var": "quantity"}]}
{"/": [{"var": "amount"}, {"var": "months"}]}
```

### Date
```json
{"before": [{"var": "deadline"}, "2025-12-31"]}
{"after": [{"var": "start_date"}, {"var": "end_date"}]}
```

### Collection
```json
{"in": [{"var": "status"}, ["active", "pending"]]}
```

---

## API Reference

### Go

```go
import (
    "time"
    "github.com/dlovans/tenet/pkg/tenet"
)

// Run executes the schema logic
result, err := tenet.Run(jsonString, time.Now())

// Verify checks a transformation is legal (returns structured result)
vr := tenet.Verify(newJSON, baseJSON)
if vr.Error != "" {
    // Internal error (parse failure, panic recovery)
}
if !vr.Valid {
    for _, issue := range vr.Issues {
        fmt.Println(issue.Code, issue.FieldID, issue.Message)
    }
}
```

### CLI

```bash
# Run with effective date
./tenet run -date 2025-01-16 -file schema.json

# Run from stdin
cat schema.json | ./tenet run -date 2025-01-16

# Verify transformation
./tenet verify -new completed.json -base original.json

# Static analysis
./tenet lint -file schema.json
```

---

## JavaScript / TypeScript

The TypeScript package is a pure TypeScript implementation — no WASM, no `init()` required. The VM is ready immediately on import.

### Setup

```typescript
import { run, verify } from '@dlovans/tenet-core';

// Run schema
const result = run(schema, new Date());
if (result.error) {
  console.error(result.error);
} else {
  console.log(result.result.status);      // "READY" | "INCOMPLETE" | "INVALID"
  console.log(result.result.errors);      // ValidationError[] (with kind field)
  console.log(result.result.definitions); // Updated definitions
}

// Verify transformation (structured output)
const vr = verify(completedDoc, baseSchema);
if (!vr.valid) {
  for (const issue of vr.issues ?? []) {
    console.log(issue.code, issue.field_id, issue.message);
  }
}
```

### Reactive UI Pattern

```typescript
import { run, TenetSchema } from '@dlovans/tenet-core';

function onFieldChange(fieldId: string, newValue: any) {
  schema.definitions[fieldId].value = newValue;
  const result = run(schema, new Date());

  if (!result.error) {
    updateUI(result.result);
  }
}

function updateUI(schema: TenetSchema) {
  for (const [id, def] of Object.entries(schema.definitions)) {
    document.getElementById(id).hidden = def.visible === false;

    if (def.readonly) {
      document.getElementById(id).setAttribute('disabled', 'true');
    }
    if (def.min !== undefined) document.getElementById(id).min = def.min;
    if (def.max !== undefined) document.getElementById(id).max = def.max;
  }

  // Display errors (each error has a kind for programmatic handling)
  for (const error of schema.errors || []) {
    showError(error.field_id, error.message, error.kind);
  }

  setFormStatus(schema.status); // "READY", "INCOMPLETE", "INVALID"
}
```

---

## Examples

### Loan Application

```json
{
  "definitions": {
    "income": {"type": "number", "value": 75000, "required": true, "min": 0},
    "loan_amount": {"type": "number", "value": 250000, "required": true, "min": 1000},
    "employment": {"type": "select", "value": "employed", "options": ["employed", "self_employed", "unemployed"]},
    "approval": {"type": "select", "value": "pending", "options": ["pending", "approved", "denied"]}
  },
  "logic_tree": [
    {
      "id": "deny_unemployed",
      "law_ref": "Lending Act §4.2",
      "when": {"==": [{"var": "employment"}, "unemployed"]},
      "then": {
        "set": {"approval": "denied"},
        "error_msg": "Unemployed applicants are not eligible."
      }
    },
    {
      "id": "approve_good_ratio",
      "when": {
        "and": [
          {"!=": [{"var": "employment"}, "unemployed"]},
          {"<=": [{"var": "debt_ratio"}, 0.4]}
        ]
      },
      "then": {"set": {"approval": "approved"}}
    }
  ],
  "state_model": {
    "inputs": ["income", "loan_amount"],
    "derived": {
      "debt_ratio": {
        "eval": {"/": [{"var": "loan_amount"}, {"*": [{"var": "income"}, 5]}]}
      }
    }
  }
}
```

### GDPR Breach Report

```json
{
  "definitions": {
    "impact_level": {"type": "select", "value": "Material", "options": ["Low", "Material", "Severe"]},
    "reporting_deadline": {"type": "string"},
    "officer_attestation": {"type": "attestation", "label": "I confirm reasonable effort was made."}
  },
  "logic_tree": [
    {
      "id": "72_hour_rule",
      "law_ref": "GDPR Art. 33(1)",
      "when": {"==": [{"var": "impact_level"}, "Material"]},
      "then": {
        "set": {"reporting_deadline": "72h"},
        "ui_modify": {"officer_attestation": {"required": true, "visible": true}},
        "error_msg": "Material breaches require reporting within 72 hours."
      }
    }
  ]
}
```

---

## Document Status

The VM sets `status` on the output:

| Status | Meaning |
|--------|---------|
| `READY` | All validations pass, all required fields present |
| `INCOMPLETE` | Missing required fields or attestations |
| `INVALID` | Type errors or constraint violations |

---

## Error Format

Errors are accumulated (non-blocking) in the `errors` array. Each error includes a `kind` field for programmatic handling:

```json
{
  "errors": [
    {
      "field_id": "loan_amount",
      "rule_id": "max_limit_rule",
      "kind": "constraint_violation",
      "message": "Loan amount exceeds maximum of 500000",
      "law_ref": "Lending Act §12.3"
    }
  ],
  "status": "INVALID"
}
```

### Error Kinds

| Kind | Meaning |
|------|---------|
| `type_mismatch` | Value doesn't match the declared type |
| `missing_required` | Required field has no value |
| `constraint_violation` | Value violates min/max/pattern/length constraints |
| `attestation_incomplete` | Required attestation not signed or missing evidence |
| `runtime_warning` | Non-fatal issue (e.g., cycle detected during rule evaluation) |
| `cycle_detected` | Derived field dependency cycle detected |
```
