# Tenet

A declarative logic VM for JSON. Define rules, compute values, validate constraints — all in pure JSON.

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## What It Does

Tenet processes JSON documents through a rules engine:

```
Your JSON Schema → [Tenet VM] → Validated JSON with computed values
```

Use it for **smart forms**, **compliance checks**, **game logic**, **workflow automation**, or anything that needs deterministic rule evaluation.

## Features

- **Reactive Rules** — If-then logic that fires when conditions match
- **Computed Fields** — Derived values calculated automatically
- **Constraint Validation** — Min/max, required fields, patterns
- **Dynamic Visibility** — Show/hide fields based on state
- **Temporal Routing** — Version rules by effective date
- **Attestation Tracking** — Validate that signatures were collected
- **Static Linter** — Catch errors before execution
- **WASM Support** — Runs in browsers

## Quick Start

```bash
# Build CLI
go build -o tenet ./cmd/tenet

# Run a schema
./tenet run -file schema.json

# Lint (static analysis)
./tenet lint -file schema.json
```

## Example

```json
{
  "definitions": {
    "income": {"type": "number", "value": 45000, "required": true},
    "tax_bracket": {"type": "string", "readonly": true}
  },
  "logic_tree": [
    {
      "id": "low_income",
      "when": {"<": [{"var": "income"}, 50000]},
      "then": {"set": {"tax_bracket": "low"}}
    },
    {
      "id": "high_income", 
      "when": {">=": [{"var": "income"}, 50000]},
      "then": {"set": {"tax_bracket": "high"}}
    }
  ]
}
```

Output: `tax_bracket` is computed as `"low"`.

## Documentation

- **[Complete Specification](SPECIFICATION.md)** — Full schema reference
- **[API Reference](docs/05-api-reference.md)** — Go, JavaScript, CLI
- **[Examples](docs/04-examples.md)** — Real-world schemas

## API

### Go

```go
import "github.com/dlovans/tenet/pkg/tenet"

result, err := tenet.Run(jsonString, time.Now())
valid, err := tenet.Verify(completedDoc, baseSchema)
```

### JavaScript

```javascript
import { init, run, verify, lint } from '@dlovans/tenet-core';

await init('./tenet.wasm');
const result = run(schema);

// Lint works without WASM
const issues = lint(schema);
```

### CLI

```bash
tenet run -file schema.json -date 2026-01-20
tenet verify -new completed.json -base schema.json
tenet lint -file schema.json
```

## Document Status

| Status | Meaning |
|--------|---------|
| `READY` | All validations pass |
| `INCOMPLETE` | Missing required fields or attestations |
| `INVALID` | Constraint violations |

## License

MIT — see [LICENSE](LICENSE)
