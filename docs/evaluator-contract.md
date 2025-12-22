# Evaluator Contract

## Overview

Evaluators are **external, sandboxed processes** that receive execution context and return decisions. They contain **ALL business logic** - the Go executor is completely dumb.

## Input: Evaluation Context

### Schema

```json
{
  "ir": { ... },              // Original IR that generated this request
  "request": {
    "method": "GET",
    "url": "https://api.example.com/users",
    "headers": { ... },
    "body": { ... }
  },
  "response": {
    "status": 200,
    "headers": { ... },
    "body": { ... },           // Parsed as JSON if possible, else string
    "latency_ms": 145.23,
    "size_bytes": 1024,
    "error": "..."             // Present if request failed
  },
  "vars": {
    "attempt": 1,              // Current retry attempt
    "env": "prod",             // Environment
    ...                        // Custom variables
  }
}
```

### Delivery Method

**Stdin**: Evaluator reads JSON from standard input

```javascript
const input = await Bun.stdin.text();
const context = JSON.parse(input);
```

```python
import sys
context = json.loads(sys.stdin.read())
```

## Output: Evaluator Decision

### Schema

```json
{
  "decision": "pass | retry | fail | branch",
  "reason": "Human-readable explanation",
  "mutations": {
    "headers": { "X-Retry": "1" },
    "query": { "page": "2" },
    "body": { ... },
    "vars": { "token": "abc123" }
  },
  "actions": {
    "retry_after_ms": 1000,
    "max_retries": 3,
    "goto": "fallback_flow",
    "extract": {
      "user_id": { "jsonpath": "$.user.id" }
    }
  },
  "metadata": {
    "performance_warning": true
  }
}
```

### Delivery Method

**Stdout**: Evaluator writes JSON to standard output

```javascript
console.log(JSON.stringify(decision));
```

```python
print(json.dumps(decision))
```

## Decision Types

### 1. pass

**Meaning**: Request succeeded, continue workflow

**Required fields**: `decision`, `reason`

**Example**:
```json
{
  "decision": "pass",
  "reason": "HTTP 200 OK"
}
```

**Orchestrator action**: Marks as success, returns result

### 2. retry

**Meaning**: Request should be retried (transient failure)

**Required fields**: `decision`, `reason`

**Optional actions**: `retry_after_ms`, `max_retries`

**Example**:
```json
{
  "decision": "retry",
  "reason": "Rate limited",
  "actions": {
    "retry_after_ms": 2000,
    "max_retries": 5
  }
}
```

**Orchestrator action**:
1. Waits `retry_after_ms` milliseconds
2. Applies mutations (if any)
3. Re-executes request
4. Increments `vars.attempt`

### 3. fail

**Meaning**: Request failed permanently (non-retryable)

**Required fields**: `decision`, `reason`

**Example**:
```json
{
  "decision": "fail",
  "reason": "HTTP 404 Not Found"
}
```

**Orchestrator action**: Marks as failure, returns error

### 4. branch

**Meaning**: Jump to different workflow step

**Required fields**: `decision`, `reason`, `actions.goto`

**Example**:
```json
{
  "decision": "branch",
  "reason": "Unauthorized - need to refresh token",
  "actions": {
    "goto": "refresh_auth"
  }
}
```

**Orchestrator action**: Executes named workflow step (future feature)

## Mutations

Mutations modify the IR for the next retry/request.

### Headers

Add or update request headers:

```json
{
  "mutations": {
    "headers": {
      "X-Retry-Count": "2",
      "Authorization": "Bearer new_token"
    }
  }
}
```

### Query Parameters

Add or update query params:

```json
{
  "mutations": {
    "query": {
      "page": "2",
      "offset": "100"
    }
  }
}
```

### Body

Replace request body:

```json
{
  "mutations": {
    "body": {
      "user": "admin",
      "action": "retry"
    }
  }
}
```

### Variables

Update context variables (passed to next evaluator):

```json
{
  "mutations": {
    "vars": {
      "auth_token": "abc123",
      "user_id": "456"
    }
  }
}
```

