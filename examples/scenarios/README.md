# Load Testing Scenarios

## Overview

This directory contains `.httpx` scenario files that define load testing scenarios using our flexible DSL.

## Features

✅ **curl-first**: Paste curl commands directly
✅ **Named blocks**: Define reusable request blocks
✅ **Variable extraction**: Pull data from responses
✅ **Nested flows**: Parent → child request chains
✅ **Parallel execution**: Concurrent requests
✅ **Conditional logic**: if/else flows
✅ **Load patterns**: VUs, RPS, iterations, stages
✅ **Assertions**: Built-in validation
✅ **No whitespace sensitivity**: Flexible formatting

## Quick Start

### Basic Load Test

```
var base_url = "https://api.example.com"

request health {
  curl ${base_url}/health
  assert status == 200
}

scenario test {
  load 100 rps for 1m
  run health
}
```

### User Journey with Nested Requests

```
request login {
  curl -X POST https://api.example.com/login -d '{...}'
  extract token = $.access_token
}

request get_profile {
  curl https://api.example.com/profile \
    -H "Authorization: Bearer ${token}"
}

scenario journey {
  load 10 vus for 5m

  run login {
    run get_profile
  }
}
```

## File Structure

- `simple-load.httpx` - Basic load testing
- `user-journey.httpx` - Complete user flow with nested requests
- `conditional-flow.httpx` - if/else logic and retries
- `parallel-dashboard.httpx` - Parallel API calls

## Running Scenarios

```bash
# Run scenario
httptool scenario run user-journey.httpx

# Run specific scenario (if file has multiple)
httptool scenario run scenarios.httpx --scenario smoke_test

# Override load config
httptool scenario run user-journey.httpx --vus 50 --duration 10m

# Dry run (validate without executing)
httptool scenario run --dry-run user-journey.httpx

# Generate HTML report
httptool scenario run user-journey.httpx --report report.html
```

## Syntax Reference

### Variables

```
# Global variables
var base_url = "https://api.example.com"
var api_key = env.API_KEY
var email = "user-${VU}@test.com"

# Built-in variables
${VU}       # Virtual user number
${ITER}     # Iteration number
${TIME}     # Current timestamp
${UUID}     # Random UUID
```

### Requests

```
# Block style
request login {
  curl -X POST ${base_url}/login \
    -d '{"email":"test@test.com","password":"secret"}'

  extract {
    token = $.access_token
    user_id = $.user.id
  }

  assert {
    status == 200
    latency < 500
  }
}

# Shorthand
req health: curl ${base_url}/health | assert status==200

# One-liner
req health: curl ${base_url}/health
```

### Load Configuration

```
# VUs (virtual users)
load {
  vus = 10
  duration = 5m
  ramp_up = 30s
}

# RPS (requests per second)
load {
  rps = 100
  duration = 2m
}

# Iterations
load {
  iterations = 1000
  vus = 20
}

# Stages (ramp up/down)
load {
  stage { duration = 1m, vus = 10 }
  stage { duration = 3m, vus = 50 }
  stage { duration = 1m, vus = 10 }
}

# Shorthand
load 10 vus for 5m
load 100 rps for 2m
```

### Flow Control

```
# Sequential
run login -> get_profile -> update_profile

# Nested
run login {
  run get_profile {
    run update_settings
  }
}

# Parallel
parallel {
  run get_user
  run get_notifications
  run get_feed
}

# Conditional
run check_feature
if ${feature_enabled} == true {
  run new_api
} else {
  run old_api
}
```

### Extraction

```
extract {
  token = $.access_token           # JSONPath
  user_id = $.user.id
  session = regex:session=([^;]+)  # Regex
  req_id = header:X-Request-ID     # Header
  csrf = cookie:csrf_token         # Cookie
}

# Inline
extract token=$.access_token, user_id=$.user.id
```

### Assertions

```
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

### Retries

```
retry {
  max_attempts = 5
  backoff = exponential
  base_delay = 100ms
  max_delay = 5s
}
```

## Advanced Examples

### Complete Workflow

```
var base = "https://api.example.com"

request register {
  curl -X POST ${base}/register -d '{"email":"user-${VU}@test.com"}'
  extract user_id=$.id, token=$.access_token
  assert status==201
}

request verify {
  curl -X POST ${base}/verify -H "Authorization: Bearer ${token}"
  assert status==200
}

request get_profile {
  curl ${base}/users/${user_id} -H "Authorization: Bearer ${token}"
  assert status==200, body.id==${user_id}
}

request update {
  curl -X PATCH ${base}/users/${user_id} \
    -H "Authorization: Bearer ${token}" \
    -d '{"bio":"Test"}'
  assert status==200
}

scenario full_flow {
  load {
    vus = 10
    duration = 5m
  }

  run register {
    run verify {
      run get_profile
      run update
    }
  }
}
```

### Parallel Dashboard Load

```
request login {
  curl -X POST ${base}/login -d '{...}'
  extract token=$.access_token
}

scenario dashboard {
  load 20 vus for 2m

  run login
  parallel {
    run get_user
    run get_notifications
    run get_feed
    run get_stats
    run get_messages
  }
}
```

## Tips

1. **Use named blocks** - Define requests once, reuse everywhere
2. **Extract variables** - Pass data between requests
3. **Add assertions** - Validate responses automatically
4. **Nest requests** - Model realistic user flows
5. **Parallelize** - Speed up independent requests
6. **Use retries** - Handle flaky endpoints

## Next Steps

- Read [DSL Specification](../../docs/dsl-spec.md) for complete reference
- See [Architecture](../../docs/architecture.md) for how it works
- Check [Quick Start](../../docs/quick-start.md) for installation

## Support

- GitHub: https://github.com/vikasavnish/httptool
- Issues: https://github.com/vikasavnish/httptool/issues
- Docs: https://github.com/vikasavnish/httptool/docs
