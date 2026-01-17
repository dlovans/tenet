# API Reference

## Go

### Installation

```bash
go get github.com/yourusername/tenet/pkg/tenet
```

### Run

Execute schema logic for a given effective date.

```go
import (
    "time"
    "github.com/yourusername/tenet/pkg/tenet"
)

result, err := tenet.Run(jsonString, time.Now())
if err != nil {
    // Parse or execution error
}

// result is JSON string with computed state, errors, status
```

### Verify

Check that a transformation is legal by replaying the logic.

```go
valid, err := tenet.Verify(newJSON, oldJSON)
if err != nil {
    // Verification failed
}
// valid == true means transformation is legal
```

---

## CLI

### Installation

```bash
go build -o tenet ./cmd/tenet
```

### Run

```bash
# From file
./tenet run -date 2025-01-16 -file schema.json

# From stdin
cat schema.json | ./tenet run -date 2025-01-16
```

### Verify

```bash
./tenet verify -new updated.json -old original.json
```

---

## JavaScript / TypeScript

### Installation

```bash
npm install @tenet/core
```

### Browser Setup

```html
<script src="node_modules/@tenet/core/wasm/wasm_exec.js"></script>
<script type="module">
import { init, run, verify } from '@tenet/core';

await init('/path/to/tenet.wasm');
</script>
```

### Node.js Setup

```javascript
import { init, run, verify } from '@tenet/core';

await init('./node_modules/@tenet/core/wasm/tenet.wasm');
```

### Run

```typescript
import { run, TenetResult } from '@tenet/core';

const schema = {
  definitions: {
    amount: { type: 'number', value: 500 }
  }
};

const result: TenetResult = run(schema, new Date());

if (result.error) {
  console.error(result.error);
} else {
  console.log(result.result.status);     // "READY" | "INCOMPLETE" | "INVALID"
  console.log(result.result.errors);     // ValidationError[]
  console.log(result.result.definitions); // Updated definitions
}
```

### Verify

```typescript
import { verify, VerifyResult } from '@tenet/core';

const result: VerifyResult = verify(newSchema, oldSchema);

console.log(result.valid); // true or false
console.log(result.error); // Error message if failed
```

### Reactive UI Pattern

```typescript
function onFieldChange(fieldId: string, newValue: any) {
  // Update the schema
  schema.definitions[fieldId].value = newValue;
  
  // Re-run the VM
  const result = run(schema, new Date());
  
  if (!result.error) {
    updateUI(result.result);
  }
}

function updateUI(schema: TenetSchema) {
  for (const [id, def] of Object.entries(schema.definitions)) {
    const element = document.getElementById(id);
    
    // Show/hide based on visibility
    element.hidden = !def.visible;
    
    // Disable computed fields
    if (def.readonly) {
      element.setAttribute('disabled', 'true');
    }
    
    // Update constraints
    if (def.min !== undefined) element.min = def.min;
    if (def.max !== undefined) element.max = def.max;
  }
  
  // Display errors
  for (const error of schema.errors || []) {
    showError(error.field_id, error.message);
  }
}
```

---

## TypeScript Types

```typescript
interface TenetSchema {
  definitions: Record<string, Definition>;
  logic_tree?: Rule[];
  state_model?: StateModel;
  errors?: ValidationError[];
  status?: 'READY' | 'INCOMPLETE' | 'INVALID';
}

interface Definition {
  type: string;
  value?: any;
  required?: boolean;
  readonly?: boolean;  // true = computed
  visible?: boolean;
  min?: number;
  max?: number;
  // ... see full types in package
}

interface ValidationError {
  field_id?: string;
  rule_id?: string;
  message: string;
  law_ref?: string;
}
```
