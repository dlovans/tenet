# Operators

Tenet uses JSON-logic syntax for expressions. All operators are nil-safe.

## Variable Access

```json
{"var": "field_name"}
{"var": "nested.path.value"}
```

Returns `null` if the path doesn't exist.

---

## Comparison

| Operator | Example | Description |
|----------|---------|-------------|
| `==` | `{"==": [{"var": "status"}, "active"]}` | Equal |
| `!=` | `{"!=": [{"var": "tier"}, "free"]}` | Not equal |
| `>` | `{">": [{"var": "amount"}, 1000]}` | Greater than |
| `<` | `{"<": [{"var": "age"}, 18]}` | Less than |
| `>=` | `{">=": [{"var": "score"}, 700]}` | Greater or equal |
| `<=` | `{"<=": [{"var": "ratio"}, 0.43]}` | Less or equal |

**Nil behavior:** Comparisons with `null` return `false`.

---

## Logic

| Operator | Example | Description |
|----------|---------|-------------|
| `and` | `{"and": [cond1, cond2]}` | All conditions true |
| `or` | `{"or": [cond1, cond2]}` | Any condition true |
| `not` | `{"not": condition}` | Negate |
| `!` | `{"!": condition}` | Alias for `not` |

```json
{
  "and": [
    {">=": [{"var": "credit_score"}, 700]},
    {"in": [{"var": "employment"}, ["employed", "self_employed"]]}
  ]
}
```

---

## Conditional

```json
{"if": [condition, then_value, else_value]}
```

Multi-branch:

```json
{
  "if": [
    {">=": [{"var": "score"}, 800]}, "excellent",
    {">=": [{"var": "score"}, 700]}, "good",
    {">=": [{"var": "score"}, 600]}, "fair",
    "poor"
  ]
}
```

---

## Arithmetic

| Operator | Example | Description |
|----------|---------|-------------|
| `+` | `{"+": [{"var": "a"}, {"var": "b"}]}` | Add |
| `-` | `{"-": [{"var": "total"}, {"var": "discount"}]}` | Subtract |
| `*` | `{"*": [{"var": "price"}, {"var": "qty"}]}` | Multiply |
| `/` | `{"/": [{"var": "amount"}, 12]}` | Divide |

**Nil behavior:** Operations with `null` return `null`.

---

## Date

| Operator | Example | Description |
|----------|---------|-------------|
| `before` | `{"before": [{"var": "start"}, {"var": "end"}]}` | Date A before Date B |
| `after` | `{"after": [{"var": "deadline"}, "2025-12-31"]}` | Date A after Date B |

Dates can be ISO 8601 strings (`"2025-01-16"`) or variables.

---

## Collection

| Operator | Example | Description |
|----------|---------|-------------|
| `in` | `{"in": [{"var": "status"}, ["active", "pending"]]}` | Value in array |
| `in` | `{"in": ["sub", {"var": "description"}]}` | Substring in string |
| `some` | `{"some": [{"var": "scores"}, {">": [{"var": ""}, 90]}]}` | Any element matches |
| `all` | `{"all": [{"var": "scores"}, {">=": [{"var": ""}, 60]}]}` | Every element matches |
| `none` | `{"none": [{"var": "flags"}, {"==": [{"var": ""}, "blocked"]}]}` | No element matches |

### Collection Operator Details

The `some`, `all`, and `none` operators iterate over an array-valued definition. Inside the condition, `{"var": ""}` refers to the current element.

```json
{
  "definitions": {
    "scores": {"type": "number", "value": [85, 92, 78, 95]}
  },
  "logic_tree": [
    {
      "id": "has_excellent",
      "when": {"some": [{"var": "scores"}, {">=": [{"var": ""}, 90]}]},
      "then": {"set": {"has_excellent_score": true}}
    },
    {
      "id": "all_passing",
      "when": {"all": [{"var": "scores"}, {">=": [{"var": ""}, 60]}]},
      "then": {"set": {"all_passed": true}}
    }
  ]
}
```

> **Note:** When a definition holds an array value, the declared `type` describes the element type. For example, `"type": "number"` with `"value": [85, 92, 78]` means an array of numbers.

---

## Complete Example

```json
{
  "when": {
    "and": [
      {">=": [{"var": "credit_score"}, 700]},
      {"in": [{"var": "employment"}, ["employed", "self_employed"]]},
      {"<=": [{"var": "debt_ratio"}, 0.43]}
    ]
  },
  "then": {
    "set": {"approval": "approved", "risk": "low"}
  }
}
```

This rule approves applicants with:
- Credit score ≥ 700
- Employment status is "employed" or "self_employed"
- Debt ratio ≤ 43%
