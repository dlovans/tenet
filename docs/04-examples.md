# Examples

Real-world examples demonstrating Tenet's capabilities.

## Loan Application

A loan approval system with credit checks, employment validation, and debt ratio calculation.

```json
{
  "definitions": {
    "income": {"type": "number", "value": 75000, "required": true, "min": 0},
    "loan_amount": {"type": "number", "value": 250000, "required": true, "min": 1000},
    "employment": {"type": "select", "value": "employed", "options": ["employed", "self_employed", "unemployed"]},
    "credit_score": {"type": "number", "value": 720, "min": 300, "max": 850},
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
      "id": "approve_good_credit",
      "when": {
        "and": [
          {">=": [{"var": "credit_score"}, 700]},
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

**Key features:**
- `debt_ratio` is computed and marked `readonly: true`
- Rules reference the derived `debt_ratio`
- Law references enable audit trails

---

## E-commerce Checkout

Dynamic pricing with discount codes and free shipping thresholds.

```json
{
  "definitions": {
    "cart_total": {"type": "currency", "value": 150.00, "min": 0},
    "discount_code": {"type": "string", "value": "SAVE20"},
    "shipping_method": {"type": "select", "value": "standard", "options": ["standard", "express"]},
    "shipping_cost": {"type": "currency", "value": 0},
    "discount_amount": {"type": "currency", "value": 0}
  },
  "logic_tree": [
    {
      "id": "free_shipping",
      "when": {">=": [{"var": "cart_total"}, 100]},
      "then": {
        "set": {"shipping_cost": 0},
        "ui_modify": {"shipping_method": {"ui_message": "Free shipping on orders over $100!"}}
      }
    },
    {
      "id": "save20_discount",
      "when": {"==": [{"var": "discount_code"}, "SAVE20"]},
      "then": {"set": {"discount_amount": 20}}
    }
  ],
  "state_model": {
    "derived": {
      "final_total": {
        "eval": {"-": [{"+": [{"var": "cart_total"}, {"var": "shipping_cost"}]}, {"var": "discount_amount"}]}
      }
    }
  }
}
```

**Key features:**
- `final_total` computed from other fields
- UI messages for user feedback
- Discount validation via rules

---

## Conditional Survey

Questions that appear based on previous answers.

```json
{
  "definitions": {
    "has_children": {"type": "boolean", "value": false, "label": "Do you have children?"},
    "children_count": {"type": "number", "min": 0, "visible": false, "label": "How many?"},
    "employment": {"type": "select", "options": ["employed", "student", "retired"]},
    "company_size": {"type": "select", "options": ["1-10", "11-50", "50+"], "visible": false}
  },
  "logic_tree": [
    {
      "id": "show_children_count",
      "when": {"==": [{"var": "has_children"}, true]},
      "then": {
        "ui_modify": {"children_count": {"visible": true, "required": true}}
      }
    },
    {
      "id": "show_company_size",
      "when": {"==": [{"var": "employment"}, "employed"]},
      "then": {
        "ui_modify": {"company_size": {"visible": true, "required": true}}
      }
    }
  ]
}
```

**Key features:**
- Fields hidden by default (`visible: false`)
- Rules dynamically show and require fields
- No derived state needed — pure UI logic

---

## Game Character Builder

RPG character with class-based abilities and stat constraints.

```json
{
  "definitions": {
    "class": {"type": "select", "value": "warrior", "options": ["warrior", "mage", "rogue"]},
    "strength": {"type": "number", "value": 10, "min": 1, "max": 20},
    "intelligence": {"type": "number", "value": 10, "min": 1, "max": 20},
    "can_cast_spells": {"type": "boolean", "value": false},
    "can_use_heavy_armor": {"type": "boolean", "value": false}
  },
  "logic_tree": [
    {
      "id": "warrior_abilities",
      "when": {"==": [{"var": "class"}, "warrior"]},
      "then": {
        "set": {"can_use_heavy_armor": true, "can_cast_spells": false},
        "ui_modify": {
          "strength": {"ui_message": "Primary stat", "ui_class": "highlight"},
          "intelligence": {"max": 15}
        }
      }
    },
    {
      "id": "mage_abilities",
      "when": {"==": [{"var": "class"}, "mage"]},
      "then": {
        "set": {"can_cast_spells": true, "can_use_heavy_armor": false},
        "ui_modify": {
          "intelligence": {"ui_message": "Primary stat"},
          "strength": {"max": 12}
        }
      }
    }
  ],
  "state_model": {
    "derived": {
      "health": {"eval": {"+": [100, {"*": [{"var": "strength"}, 10]}]}},
      "mana": {
        "eval": {"if": [{"var": "can_cast_spells"}, {"+": [50, {"*": [{"var": "intelligence"}, 15]}]}, 0]}
      }
    }
  }
}
```

**Key features:**
- Class selection changes stat constraints
- Derived `health` and `mana` computed from stats
- `ui_class` for styling hints

---

## Edge Case Examples

The `examples/` directory contains additional schemas testing edge cases:

| File | What it tests |
|------|---------------|
| `temporal_tax_reform.json` | Temporal routing with 3 versioned rule sets |
| `competing_rules.json` | Two rules setting the same field (cycle detection) |
| `chained_derived.json` | 7-level chained derived computation |
| `visibility_edge_cases.json` | `visible: false` on load, conditional visibility |
| `null_arithmetic.json` | Null propagation, division by zero |
| `collection_operators.json` | `some`/`all`/`none`/`in` operators with arrays |
| `dynamic_constraints.json` | `ui_modify` changing min/max dynamically |
| `falsy_values.json` | Preservation of `false`, `0`, `""` values |
| `pattern_validation.json` | Regex patterns, length constraints |

Run any example through the CLI:

```bash
./tenet run -date 2025-06-15 -file examples/temporal_tax_reform.json
```
