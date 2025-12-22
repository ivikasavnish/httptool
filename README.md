# HTTP Execution & Evaluation Engine

**Production-grade, extensible HTTP execution and evaluation platform**

## Architecture Overview

```
curl / API spec / tool
        ↓
   HTTP JSON IR   ← canonical contract (versioned)
        ↓
   Go Executor (pure orchestration)
        ↓
  Evaluation Context (request + response)
        ↓
External Evaluators (polyglot, sandboxed)
        ↓
   Decision JSON (pass / retry / fail / branch)
        ↓
   Go applies decision (no logic inside Go)
```

## Core Principles

- **IR is the product**: Canonical JSON format is the contract
- **Execution ≠ Evaluation**: Go orchestrates, external evaluators decide
- **Logic lives outside Go**: Business rules in replaceable evaluators
- **Everything is replaceable**: Polyglot evaluators, extensible wrappers
- **No language lock-in**: Support JS, Python, Go, WASM evaluators
- **Future-proof**: AI/LLM evaluators fit naturally

## Project Structure

```
httptool/
├── cmd/
│   ├── httptool/           # Main CLI
│   └── evaluators/         # Reference evaluator implementations
│       ├── bun/            # JavaScript/Bun evaluator
│       ├── python/         # Python evaluator
│       └── go/             # Go evaluator (fast-path)
├── pkg/
│   ├── ir/                 # IR schema and validation
│   ├── parser/             # curl → IR converter
│   ├── executor/           # HTTP execution engine
│   ├── evaluator/          # Evaluator management
│   └── wrappers/           # Tool adapters (k6, Locust, etc.)
├── examples/               # Usage examples
├── schemas/                # JSON schemas for IR versions
└── docs/                   # Architecture and API docs
```

## Quick Start

```bash
# Install
go install github.com/vikasavnish/httptool/cmd/httptool@latest

# Execute a curl command
httptool exec 'curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d "{\"name\":\"test\"}"'

# Convert to IR
httptool convert 'curl https://example.com' > request.json

# Execute from IR
httptool run request.json

# With custom evaluator
httptool run request.json --evaluator ./evaluators/custom.js
```

## Features

### Core Capabilities

- ✅ curl → IR conversion with shell tokenization
- ✅ Pure Go HTTP executor (no business logic)
- ✅ Polyglot evaluators (JS, Python, Go, WASM)
- ✅ Sandboxed evaluation with resource limits
- ✅ Versioned IR schema with backward compatibility
- ✅ Extensible wrapper architecture

### Supported Evaluators

- **Bun (JavaScript)**: Fast startup, rich ecosystem
- **Python**: CPython/Mojo support, ML-ready
- **Go**: Fast-path native evaluation
- **WASM**: Future sandboxed execution

### Extension Points

- Tool wrappers (k6, Locust, Postman, HAR)
- Custom evaluators
- Load testing modes
- Replay engines
- CI/CD integrations

## Documentation

- [Architecture](docs/architecture.md)
- [IR Schema Reference](docs/ir-schema.md)
- [Evaluator Contract](docs/evaluator-contract.md)
- [Writing Custom Evaluators](docs/custom-evaluators.md)
- [Tool Wrappers](docs/wrappers.md)

## Security

- Evaluators run sandboxed (Docker/WASM)
- No arbitrary code execution in Go core
- JSON schema validation enforced
- Resource limits on all evaluators
- No shared mutable state

## Use Cases

- API testing with smart retries
- Load testing with custom logic
- CI/CD API gates
- Trading/broker safety rules
- Chaos testing
- AI-assisted evaluation
- Visual workflow automation

## License

MIT
