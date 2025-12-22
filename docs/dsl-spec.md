# Load Testing DSL Specification

## Design Philosophy

- **Not whitespace-sensitive** (unlike YAML)
- **Named blocks** that you can reference and compose
- **curl-first**: Paste curl commands directly
- **Flexible syntax**: Multiple ways to express the same thing
- **Link sections**: Reference blocks by name

## Syntax

### File Extension
`.httpx` or `.loadtest`

### Basic Structure

```
# Comments start with #

# Define named request blocks
@request login
curl -X POST https://api.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"secret"}'

extract {
  token = $.access_token
  user_id = $.user.id
}

assert {
  status == 200
  latency < 500ms
}
end

# Define another block
@request get_profile
curl https://api.example.com/users/${user_id} \
  -H "Authorization: Bearer ${token}"

assert {
  status == 200
}
end

# Link them together in a scenario
@scenario user_journey
load {
  vus = 10
  duration = 5m
}

run login
  then get_profile
end
```

### Alternative: Inline Style

```
# No @decorators needed, just blocks

request login {
  curl -X POST https://api.example.com/login \
    -d '{"email":"user@test.com","password":"test"}'

  extract {
    token = $.access_token
  }
}

request get_profile {
  curl https://api.example.com/users/${user_id} \
    -H "Authorization: Bearer ${token}"
}

scenario user_journey {
  load: vus=10, duration=5m

  run login -> get_profile
}
```

### Alternative: JSON-like (No Indentation Required)

```javascript
request "login" {
  curl: `curl -X POST https://api.example.com/login -d '{"email":"test"}'`,
  extract: {
    token: "$.access_token",
    user_id: "$.user.id"
  },
  assert: {
    status: 200,
    latency: "<500ms"
  }
}

request "get_profile" {
  curl: `curl https://api.example.com/users/${user_id} -H "Authorization: Bearer ${token}"`,
  assert: { status: 200 }
}

scenario "user_journey" {
  load: { vus: 10, duration: "5m" },
  flow: ["login", "get_profile"]
}
```

## Core Concepts

### 1. Named Blocks

Define reusable request blocks:

```
# Style 1: @ decorator
@request my_request
curl https://example.com
end

# Style 2: Block syntax
request my_request {
  curl https://example.com
}

# Style 3: Shorthand
req my_request: curl https://example.com
```

### 2. Variables

Define once, use everywhere:

```
# Global variables
var base_url = "https://api.example.com"
var api_key = env.API_KEY
var test_email = "user-${VU}@test.com"

# Built-in variables
${VU}       # Virtual user number (1-N)
${ITER}     # Iteration number
${TIME}     # Current timestamp
${UUID}     # Random UUID
${COUNTER}  # Auto-increment counter

# Use in requests
request test {
  curl ${base_url}/users \
    -H "X-API-Key: ${api_key}"
}
```

### 3. Extraction

Pull data from responses:

```
request login {
  curl -X POST https://api.example.com/login -d '{...}'

  # Multiple syntaxes
  extract {
    token = $.access_token           # JSONPath
    user_id = $.user.id
    session = regex:session=([^;]+)  # Regex
    req_id = header:X-Request-ID     # Header
    csrf = cookie:csrf_token         # Cookie
  }

  # Alternative: inline
  extract token=$.access_token, user_id=$.user.id
}
```

### 4. Assertions

Validate responses:

```
request test {
  curl https://example.com

  # Block style
  assert {
    status == 200
    status in [200, 201, 204]
    latency < 500ms
    latency between 100ms and 1000ms
    body.success == true
    body.items.length > 0
    body.email contains "@example.com"
    header.content-type == "application/json"
  }

  # Alternative: inline
  assert status==200, latency<500ms
}
```

### 5. Linking Requests (Flow Control)

```
# Sequential
scenario flow1 {
  run login -> get_profile -> update_profile
}

# Nested (child requests)
scenario flow2 {
  run login {
    run get_profile {
      run update_settings
    }
  }
}

