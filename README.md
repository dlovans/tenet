# Tenet

**Law as Code** — A declarative logic VM for standardizing legal and regulatory requirements as executable JSON schemas.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Reactive Logic** — If-then rules that fire when conditions change
- **Temporal Routing** — Version logic based on effective dates
- **Derived State** — Computed values using JSON-logic expressions (marked `readonly`)
- **Constraint Validation** — Min/max bounds, string lengths, required fields
- **Attestation Enforcement** — Documents are INCOMPLETE until confirmed
- **Cycle Detection** — Runtime warning when multiple rules set the same field
- **Static Linter** — Catch undefined variables and potential issues before execution
- **Error Accumulation** — Collect all errors (non-blocking)
- **WASM Support** — Runs in browsers for reactive UIs

## Quick Start

```bash
# Install
go get github.com/yourusername/tenet

# Build CLI
go build -o tenet ./cmd/tenet

# Run a schema
./tenet run -date 2025-01-16 -file schema.json

# Lint a schema (static analysis)
./tenet lint -file schema.json
```

## Minimal Example

```json
{
  "definitions": {
    "amount": {"type": "number", "value": 500, "min": 0, "max": 10000}
  },
  "logic_tree": [
    {
      "id": "high_amount_warning",
      "when": {">": [{"var": "amount"}, 5000]},
      "then": {"error_msg": "Amount exceeds recommended limit."}
    }
  ]
}
```

## Documentation

- **[Full Documentation](docs/README.md)** — Schema reference, operators, API
- **[Contributing](CONTRIBUTING.md)** — Development setup, code style

## API

### Go

```go
import "github.com/yourusername/tenet"

result, err := tenet.Run(jsonString, time.Now())
valid, err := tenet.Verify(newJSON, oldJSON)
```

### JavaScript (WASM)

```javascript
const result = TenetRun(jsonString, "2025-01-16");
const verification = TenetVerify(newJSON, oldJSON);
```

## Document Status

| Status | Meaning |
|--------|---------|
| `READY` | All validations pass |
| `INCOMPLETE` | Missing required fields or attestations |
| `INVALID` | Type errors or constraint violations |

## License

MIT — see [LICENSE](LICENSE)
