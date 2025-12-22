# Quick Start Guide

## Installation

### From Source

```bash
git clone https://github.com/vikasavnish/httptool
cd httptool
go build -o bin/httptool ./cmd/httptool
sudo mv bin/httptool /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/vikasavnish/httptool/cmd/httptool@latest
```

### Prerequisites

For evaluators:

```bash
# Bun (JavaScript evaluator)
curl -fsSL https://bun.sh/install | bash

# Python (Python evaluator)
# Already installed on most systems, or:
brew install python3  # macOS
apt-get install python3  # Ubuntu/Debian
```

## Basic Usage

### 1. Execute a curl Command

```bash
httptool exec 'curl https://httpbin.org/get'
```

Output:
```
Request:  GET https://httpbin.org/get
Status:   200
Latency:  145.23ms
Size:     324 bytes

Decision: pass
Reason:   default pass
```

### 2. Convert curl to IR

```bash
httptool convert 'curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Alice\"}"' > request.json
```

View the IR:
```bash
cat request.json
```

```json
{
  "version": "1.0",
  "metadata": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "source": "curl",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "request": {
    "method": "POST",
    "url": "https://api.example.com/users",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "type": "json",
      "content": {
        "name": "Alice"
      }
    }
  },
  "transport": {
    "tls_verify": true,
    "follow_redirects": true,
    "max_redirects": 10,
    "timeout_ms": 30000
  },
  "evaluation": {
    "evaluator": "bun",
    "timeout_ms": 5000,
    "vars": {}
  }
}
```

### 3. Execute from IR File

```bash
httptool run request.json
```

### 4. Use Custom Evaluator

```bash
httptool run request.json \
  --evaluator bun \
  --evaluator-path ./examples/custom-evaluator.js
```

## Common Workflows

### API Testing

```bash
# 1. Test endpoint availability
httptool exec 'curl https://api.example.com/health'

# 2. Test authentication
httptool exec 'curl https://api.example.com/login \
  -X POST \
  -d "{\"username\":\"test\",\"password\":\"secret\"}"'

# 3. Test with authentication token
httptool exec 'curl https://api.example.com/users \
  -H "Authorization: Bearer YOUR_TOKEN"'
```

### Load Testing

Create a load test script:

```go
package main

import (
    "context"
    "time"
    "github.com/vikasavnish/httptool/pkg/ir"
    "github.com/vikasavnish/httptool/pkg/orchestrator"
)

func main() {
    // Define request
    irSpec := &ir.IR{
        Version: ir.Version,
        Request: ir.Request{
            Method: "GET",
            URL:    "https://api.example.com/endpoint",
        },
        Transport: ir.DefaultTransport(),
    }

    // Create orchestrator
    orch := orchestrator.NewOrchestrator(3, 5*time.Second)

    // Run load test: 100 RPS for 30 seconds
    results, stats := orch.ExecuteLoad(
        context.Background(),
        irSpec,
        30*time.Second,
        100,
    )

    // Print results
    fmt.Printf("Total: %d\n", stats.Total)
    fmt.Printf("Success: %d\n", stats.Success)
    fmt.Printf("Failed: %d\n", stats.Failed)
    fmt.Printf("Avg Latency: %.2fms\n", stats.AvgLatency)
}
```

### Replay Stored Requests

Save multiple requests:

```bash
# Save multiple IRs
httptool convert 'curl https://api.example.com/step1' > step1.json
httptool convert 'curl https://api.example.com/step2' > step2.json
httptool convert 'curl https://api.example.com/step3' > step3.json
```

Replay in sequence:

```go
package main

import (
    "context"
    "encoding/json"
    "os"
    "github.com/vikasavnish/httptool/pkg/ir"
    "github.com/vikasavnish/httptool/pkg/orchestrator"
)

func main() {
    // Load IRs
    files := []string{"step1.json", "step2.json", "step3.json"}
    var specs []*ir.IR

    for _, file := range files {
        data, _ := os.ReadFile(file)
        var spec ir.IR
        json.Unmarshal(data, &spec)
        specs = append(specs, &spec)
    }

    // Replay
    orch := orchestrator.NewOrchestrator(3, 5*time.Second)
    results, stats := orch.Replay(context.Background(), specs)

    // Check results
    for i, result := range results {
        fmt.Printf("Step %d: %s\n", i+1, result.Decision.Decision)
    }
}
```