# Conditional
scenario flow3 {
  run check_feature
  if ${feature_enabled} == true {
    run new_api
  } else {
    run old_api
  }
}

# Parallel
scenario flow4 {
  run login
  parallel {
    run get_user
    run get_notifications
    run get_feed
  }
}

# Mix
scenario complex {
  run login
    -> get_profile
    -> parallel(get_stats, get_activity, get_settings)
    -> update_profile
}
```

### 6. Load Configuration

```
# Style 1: Block
load {
  vus = 10
  duration = 5m
}

# Style 2: Inline
load: vus=10, duration=5m, rps=100

# Style 3: Stages
load {
  stage { duration=1m, vus=10 }
  stage { duration=3m, vus=50 }
  stage { duration=1m, vus=10 }
}

# Style 4: Shorthand
load 10 vus for 5m
load 100 rps for 2m
load 1000 iterations with 20 vus
```

## Complete Examples

### Example 1: Simple Test (Minimal Syntax)

```
var base = "https://api.example.com"

req health: curl ${base}/health
assert status==200, latency<100ms

scenario test {
  load 100 rps for 1m
  run health
}
```

### Example 2: User Journey with Named Blocks

```
# Variables
var base_url = "https://api.example.com"
var email = "user-${VU}@test.com"

# Request blocks
request register {
  curl -X POST ${base_url}/register \
    -H "Content-Type: application/json" \
    -d '{"email":"${email}","password":"test123"}'

  extract {
    user_id = $.id
    token = $.access_token
  }

  assert status == 201
}

request get_profile {
  curl ${base_url}/users/${user_id} \
    -H "Authorization: Bearer ${token}"

  assert status == 200, body.id == ${user_id}
}

request update_profile {
  curl -X PATCH ${base_url}/users/${user_id} \
    -H "Authorization: Bearer ${token}" \
    -d '{"bio":"Test user"}'

  assert status == 200
}

request upload_avatar {
  curl -X POST ${base_url}/users/${user_id}/avatar \
    -H "Authorization: Bearer ${token}" \
    -F "file=@avatar.jpg"

  assert status == 200
}

# Scenario
scenario user_journey {
  load {
    vus = 10
    duration = 5m
    ramp_up = 30s
  }

  run register {
    run get_profile {
      run update_profile
      run upload_avatar
    }
  }
}
```

### Example 3: Conditional Flow

```
request check_feature {
  curl https://api.example.com/features/new_api
  extract enabled = $.enabled
}

request new_api {
  curl https://api.example.com/v2/endpoint
}

request old_api {
  curl https://api.example.com/v1/endpoint
}

scenario adaptive {
  load 50 rps for 2m

  run check_feature
  if ${enabled} == true {
    run new_api
  } else {
    run old_api
  }
}
```

### Example 4: Parallel Dashboard Load

```
request login {
  curl -X POST https://api.example.com/login -d '{...}'
  extract token = $.access_token
}

request get_user {
  curl https://api.example.com/user -H "Authorization: Bearer ${token}"
}

request get_notifications {
  curl https://api.example.com/notifications -H "Authorization: Bearer ${token}"
}

request get_feed {
  curl https://api.example.com/feed -H "Authorization: Bearer ${token}"
}

request get_stats {
  curl https://api.example.com/stats -H "Authorization: Bearer ${token}"
}

scenario dashboard {
  load 20 vus for 2m

  run login
  parallel {
    run get_user
    run get_notifications
    run get_feed
    run get_stats
  }
}
```

### Example 5: Multi-Scenario File

```
# Define requests once
request health {
  curl https://api.example.com/health
  assert status == 200
}

request login {
  curl -X POST https://api.example.com/login -d '{...}'
  extract token = $.access_token
}

request get_data {
  curl https://api.example.com/data -H "Authorization: Bearer ${token}"
}

# Scenario 1: Smoke test
scenario smoke {
  load 1 vus for 10s
  run health
}

