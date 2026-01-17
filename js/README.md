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

### `isReady(): boolean`
Check if WASM is initialized.

## TypeScript

Full type definitions are included. See `TenetSchema`, `Definition`, `Rule`, etc.

## License

MIT
