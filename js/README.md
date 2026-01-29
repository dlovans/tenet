# @dlovans/tenet-core

Declarative logic VM for JSON schemas. Reactive validation, temporal routing, and computed state.

**Pure TypeScript** — No WASM, no native dependencies. Works in browsers, Node.js, Deno, Bun.

## Installation

```bash
npm install @dlovans/tenet-core
```

## Usage

### Browser

```html
<script type="module">
import { run } from '@dlovans/tenet-core';

const schema = {
  definitions: {
    amount: { type: 'number', value: 150, min: 0, max: 10000 }
  },
  logic_tree: [
    {
      id: 'high_amount',
      when: { '>': [{ var: 'amount' }, 1000] },
      then: { error_msg: 'Amount exceeds limit.' }
    }
  ]
};

const result = run(schema);
console.log(result.result.status); // 'READY'
</script>
```

### Node.js

```javascript
import { run, verify } from '@dlovans/tenet-core';

// Run schema logic - no initialization needed
const result = run(schema, new Date());

if (result.error) {
  console.error(result.error);
} else {
  console.log(result.result.status); // 'READY', 'INCOMPLETE', or 'INVALID'
  console.log(result.result.errors); // Validation errors (if any)
}

// Verify transformation
const verification = verify(newSchema, oldSchema);
console.log(verification.valid);
```

## API

### `run(schema, date?): TenetResult`
Execute the schema logic for the given effective date.

- `schema` — TenetSchema object or JSON string
- `date` — Effective date for temporal routing (default: now)

Returns `{ result: TenetSchema }` or `{ error: string }`.

### `verify(newSchema, oldSchema): TenetVerifyResult`
Verify that a transformation is legal by replaying the logic.

Returns `{ valid: boolean, error?: string }`.

### `isReady(): boolean`
Always returns `true`. Kept for backwards compatibility.

### `init(): Promise<void>` *(deprecated)*
No-op. Kept for backwards compatibility with v0.1.x.

## Runtime Validation

The VM automatically detects and reports:

- **Undefined variables** — `{"var": "unknown_field"}`
- **Unknown operators** — `{"invalid_op": [...]}`
- **Temporal conflicts** — Overlapping date ranges, same start/end dates

All errors are returned in `result.errors` without failing execution.

## TypeScript

Full type definitions included:

```typescript
import type { TenetSchema, TenetResult, Definition, Rule } from '@dlovans/tenet-core';
```

## Migration from v0.1.x

```javascript
// Before (v0.1.x with WASM)
import { init, run, lint } from '@dlovans/tenet-core';
await init('./tenet.wasm');
const result = run(schema);
const issues = lint(schema);

// After (v0.2.x pure TypeScript)
import { run } from '@dlovans/tenet-core';
const result = run(schema);
// Validation errors are now in result.result.errors
```

## License

MIT
