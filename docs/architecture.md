# Architecture

## Overview

The HTTP Execution & Evaluation Engine follows a strict separation of concerns:

```
┌─────────────────┐
│  Input Sources  │  curl, k6, Postman, HAR, OpenAPI
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Wrappers      │  Convert to canonical IR
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  JSON IR v1.0   │  ← The Contract
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Go Executor    │  Pure HTTP orchestration (no logic)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Eval Context    │  request + response + vars
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Evaluators     │  External, polyglot (JS, Python, Go, WASM)
│  (sandboxed)    │  Contains ALL business logic
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Decision JSON   │  pass / retry / fail / branch
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Orchestrator   │  Applies decision, handles retries
└─────────────────┘
```

## Core Principles

### 1. IR is the Product

The JSON Intermediate Representation (IR) is the **canonical contract**. Everything flows through IR:

- curl commands → IR
- k6 scripts → IR
- Postman collections → IR
- OpenAPI examples → IR
- Manual definitions → IR

IR is:
- **Versioned**: Breaking changes require version bump
- **Stable**: Once published, v1.0 never changes semantically
- **Language-agnostic**: JSON, not Go structs
- **Complete**: Contains everything needed for execution

### 2. Execution ≠ Evaluation

**Executor responsibilities** (Go):
- Parse IR
- Build HTTP request
- Execute network call
- Capture response
- Build evaluation context
- **NO BUSINESS LOGIC**

**Evaluator responsibilities** (External):
- Analyze response
- Apply business rules
- Make decisions (retry? fail? pass?)
- Extract data
- Mutate future requests
- **ALL BUSINESS LOGIC**

### 3. Polyglot by Design

Evaluators can be written in any language:

```javascript
// Bun (JavaScript) - Fast startup, rich ecosystem
const decision = {
  decision: response.status === 429 ? "retry" : "pass",
  reason: "Rate limited",
  actions: { retry_after_ms: 1000 }
};
```

```python
# Python - ML/data science ready
decision = {
    "decision": "retry" if response["status"] == 429 else "pass",
    "reason": "Rate limited",
    "actions": {"retry_after_ms": 1000}
}
```

```go
// Go - Fast-path evaluation
decision := &ir.EvaluatorDecision{
    Decision: "retry",
    Reason:   "Rate limited",
    Actions:  &ir.Actions{RetryAfterMs: 1000},
}
```

### 4. Sandboxed Execution

Evaluators run isolated:

- **Process boundary**: Execute as separate processes
- **Resource limits**: CPU, memory, timeout enforced
- **No shared state**: Stateless by contract
- **Stdin/stdout only**: JSON in, JSON out
- **Future WASM**: Complete sandboxing

### 5. Everything is Replaceable

No component is sacred:

- Swap Bun evaluator for Python
- Replace executor implementation
- Add new wrapper for any tool
- Introduce WASM evaluators
- Plug in AI/LLM evaluators

## Data Flow

### 1. Input → IR Conversion

```go
// curl parser
parser := parser.NewCurlParser()
ir, _ := parser.Parse("curl -X POST https://api.example.com/users")

// k6 wrapper
wrapper := wrappers.NewK6Wrapper()
ir, _ := wrapper.ConvertFromJSON(k6JSON)

// Direct IR
ir := &ir.IR{
    Version: "1.0",
    Request: ir.Request{
        Method: "GET",
        URL:    "https://api.example.com/users",
    },
}
```

### 2. Execution

```go
executor := executor.NewExecutor()
evalContext, err := executor.Execute(ir)
```

Executor:
1. Builds HTTP request from IR
2. Configures transport (TLS, proxy, timeout)
3. Executes HTTP call
4. Captures timing and response
5. Returns evaluation context (NO DECISIONS)

### 3. Evaluation

```go
manager := evaluator.NewManager(5 * time.Second)
decision, err := manager.Evaluate(ctx, evalContext, "bun", "evaluator.js")
```

Evaluator manager:
1. Serializes context to JSON
2. Spawns evaluator process
3. Pipes JSON via stdin
4. Reads decision from stdout
5. Validates decision schema
6. Returns decision

### 4. Orchestration

```go
orchestrator := orchestrator.NewOrchestrator(maxRetries, evalTimeout)
result, err := orchestrator.ExecuteOne(ctx, ir)
```

Orchestrator:
1. Executes IR
2. Gets decision from evaluator
3. Handles `retry`: applies mutations, waits, re-executes
4. Handles `fail`: returns error
5. Handles `pass`: returns success
6. Handles `branch`: (future) navigates workflow

## Extension Points

### 1. Custom Wrappers

