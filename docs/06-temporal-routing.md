# Temporal Routing

Tenet supports versioning logic based on effective dates. This is essential for legal/compliance use cases where rules change over time.

## Overview

When regulations change:
1. Old documents should be validated against old rules
2. New documents should use new rules
3. Everything must be auditable

Tenet solves this with the `temporal_map` and `logic_version` fields.

---

## Temporal Map

Define which logic versions are active for which date ranges:

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

- `valid_range`: `[start, end]` — ISO 8601 dates, `null` end means open-ended
- `logic_version`: Identifier that rules reference
- `status`: `ACTIVE` or `ARCHIVED` (informational)

---

## Versioned Rules

Tag rules with their `logic_version`:

```json
{
  "logic_tree": [
    {
      "id": "old_threshold",
      "logic_version": "v1.0",
      "when": {">": [{"var": "amount"}, 50000]},
      "then": {"error_msg": "Amount exceeds old limit"}
    },
    {
      "id": "new_threshold",
      "logic_version": "v2.0",
      "when": {">": [{"var": "amount"}, 100000]},
      "then": {"error_msg": "Amount exceeds new limit"}
    }
  ]
}
```

---

## How It Works

When you call `Run(schema, date)`:

1. **Select branch**: Find the `temporal_map` entry where `date` falls within `valid_range`
2. **Prune rules**: Disable rules whose `logic_version` doesn't match
3. **Evaluate**: Only enabled rules fire

```
Run(schema, "2024-06-15")
  → active branch = v1.0
  → "old_threshold" enabled, "new_threshold" disabled
  → Error if amount > 50000

Run(schema, "2025-03-01")
  → active branch = v2.0
  → "old_threshold" disabled, "new_threshold" enabled
  → Error if amount > 100000
```

---

## Rules Without Version

Rules without a `logic_version` are **always enabled**:

```json
{
  "id": "always_required",
  "when": {"==": [{"var": "required_field"}, null]},
  "then": {"error_msg": "This field is always required"}
}
```

---

## Audit Trail

The `Verify()` function replays logic to prove transformations are legal:

```go
// Prove that newJSON was derived from oldJSON
valid, err := tenet.Verify(newJSON, oldJSON)
```

This re-runs `oldJSON` through the VM and compares the result with `newJSON`. If values match, the transformation is valid.

---

## Example: Regulatory Change

GDPR breach reporting deadline changed from 72 hours to 48 hours:

```json
{
  "temporal_map": [
    {"valid_range": ["2018-05-25", "2024-12-31"], "logic_version": "gdpr_v1", "status": "ARCHIVED"},
    {"valid_range": ["2025-01-01", null], "logic_version": "gdpr_v2", "status": "ACTIVE"}
  ],
  "logic_tree": [
    {
      "id": "72h_deadline",
      "logic_version": "gdpr_v1",
      "law_ref": "GDPR Art. 33(1) [2018]",
      "when": {"==": [{"var": "breach_type"}, "material"]},
      "then": {"set": {"reporting_deadline": "72h"}}
    },
    {
      "id": "48h_deadline",
      "logic_version": "gdpr_v2",
      "law_ref": "GDPR Art. 33(1) [2025 Amendment]",
      "when": {"==": [{"var": "breach_type"}, "material"]},
      "then": {"set": {"reporting_deadline": "48h"}}
    }
  ]
}
```

Running this schema with `date = 2024-06-01` applies the 72-hour rule.
Running with `date = 2025-06-01` applies the 48-hour rule.