## Actions

Actions control orchestration behavior.

### retry_after_ms

Delay before retry (in milliseconds):

```json
{
  "actions": {
    "retry_after_ms": 1000  // Wait 1 second
  }
}
```

**Common patterns**:
- Fixed delay: `1000`
- Exponential backoff: `Math.pow(2, attempt) * 1000`
- Server-specified: `parseInt(headers['retry-after']) * 1000`

### max_retries

Override default max retry count:

```json
{
  "actions": {
    "max_retries": 5  // Override default (usually 3)
  }
}
```

### goto

Branch target for workflow:

```json
{
  "actions": {
    "goto": "refresh_auth_flow"
  }
}
```

*Note: Branching not yet implemented, reserved for future*

### extract

Data extraction rules (future feature):

```json
{
  "actions": {
    "extract": {
      "user_id": { "jsonpath": "$.user.id" },
      "session": { "regex": "session=([^;]+)" }
    }
  }
}
```

## Metadata

Arbitrary data for logging/debugging:

```json
{
  "metadata": {
    "slow_response": true,
    "large_payload": true,
    "performance_percentile": 95
  }
}
```

## Example Evaluators

### Simple Success/Fail

```javascript
const decision = {
  decision: response.status >= 200 && response.status < 300 ? "pass" : "fail",
  reason: `HTTP ${response.status}`
};
```

### Smart Retry Logic

```javascript
const decision = { decision: "pass" };

if (response.status === 429) {
  const attempt = vars.attempt || 1;
  if (attempt < 5) {
    decision.decision = "retry";
    decision.reason = "Rate limited";
    decision.actions = {
      retry_after_ms: Math.pow(2, attempt) * 1000
    };
  } else {
    decision.decision = "fail";
    decision.reason = "Rate limited after 5 attempts";
  }
}
```

### Data Extraction

```javascript
const decision = { decision: "pass" };

if (response.body && response.body.token) {
  decision.mutations = {
    vars: { auth_token: response.body.token },
    headers: { "Authorization": `Bearer ${response.body.token}` }
  };
}
```

### Performance Monitoring

```javascript
const decision = { decision: "pass" };

if (response.latency_ms > 1000) {
  decision.metadata = {
    performance_warning: true,
    latency_bucket: "p95"
  };
}
```

## Runtime Requirements

### Timeouts

Evaluators **must** complete within timeout (default 5s):

```json
{
  "evaluation": {
    "timeout_ms": 5000
  }
}
```

If timeout exceeded:
- Evaluator process is killed
- Orchestrator falls back to default evaluator

### Resource Limits

Evaluators should be:
- **Stateless**: No persistent state between calls
- **Deterministic**: Same input → same output
- **Fast**: Complete in <100ms typically
- **Safe**: No side effects (network calls, file writes)

### Error Handling

If evaluator fails (crashes, invalid JSON):
- Orchestrator catches error
- Falls back to default evaluator (simple pass/fail based on status code)

**Good practice**: Always output valid JSON, even on error:

```javascript
try {
  // ... evaluation logic
} catch (error) {
  console.log(JSON.stringify({
    decision: "fail",
    reason: `Evaluator error: ${error.message}`
  }));
}
```

## Language Examples

### Bun (JavaScript)

```javascript
#!/usr/bin/env bun

async function main() {
  const input = await Bun.stdin.text();
  const context = JSON.parse(input);

  const decision = {
    decision: context.response.status < 400 ? "pass" : "fail",
    reason: `HTTP ${context.response.status}`
  };

  console.log(JSON.stringify(decision));
}

main();
```

**Run**: `bun run evaluator.js < context.json > decision.json`

### Python

```python
#!/usr/bin/env python3

import json
import sys

def main():
    context = json.loads(sys.stdin.read())

    decision = {
        "decision": "pass" if context["response"]["status"] < 400 else "fail",
        "reason": f"HTTP {context['response']['status']}"
    }

    print(json.dumps(decision))

if __name__ == "__main__":
    main()
```

**Run**: `python3 evaluator.py < context.json > decision.json`

