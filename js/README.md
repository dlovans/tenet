# @tenet/core

Declarative logic VM for JSON schemas. Reactive validation, temporal routing, and computed state.

## Installation

```bash
npm install @tenet/core
```

## Usage

### Browser

```html
<script src="https://unpkg.com/@tenet/core/wasm/wasm_exec.js"></script>
<script type="module">
import { init, run } from '@tenet/core';

await init('/path/to/tenet.wasm');

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
console.log(result);
</script>
```

### Node.js

```javascript
import { init, run, verify } from '@tenet/core';

// Initialize WASM
await init('./node_modules/@tenet/core/wasm/tenet.wasm');

// Run schema logic
const result = run(schema, new Date());

if (result.error) {
  console.error(result.error);
} else {
  console.log(result.result.status); // 'READY', 'INCOMPLETE', or 'INVALID'
  console.log(result.result.errors); // Validation errors
}

// Verify transformation
const verification = verify(newSchema, oldSchema);
console.log(verification.valid);
```

## API

### `init(wasmPath?: string): Promise<void>`
Initialize the WASM module. Must be called before `run()` or `verify()`.

### `run(schema, date?): TenetResult`
Execute the schema logic for the given effective date.

### `verify(newSchema, oldSchema): TenetVerifyResult`
Verify that a transformation is legal by replaying the logic.

### `lint(schema): LintResult` *(No WASM required)*
Static analysis - find issues without executing the schema.

```javascript
import { lint } from '@tenet/core';

const result = lint(schema);
// No init() needed - pure TypeScript!

if (!result.valid) {
  for (const issue of result.issues) {
    console.log(`${issue.severity}: ${issue.message}`);
  }
}
```

### `isTenetSchema(obj): boolean`
Check if an object is a Tenet schema.

### `isReady(): boolean`
Check if WASM is initialized.

## JSON Schema (IDE Support)

Add to your schema files for autocompletion:

```json
{
  "$schema": "https://tenet.dev/schema/v1.json",
  "definitions": { ... }
}
```

## TypeScript

Full type definitions are included. See `TenetSchema`, `Definition`, `Rule`, `LintResult`, etc.

## License

MIT