# Scenario 2: Load test
scenario load {
  load 100 rps for 5m
  run login -> get_data
}

# Scenario 3: Spike test
scenario spike {
  load {
    stage { duration=1m, vus=10 }
    stage { duration=10s, vus=100 }  # Spike!
    stage { duration=1m, vus=10 }
  }
  run login -> get_data
}

# Run specific scenario
# httptool run test.httpx --scenario smoke
```

### Example 6: Data-Driven Testing

```
# Define data
data products = [
  { id: 1, name: "Product A" },
  { id: 2, name: "Product B" },
  { id: 3, name: "Product C" }
]

request get_product {
  curl https://api.example.com/products/${product.id}
  assert status == 200, body.name == "${product.name}"
}

scenario test_products {
  load 5 vus for 30s

  foreach product in products {
    run get_product
  }
}
```

### Example 7: Retry Logic

```
request unreliable {
  curl https://api.example.com/flaky

  retry {
    max_attempts = 5
    backoff = exponential
    base_delay = 100ms
    max_delay = 5s
  }

  assert status in [200, 201]
}

scenario chaos {
  load 10 vus for 5m
  run unreliable
}
```

## Flexible Syntax Examples

### Same scenario, 4 different styles:

**Style 1: Verbose**
```
request login {
  curl -X POST https://api.example.com/login -d '{...}'
  extract {
    token = $.access_token
  }
  assert {
    status == 200
  }
}

scenario test {
  load {
    vus = 10
    duration = 1m
  }
  run login
}
```

**Style 2: Compact**
```
req login: curl -X POST https://api.example.com/login -d '{...}'
extract token=$.access_token
assert status==200

scenario test {
  load: vus=10, duration=1m
  run login
}
```

**Style 3: One-liner**
```
req login: curl -X POST https://api.example.com/login -d '{...}' | extract token=$.access_token | assert status==200
scenario test: load 10 vus for 1m | run login
```

**Style 4: Arrow syntax**
```
login = curl -X POST https://api.example.com/login -d '{...}'
  >> extract token=$.access_token
  >> assert status==200

test = load(10 vus, 1m) >> run(login)
```

## CLI Usage

```bash
# Run scenario file
httptool run scenario.httpx

# Run specific scenario
httptool run scenario.httpx --scenario smoke

# Override load config
httptool run scenario.httpx --vus 50 --duration 10m

# Dry run (show what would execute)
httptool run --dry-run scenario.httpx

# Validate syntax
httptool validate scenario.httpx

# Convert to IR tree
httptool convert scenario.httpx -o scenario.json

# Generate report
httptool run scenario.httpx --report report.html

# Debug mode
httptool run --debug scenario.httpx
```

## Advanced Features

### Think Time
```
scenario realistic {
  load 10 vus for 5m
  think_time = 1s ± 0.5s  # Random between 0.5s-1.5s

  run login
  think 2s
  run get_profile
  think 1s
  run update_settings
}
```

### Setup/Teardown
```
setup {
  run create_test_data
  extract test_id = $.id
}

scenario main {
  load 10 vus for 1m
  run test_endpoint  # Uses ${test_id}
}

teardown {
  run cleanup
}
```

### Shared State
```
shared session_pool = []

scenario login_pool {
  load 100 vus for 10m

  run login
  shared.session_pool.push(${token})

  run use_random_session {
    var random_token = shared.session_pool.random()
  }
}
```

## Benefits

✅ **No whitespace sensitivity** - Use any indentation
✅ **Named blocks** - Define once, reference everywhere
✅ **curl-first** - Paste curl commands directly
✅ **Flexible syntax** - Multiple ways to write same thing
✅ **Link sections** - Compose flows easily
✅ **Variable passing** - Extract and reuse data
✅ **Conditional logic** - if/else, parallel, foreach
✅ **Load patterns** - VUs, RPS, stages, iterations
✅ **Assertions** - Built-in validation
✅ **Retries** - Smart backoff strategies

Next: Implementation...
