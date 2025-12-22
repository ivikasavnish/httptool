# Load Testing Quick Start

## ✅ You're Ready to Go!

The load testing DSL is fully integrated and working. Here's how to use it:

## 1. Create a Scenario File

Create a `.httpx` file with your test scenario:

```httpx
# my-test.httpx
var base_url = "https://httpbin.org"

request health_check {
  curl ${base_url}/get

  assert {
    status == 200
    latency < 2000
  }
}

scenario load_test {
  load {
    vus = 10
    duration = 30s
  }

  run health_check
}
```

## 2. Validate Your Scenario

```bash
./bin/httptool scenario validate my-test.httpx
```

Output:
```
✓ Scenario file is valid
  Variables: 1
  Requests: 1
  Scenarios: 1

Scenarios:
  - load_test
```

## 3. Run a Dry-Run

Test without executing:

```bash
./bin/httptool scenario run my-test.httpx --dry-run
```

## 4. Execute the Load Test

```bash
./bin/httptool scenario run my-test.httpx
```

Output:
```
📋 Parsing scenario: my-test.httpx
✓ Parsed successfully

🚀 Preparing scenario: load_test
✓ Compiled successfully

⚡ Load Configuration:
  Virtual Users: 10
  Duration: 30s

🏃 Executing scenario...

======================================================================
  Scenario: load_test
======================================================================

⏱  Duration: 30.123s
👥 VUs: 10

📊 Results:
  Total Requests:      150
  ✓ Successful:        150 (100.0%)
  ✗ Failed:            0 (0.0%)

⚡ Latency:
  Avg:   245.50 ms
  Min:    120.00 ms
  Max:    890.00 ms

📦 Data Transferred: 0.45 MB

🚀 Throughput: 4.98 req/sec
======================================================================
```

## Example Scenarios

### Basic Load Test

```httpx
var api = "https://api.example.com"

request test {
  curl ${api}/endpoint
  assert status == 200
}

scenario test {
  load 100 rps for 1m  # 100 requests/sec for 1 minute
  run test
}
```

### User Journey with Nested Requests

```httpx
var base = "https://api.example.com"
var email = "user-${VU}@test.com"

request register {
  curl -X POST ${base}/register \
    -d '{"email":"${email}","password":"test123"}'

  extract {
    user_id = $.id
    token = $.access_token
  }

  assert status == 201
}

request get_profile {
  curl ${base}/users/${user_id} \
    -H "Authorization: Bearer ${token}"

  assert status == 200
}

scenario journey {
  load {
    vus = 10
    duration = 5m
  }

  run register {
    run get_profile
  }
}
```

### Parallel Requests

```httpx
request login {
  curl -X POST ${base}/login -d '{...}'
  extract token = $.access_token
}

request get_user {
  curl ${base}/user -H "Authorization: Bearer ${token}"
}

request get_stats {
  curl ${base}/stats -H "Authorization: Bearer ${token}"
}

scenario dashboard {
  load 20 vus for 2m

  run login
  parallel {
    run get_user
    run get_stats
  }
}
```

## Available Commands

```bash
# Validate syntax
./bin/httptool scenario validate <file.httpx>

# Dry run (no execution)
./bin/httptool scenario run <file.httpx> --dry-run

# Run scenario
./bin/httptool scenario run <file.httpx>

# Show compiled info
./bin/httptool scenario convert <file.httpx>

# Run specific scenario (if file has multiple)
./bin/httptool scenario run <file.httpx> --scenario scenario_name
```

## Load Patterns

### Virtual Users (VUs)
```httpx
load {
  vus = 10        # 10 concurrent virtual users
  duration = 5m   # Run for 5 minutes
}

# Shorthand
load 10 vus for 5m
```

### Requests Per Second (RPS)
```httpx
load {
  rps = 100       # 100 requests per second
  duration = 2m   # Run for 2 minutes
}

# Shorthand
load 100 rps for 2m
```

### Iterations
```httpx
load {
  iterations = 1000  # Total iterations
  vus = 20           # Distributed across 20 VUs
}
```

## Variable System

### Built-in Variables
- `${VU}` - Virtual user number (1-N)
- `${ITER}` - Iteration number
- `${TIME}` - Current timestamp
- `${UUID}` - Random UUID

### Custom Variables
```httpx
var base_url = "https://api.example.com"
var api_key = env.API_KEY  # From environment
var test_email = "user-${VU}@test.com"  # Per-VU email
```

### Extract from Responses
```httpx
request login {
  curl -X POST ${base}/login -d '{...}'

  extract {
    token = $.access_token     # JSONPath
    user_id = $.user.id
    session = regex:session=([^;]+)  # Regex
    req_id = header:X-Request-ID     # Header
  }
}

# Use extracted variables
request next {
  curl ${base}/users/${user_id} \
    -H "Authorization: Bearer ${token}"
}
```

## Assertions

```httpx
assert {
  status == 200
  status in [200, 201, 204]
  latency < 500
  latency between 100 and 1000
  body.success == true
  body.items.length > 0
  body.email contains "@example.com"
  header.content-type == "application/json"
}

# Inline
assert status==200, latency<500
```

## Retries

```httpx
request flaky {
  curl https://api.example.com/unreliable

  retry {
    max_attempts = 5
    backoff = exponential
    base_delay = 100ms
    max_delay = 5s
  }
}
```

## Tips

1. **Start small**: Test with 1-2 VUs first
2. **Use dry-run**: Validate before executing
3. **Add assertions**: Catch failures early
4. **Extract variables**: Build realistic flows
5. **Monitor output**: Watch for failures/latency

## Next Steps

- See `examples/scenarios/` for more examples
- Read `docs/dsl-spec.md` for complete reference
- Check `docs/architecture.md` for how it works

## Troubleshooting

### Parse errors
```bash
# Validate syntax first
./bin/httptool scenario validate my-test.httpx
```

### Connection errors
- Check URL is accessible
- Verify network connectivity
- Test with single request first

### High latency
- Reduce VUs/RPS
- Check target server capacity
- Add think time between requests

## Help

```bash
./bin/httptool scenario --help
./bin/httptool --help
```
