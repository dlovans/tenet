# Introduction

Tenet is a declarative logic VM for JSON schemas. It evaluates reactive rules, computes derived values, and validates documents — all from a single JSON file.

## What is Tenet?

Tenet is a **"Law as Code"** runtime. It standardizes legal and regulatory logic into machine-readable JSON that can be executed, validated, and audited.

Instead of embedding compliance rules in application code, you express them declaratively:

```json
{
  "definitions": {
    "breach_type": {"type": "select", "value": "material", "options": ["minor", "material", "severe"]},
    "reporting_deadline": {"type": "string"}
  },
  "logic_tree": [
    {
      "id": "gdpr_72h_rule",
      "law_ref": "GDPR Art. 33(1)",
      "when": {"==": [{"var": "breach_type"}, "material"]},
      "then": {
        "set": {"reporting_deadline": "72h"},
        "error_msg": "Material breaches must be reported within 72 hours."
      }
    }
  ]
}
```

When you run this through Tenet, it:
1. Evaluates all rules in the `logic_tree`
2. Computes any `derived` values in the `state_model`
3. Validates all `definitions` against their constraints
4. Returns the transformed JSON with `errors`, `status`, and audit trail

## Primary Use Case: Law as Code

Tenet was built to **standardize legal requirements in the digital world**:

- **Regulatory Compliance** — Encode laws (GDPR, NIS2, lending regulations) as executable schemas
- **Legal Audit Trails** — Every rule references its `law_ref` for traceability
- **Temporal Versioning** — Laws change; Tenet handles version routing by effective date
- **Verification** — Prove that document transformations are legal

The goal: machine-readable regulations that developers can fetch via API and embed in their applications.

## Other Applications

Because Tenet is a general-purpose declarative logic engine, developers have found many uses:

- **Dynamic Forms** — Conditional fields, computed values, reactive validation
- **E-commerce** — Pricing rules, discount codes, eligibility checks
- **Configuration Wizards** — Dependency-aware software setup
- **Game Systems** — Character builders, skill trees with constraints

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Definitions** | Typed fields with values and constraints |
| **Logic Tree** | Rules that fire when conditions are met |
| **Derived State** | Computed values (marked `readonly`) |
| **Temporal Routing** | Version logic based on effective dates |

## Installation

```bash
# Go
go get github.com/yourusername/tenet/pkg/tenet

# npm (browser + Node.js)
npm install @tenet/core
```

## Next Steps

- [Schema Reference](02-schema-reference.md) — Full definition syntax
- [Operators](03-operators.md) — JSON-logic operators
- [Examples](04-examples.md) — Real-world use cases
