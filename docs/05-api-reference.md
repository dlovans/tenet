# API Reference

## Go

### Installation

```bash
go get github.com/dlovans/tenet/pkg/tenet
```

### Run

Execute schema logic for a given effective date. Includes panic recovery — will never crash your process.

```go
import (
    "time"
    "github.com/dlovans/tenet/pkg/tenet"
)

result, err := tenet.Run(jsonString, time.Now())
if err != nil {
    // Parse error or internal panic (recovered safely)
}

// result is JSON string with computed state, errors, status
```

### Verify

Check that a completed document was correctly derived from a base schema. Returns a structured result with all issues found (not just the first).

```go
vr := tenet.Verify(completedJSON, baseSchemaJSON)

// Check for internal errors first
if vr.Error != "" {
    log.Fatal("Verification error:", vr.Error)
}

// Check validity
if vr.Valid {
    fmt.Println("Document verified")
    fmt.Println("Status:", vr.Status) // "READY", "INCOMPLETE", "INVALID"
} else {
    for _, issue := range vr.Issues {
        fmt.Printf("[%s] %s: %s\n", issue.Code, issue.FieldID, issue.Message)
        // issue.Expected and issue.Claimed available for computed_mismatch
    }
}

// vr.Schema contains the full re-run result for inspection
```

### Types

```go
// VerifyResult is the structured output of Verify()
type VerifyResult struct {
    Valid  bool          `json:"valid"`
    Status DocStatus     `json:"status,omitempty"`
    Issues []VerifyIssue `json:"issues,omitempty"`
    Schema *Schema       `json:"schema,omitempty"`
    Error  string        `json:"error,omitempty"`
}

type VerifyIssue struct {
    Code     VerifyIssueCode `json:"code"`
    FieldID  string          `json:"field_id,omitempty"`
    Message  string          `json:"message"`
    Expected any             `json:"expected,omitempty"`
    Claimed  any             `json:"claimed,omitempty"`
}

// VerifyIssueCode values:
// "unknown_field"           - Submitted field doesn't exist in schema
// "computed_mismatch"       - Readonly field value was tampered
// "attestation_unsigned"    - Required attestation not signed
// "attestation_no_evidence" - Signed but missing evidence
// "attestation_no_timestamp"- Evidence missing timestamp
// "status_mismatch"         - Claimed status doesn't match computed
// "convergence_failed"      - Document didn't converge in max iterations
// "internal_error"          - Unexpected error (parse failure, panic, etc.)
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
./tenet verify -new completed.json -base original.json
```

Output on success:
```
✓ Document verified: transformation is legal
```

Output on failure (structured issues):
```
✗ Document verification failed
  computed_mismatch [tax_bracket]: readonly field value mismatch: expected high, got low
  status_mismatch: claimed READY but computed INCOMPLETE
```

### Lint

```bash
./tenet lint -file schema.json
```

---

## JavaScript / TypeScript

### Installation

```bash
npm install @dlovans/tenet-core
```

The TypeScript package is a **pure TypeScript implementation** — no WASM, no native dependencies. The VM is ready immediately on import.

### Run

```typescript
import { run, TenetResult } from '@dlovans/tenet-core';

const schema = {
  definitions: {
    amount: { type: 'number', value: 500 }
  }
};

const result: TenetResult = run(schema, new Date());

if (result.error) {
  console.error(result.error);
} else {
  console.log(result.result.status);      // "READY" | "INCOMPLETE" | "INVALID"
  console.log(result.result.errors);      // ValidationError[] (with kind field)
  console.log(result.result.definitions); // Updated definitions
}
```

### Verify

```typescript
import { verify, TenetVerifyResult } from '@dlovans/tenet-core';

const vr: TenetVerifyResult = verify(completedDoc, baseSchema);

if (vr.error) {
  console.error('Internal error:', vr.error);
}

if (vr.valid) {
  console.log('Verified! Status:', vr.status);
} else {
  for (const issue of vr.issues ?? []) {
    console.log(`[${issue.code}] ${issue.field_id}: ${issue.message}`);
    // issue.expected and issue.claimed available for computed_mismatch
  }
}
```

### Reactive UI Pattern

```typescript
import { run, TenetSchema } from '@dlovans/tenet-core';

function onFieldChange(fieldId: string, newValue: any) {
  schema.definitions[fieldId].value = newValue;
  const result = run(schema, new Date());

  if (!result.error) {
    updateUI(result.result);
  }
}

function updateUI(schema: TenetSchema) {
  for (const [id, def] of Object.entries(schema.definitions)) {
    const element = document.getElementById(id);

    // visible defaults to true when not specified
    element.hidden = def.visible === false;

    if (def.readonly) {
      element.setAttribute('disabled', 'true');
    }
    if (def.min !== undefined) element.min = def.min;
    if (def.max !== undefined) element.max = def.max;
  }

  // Display errors (each has a kind for programmatic handling)
  for (const error of schema.errors || []) {
    showError(error.field_id, error.message, error.kind);
  }
}
```

### Backwards Compatibility

The `init()` function still exists as a no-op for backwards compatibility but is deprecated:

```typescript
// No longer needed — safe to remove from your code
import { init } from '@dlovans/tenet-core';
await init(); // no-op, returns immediately
```

---

## TypeScript Types

```typescript
interface TenetSchema {
  definitions: Record<string, Definition>;
  logic_tree?: Rule[];
  state_model?: StateModel;
  attestations?: Record<string, Attestation>;
  errors?: ValidationError[];
  status?: 'READY' | 'INCOMPLETE' | 'INVALID';
}

interface Definition {
  type: string;
  value?: any;
  required?: boolean;
  readonly?: boolean;   // true = computed
  visible?: boolean;    // defaults to true when not specified
  min?: number;
  max?: number;
  pattern?: string;     // regex validation
  // ... see full types in package
}

type ErrorKind =
  | 'type_mismatch'
  | 'missing_required'
  | 'constraint_violation'
  | 'attestation_incomplete'
  | 'runtime_warning'
  | 'cycle_detected';

interface ValidationError {
  field_id?: string;
  rule_id?: string;
  kind: ErrorKind;
  message: string;
  law_ref?: string;
}

type VerifyIssueCode =
  | 'unknown_field'
  | 'computed_mismatch'
  | 'attestation_unsigned'
  | 'attestation_no_evidence'
  | 'attestation_no_timestamp'
  | 'status_mismatch'
  | 'convergence_failed'
  | 'internal_error';

interface VerifyIssue {
  code: VerifyIssueCode;
  field_id?: string;
  message: string;
  expected?: unknown;
  claimed?: unknown;
}

interface TenetVerifyResult {
  valid: boolean;
  status?: string;
  issues?: VerifyIssue[];
  schema?: TenetSchema;
  error?: string;
}
```