```go
type Wrapper interface {
    Convert(input any) (*ir.IR, error)
}

// Implement for any tool
type LocustWrapper struct{}
type PostmanWrapper struct{}
type HARWrapper struct{}
type OpenAPIWrapper struct{}
```

### 2. Custom Evaluators

Any language, any runtime:

```bash
# JavaScript (Bun)
bun run evaluator.js < context.json > decision.json

# Python
python3 evaluator.py < context.json > decision.json

# Rust
./evaluator < context.json > decision.json

# WASM
wasmtime evaluator.wasm < context.json > decision.json
```

### 3. AI Evaluators

Future: LLM-powered evaluation

```python
# evaluator-ai.py
import anthropic

context = json.loads(sys.stdin.read())
prompt = f"Analyze this API response: {context['response']}"

client = anthropic.Anthropic()
message = client.messages.create(
    model="claude-sonnet-4",
    messages=[{"role": "user", "content": prompt}]
)

decision = extract_decision(message.content)
print(json.dumps(decision))
```

### 4. Load Testing Modes

```go
// Single execution
result, _ := orchestrator.ExecuteOne(ctx, ir)

// Concurrent batch
results, stats := orchestrator.ExecuteConcurrent(ctx, irs, concurrency)

// Load test (RPS-based)
results, stats := orchestrator.ExecuteLoad(ctx, ir, duration, rps)

// Replay (sequential)
results, stats := orchestrator.Replay(ctx, irs)
```

## Security Model

### Evaluator Sandboxing

**Current** (Process isolation):
- Spawned as child process
- Timeout enforcement
- No filesystem access (can be added via seccomp)
- No network access (can be added via network namespaces)

**Future** (WASM):
- Complete sandboxing
- Capability-based security
- Deterministic execution
- No side effects

### Input Validation

- JSON schema validation on IR
- JSON schema validation on decisions
- No arbitrary code execution in Go
- All user input passes through IR parser

### Resource Limits

- Timeout on evaluators (default 5s)
- Timeout on HTTP requests (configurable)
- Memory limits (OS-level)
- CPU limits (OS-level)

## Performance Characteristics

### Executor
- **Latency**: Network latency + minimal overhead (~1ms)
- **Throughput**: Limited by network, not CPU
- **Memory**: O(response size)

### Evaluator
- **Bun**: ~50ms startup, ~1ms evaluation
- **Python**: ~100ms startup, ~5ms evaluation
- **Go**: ~1ms startup, ~0.1ms evaluation
- **WASM**: ~10ms startup, ~0.5ms evaluation (future)

### Orchestrator
- **Concurrent**: Goroutine per request (10k+ concurrent)
- **Load test**: Ticker-based RPS control
- **Replay**: Sequential, deterministic

## Future Directions

### 1. Visual Workflow Builder
- Drag-and-drop IR construction
- Branching/conditional flows
- Variable extraction/passing
- Multi-step scenarios

### 2. CI/CD Integration
- GitHub Actions plugin
- GitLab CI integration
- API contract testing
- Regression detection

### 3. Chaos Testing
- Random failures
- Latency injection
- Partial response simulation
- Network partition testing

### 4. Smart Retries
- ML-based retry prediction
- Adaptive backoff
- Circuit breakers
- Bulkheading

### 5. Trading/Broker Safety
- Pre-flight validation
- Mutation testing
- Order replay
- Risk checks

## Design Trade-offs

### Why External Evaluators?

**Pros**:
- Complete language flexibility
- Easy to swap/upgrade
- Natural sandboxing
- No embedding complexity
- Future AI integration

**Cons**:
- Process spawn overhead (~50-100ms)
- IPC serialization cost
- Harder debugging

**Mitigation**:
- Go fast-path evaluator for hot paths
- Connection pooling (future)
- gRPC evaluators (future)

### Why JSON IR vs Code Generation?

**Pros**:
- Language-agnostic
- Versionable
- Storable/replayable
- Inspectable/debuggable
- Tooling-friendly

**Cons**:
- Verbose
- No compile-time checks
- Requires runtime validation

**Mitigation**:
- JSON schema validation
- Code generation from IR (future)
- IDE plugins with autocomplete (future)

## Summary

This architecture prioritizes:

1. **Extensibility**: Easy to add new tools, evaluators, features
2. **Stability**: IR is the contract, implementations can change
3. **Flexibility**: Polyglot evaluators, multiple execution modes
4. **Safety**: Sandboxed evaluation, resource limits, validation
5. **Future-proofing**: AI evaluators, visual builders, advanced features

The system is designed to **live for years**, not months.