### Go

```go
package main

import (
    "encoding/json"
    "os"
    "github.com/vikasavnish/httptool/pkg/ir"
)

func main() {
    var ctx ir.EvaluationContext
    json.NewDecoder(os.Stdin).Decode(&ctx)

    decision := &ir.EvaluatorDecision{
        Decision: "pass",
        Reason:   "HTTP OK",
    }

    if ctx.Response.Status >= 400 {
        decision.Decision = "fail"
    }

    json.NewEncoder(os.Stdout).Encode(decision)
}
```

**Run**: `./evaluator < context.json > decision.json`

### Rust

```rust
use serde::{Deserialize, Serialize};
use std::io::{self, Read};

#[derive(Deserialize)]
struct Context {
    response: Response,
}

#[derive(Deserialize)]
struct Response {
    status: u16,
}

#[derive(Serialize)]
struct Decision {
    decision: String,
    reason: String,
}

fn main() {
    let mut input = String::new();
    io::stdin().read_to_string(&mut input).unwrap();

    let ctx: Context = serde_json::from_str(&input).unwrap();

    let decision = Decision {
        decision: if ctx.response.status < 400 { "pass" } else { "fail" }.to_string(),
        reason: format!("HTTP {}", ctx.response.status),
    };

    println!("{}", serde_json::to_string(&decision).unwrap());
}
```

**Run**: `./evaluator < context.json > decision.json`

## Advanced Patterns

### Conditional Retries

```javascript
if (response.status === 503) {
  const attempt = vars.attempt || 1;
  decision.decision = attempt < 3 ? "retry" : "fail";
  decision.actions = { retry_after_ms: 2000 };
}
```

### Header-Based Backoff

```javascript
const retryAfter = response.headers["retry-after"];
if (retryAfter) {
  decision.actions = {
    retry_after_ms: parseInt(retryAfter) * 1000
  };
}
```

### Token Refresh Flow

```javascript
if (response.status === 401 && vars.refresh_token) {
  decision.decision = "branch";
  decision.actions = { goto: "refresh_auth" };
  decision.mutations = {
    vars: { original_request: ir }
  };
}
```

### Circuit Breaker Pattern

```javascript
const failureRate = vars.recent_failures / vars.total_requests;
if (failureRate > 0.5) {
  decision.decision = "fail";
  decision.reason = "Circuit breaker open (50% failure rate)";
}
```

## Validation

Orchestrator validates evaluator output:

✅ **Required**:
- `decision` must be "pass", "retry", "fail", or "branch"
- If `decision === "branch"`, `actions.goto` must be present

✅ **Optional but validated**:
- `retry_after_ms` must be >= 0
- `max_retries` must be >= 1

❌ **Invalid** (falls back to default evaluator):
- Missing `decision` field
- Invalid JSON
- Unknown decision type
- Branch without goto target

## Security

Evaluators run in **process sandboxing**:

1. Separate process per evaluation
2. Timeout enforcement (kills hung processes)
3. No filesystem access (by convention, can enforce with seccomp)
4. No network access (by convention, can enforce with namespaces)
5. Stdin/stdout only communication

**Future** (WASM):
- Complete capability-based sandboxing
- No syscalls
- Deterministic execution
- Memory limits

## Performance

**Typical latencies**:
- Bun: ~50ms startup + ~1ms evaluation
- Python: ~100ms startup + ~5ms evaluation
- Go: ~1ms startup + ~0.1ms evaluation
- WASM (future): ~10ms startup + ~0.5ms evaluation

**Optimization tips**:
1. Use Go for hot paths
2. Pool evaluator processes (future)
3. Use gRPC instead of exec (future)
4. Keep evaluation logic simple (<100 LOC)

## Summary

Evaluators are **pure functions**:

```
(IR + Request + Response + Vars) → Decision
```

They are:
- ✅ Stateless
- ✅ Sandboxed
- ✅ Replaceable
- ✅ Polyglot
- ✅ Testable
- ✅ Composable

**All business logic lives here. The Go executor is completely dumb.**