## Environment Variables

### VERBOSE

Show response headers:

```bash
VERBOSE=1 httptool exec 'curl https://httpbin.org/get'
```

Output includes:
```
Response Headers:
  Content-Type: application/json
  Content-Length: 324
  Date: Mon, 15 Jan 2024 10:30:00 GMT
```

### SHOW_BODY

Show response body:

```bash
SHOW_BODY=1 httptool exec 'curl https://httpbin.org/get'
```

Output includes:
```
Response Body:
  {
    "args": {},
    "headers": {
      "Host": "httpbin.org"
    },
    "url": "https://httpbin.org/get"
  }
```

## Writing Custom Evaluators

### JavaScript (Bun)

Create `my-evaluator.js`:

```javascript
#!/usr/bin/env bun

async function main() {
  const input = await Bun.stdin.text();
  const context = JSON.parse(input);

  const decision = {
    decision: "pass",
    reason: "Custom logic",
    mutations: {},
    actions: {}
  };

  // Your custom logic
  if (context.response.status === 404) {
    decision.decision = "retry";
    decision.reason = "Not found, retrying";
    decision.actions.retry_after_ms = 1000;
  }

  console.log(JSON.stringify(decision));
}

main();
```

Make executable:
```bash
chmod +x my-evaluator.js
```

Use it:
```bash
httptool run request.json --evaluator bun --evaluator-path ./my-evaluator.js
```

### Python

Create `my-evaluator.py`:

```python
#!/usr/bin/env python3

import json
import sys

def main():
    context = json.loads(sys.stdin.read())

    decision = {
        "decision": "pass",
        "reason": "Custom logic"
    }

    # Your custom logic
    if context["response"]["status"] == 404:
        decision["decision"] = "retry"
        decision["reason"] = "Not found, retrying"
        decision["actions"] = {"retry_after_ms": 1000}

    print(json.dumps(decision))

if __name__ == "__main__":
    main()
```

Make executable:
```bash
chmod +x my-evaluator.py
```

Use it:
```bash
httptool run request.json --evaluator python --evaluator-path ./my-evaluator.py
```

## Integrating with Tools

### k6 to IR

```go
package main

import (
    "github.com/vikasavnish/httptool/pkg/wrappers"
)

func main() {
    k6JSON := `{
        "method": "POST",
        "url": "https://api.example.com/users",
        "body": {"name": "Alice"},
        "params": {
            "headers": {"Authorization": "Bearer token"},
            "timeout": "30s"
        }
    }`

    wrapper := wrappers.NewK6Wrapper()
    ir, _ := wrapper.ConvertFromJSON(k6JSON)

    // Use ir with executor
}
```

### Postman Collection

```go
// Coming soon: Postman wrapper
wrapper := wrappers.NewPostmanWrapper()
irs, _ := wrapper.ConvertCollection("collection.json")
```

## Next Steps

- Read the [Architecture](architecture.md) to understand the design
- Check [IR Schema Reference](ir-schema.md) for complete IR spec
- See [Evaluator Contract](evaluator-contract.md) for evaluator details
- Browse [examples/](../examples/) for more use cases
- Learn about [Tool Wrappers](wrappers.md) for integrations

## Troubleshooting

### Evaluator Not Found

```
Error: evaluator failed: exec: "bun": executable file not found in $PATH
```

**Solution**: Install Bun:
```bash
curl -fsSL https://bun.sh/install | bash
```

### Timeout Errors

```
Error: evaluator timeout after 5s
```

**Solution**: Increase timeout in IR:
```json
{
  "evaluation": {
    "timeout_ms": 10000
  }
}
```

### Invalid IR

```
Error: Invalid IR JSON
```

**Solution**: Validate IR:
```bash
httptool validate request.json
```

## Support

- GitHub Issues: https://github.com/vikasavnish/httptool/issues
- Documentation: https://github.com/vikasavnish/httptool/docs
- Examples: https://github.com/vikasavnish/httptool/examples
