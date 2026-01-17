# Schema Reference

This page documents the complete structure of a Tenet schema.

## Root Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `definitions` | object | **Yes** | Field definitions |
| `logic_tree` | array | No | Reactive rules |
| `state_model` | object | No | Derived (computed) values |
| `temporal_map` | array | No | Version routing |
| `protocol` | string | No | Protocol identifier |
| `schema_id` | string | No | Schema identifier |
| `version` | string | No | Schema version |
| `valid_from` | string | No | Effective date (ISO 8601) |

## Output Fields

These are added by `Run()`:

| Field | Type | Description |
|-------|------|-------------|
| `errors` | array | Accumulated validation errors |
| `status` | string | `READY`, `INCOMPLETE`, or `INVALID` |

---

## Definitions

Each definition is a typed field:

```json
{
  "definitions": {
    "loan_amount": {
      "type": "number",
      "value": 50000,
      "required": true,
      "min": 1000,
      "max": 500000,
      "step": 1000,
      "label": "Loan Amount ($)"
    }
  }
}
```

### Definition Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `string`, `number`, `boolean`, `select`, `date`, `attestation`, `currency` |
| `value` | any | Current value (`null` = unset) |
| `label` | string | Human-readable label |
| `required` | boolean | Is this field required? |
| `readonly` | boolean | `true` = computed, `false` = user-editable |
| `visible` | boolean | UI visibility hint |
| `options` | array | Options for `select` type |

### Numeric Constraints

| Field | Type | Description |
|-------|------|-------------|
| `min` | number | Minimum allowed value |
| `max` | number | Maximum allowed value |
| `step` | number | UI increment (e.g., `0.01` for currency) |

### String Constraints

| Field | Type | Description |
|-------|------|-------------|
| `min_length` | integer | Minimum string length |
| `max_length` | integer | Maximum string length |
| `pattern` | string | Regex pattern |

### UI Metadata

| Field | Type | Description |
|-------|------|-------------|
| `ui_class` | string | CSS class hint |
| `ui_message` | string | Inline message/hint |

---

## Logic Tree

Rules are evaluated in order:

```json
{
  "logic_tree": [
    {
      "id": "deny_unemployed",
      "law_ref": "Lending Act §4.2",
      "when": {"==": [{"var": "employment"}, "unemployed"]},
      "then": {
        "set": {"approval": "denied"},
        "ui_modify": {"approval": {"ui_class": "error"}},
        "error_msg": "Unemployed applicants are not eligible."
      }
    }
  ]
}
```

### Rule Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique rule identifier |
| `law_ref` | string | Legal citation (for audit trail) |
| `when` | object | JSON-logic condition |
| `then` | object | Action to execute |
| `logic_version` | string | Temporal branch (optional) |

### Action Fields

| Field | Type | Description |
|-------|------|-------------|
| `set` | object | Values to set in definitions |
| `ui_modify` | object | UI metadata changes |
| `error_msg` | string | Validation error message |

### ui_modify Options

```json
{
  "ui_modify": {
    "field_name": {
      "visible": true,
      "required": true,
      "min": 0,
      "max": 100000,
      "ui_class": "highlight",
      "ui_message": "This field is now required"
    }
  }
}
```

---

## State Model

Computed values that update reactively:

```json
{
  "state_model": {
    "inputs": ["income", "loan_amount"],
    "derived": {
      "debt_ratio": {
        "eval": {"/": [{"var": "loan_amount"}, {"var": "income"}]}
      },
      "max_loan": {
        "eval": {"*": [{"var": "income"}, 4]}
      }
    }
  }
}
```

Derived fields are added to `definitions` with `"readonly": true`.

---

## Temporal Map

Version logic based on effective dates:

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

Rules with a matching `logic_version` are enabled; others are disabled.

---

## Validation Errors

```json
{
  "errors": [
    {
      "field_id": "loan_amount",
      "rule_id": "max_limit_rule",
      "message": "Loan amount exceeds maximum of 500000",
      "law_ref": "Lending Act §12.3"
    }
  ]
}
```

---

## Document Status

| Status | Meaning |
|--------|---------|
| `READY` | All validations pass |
| `INCOMPLETE` | Missing required fields or attestations |
| `INVALID` | Type errors or constraint violations |
