# Cookie Management

## Overview

httptool automatically manages cookies across requests, making it easy to test authenticated flows and session-based APIs.

## How It Works

### Automatic Cookie Management

Cookies are **automatically captured and reused** across all requests in a scenario:

1. **Response cookies are captured**: When a response includes `Set-Cookie` headers, they're stored
2. **Cookies are sent automatically**: Subsequent requests automatically include relevant cookies
3. **Cookies persist**: Cookies remain available throughout the entire scenario execution

### No Configuration Required

You don't need to do anything special - cookies just work!

```httpx
request login {
  curl https://api.example.com/login -d '{"user":"test"}'
  # Response sets cookies automatically
}

request get_profile {
  curl https://api.example.com/profile
  # Cookies from login are sent automatically!
}

scenario test {
  load 5 vus for 30s
  run login -> get_profile  # Cookies flow automatically
}
```

## Cookie Extraction

If you need to **explicitly extract** cookie values for other purposes:

```httpx
request login {
  curl https://api.example.com/login -d '...'

  extract {
    session_id = cookie:SESSIONID
    user_token = cookie:USER_TOKEN
    csrf_token = cookie:CSRF-TOKEN
  }

  assert status == 200
}

# Later use extracted values
request api_call {
  curl https://api.example.com/api \
    -H "X-CSRF-Token: ${csrf_token}"
  # Note: The cookie is ALSO sent automatically in Cookie header
}
```

## Common Patterns

### Pattern 1: Login Flow

```httpx
var base = "https://api.example.com"

request login {
  curl -X POST ${base}/login \
    -d '{"email":"user@test.com","password":"secret"}'

  # Cookies set automatically (session, auth tokens, etc.)
  assert status == 200
}

request get_dashboard {
  curl ${base}/dashboard
  # Cookies from login sent automatically
  assert status == 200
}

request update_profile {
  curl -X PATCH ${base}/profile \
    -d '{"name":"Updated"}'
  # Still using cookies from login
  assert status == 200
}

scenario user_flow {
  load 10 vus for 5m
  run login -> get_dashboard -> update_profile
}
```

### Pattern 2: Session-Based API

```httpx
var api = "https://api.example.com"

request start_session {
  curl ${api}/session/start
  # Response: Set-Cookie: session_id=abc123
  extract session_id = cookie:session_id
}

request upload_data {
  curl -X POST ${api}/upload \
    -F "file=@data.csv"
  # Cookie: session_id=abc123 (sent automatically)
}

request process_data {
  curl -X POST ${api}/process
  # Cookie: session_id=abc123 (still active)
}

request get_results {
  curl ${api}/results
  # Cookie: session_id=abc123 (persists)
}

scenario data_processing {
  load 5 vus for 2m
  run start_session -> upload_data -> process_data -> get_results
}
```

### Pattern 3: OAuth/Token + Cookies

```httpx
var base = "https://api.example.com"

request oauth_login {
  curl -X POST ${base}/oauth/token \
    -d "grant_type=password&username=test&password=secret"

  extract {
    access_token = $.access_token
    session_cookie = cookie:SESSION
  }

  assert status == 200
}

request api_call_with_both {
  # Uses both: Bearer token AND cookies
  curl ${base}/api/data \
    -H "Authorization: Bearer ${access_token}"
  # Cookie header added automatically with session_cookie
}

scenario oauth_flow {
  load 5 vus for 1m
  run oauth_login -> api_call_with_both
}
```

### Pattern 4: CSRF Protection

```httpx
var base = "https://api.example.com"

request get_csrf_token {
  curl ${base}/csrf-token
  # Sets cookie: XSRF-TOKEN=token123

  extract {
    csrf = cookie:XSRF-TOKEN
  }
}

request submit_form {
  curl -X POST ${base}/submit \
    -H "X-XSRF-TOKEN: ${csrf}" \
    -d '{"data":"value"}'
  # Cookie automatically sent: XSRF-TOKEN=token123
}

scenario csrf_protected {
  load 3 vus for 30s
  run get_csrf_token -> submit_form
}
```

## Cookie Scope

### Per-VU Cookie Jar

Each Virtual User (VU) has its own cookie jar:

```httpx
scenario multi_user {
  load 10 vus for 5m

  # VU #1 gets its own cookies
  # VU #2 gets its own cookies
  # ... cookies don't mix between VUs
  run login -> get_data
}
```

This simulates real users - each has their own session.

### Shared Across Requests

Within a VU, cookies are shared across all requests:

```httpx
run login {           # Sets cookies
  run get_profile {   # Uses login cookies
    run update_data { # Uses login cookies
      run logout      # Uses login cookies
    }
  }
}
```

## Cookie Extraction Rules

### Extract from Set-Cookie Header

```httpx
extract {
  session = cookie:sessionid
  user = cookie:user_id
  token = cookie:auth_token
}
```

### Cookie Name Matching

- Case-sensitive
- Exact match required
- First cookie with matching name is extracted

## Advanced Usage

### Conditional Based on Cookie

```httpx
request check_auth {
  curl ${base}/status

  extract {
    has_session = cookie:SESSION
  }
}

request login {
  curl ${base}/login -d '...'
}

request get_data {
  curl ${base}/data
}

scenario smart_flow {
  load 5 vus for 1m

  run check_auth
  if ${has_session} == "" {
    # No session, need to login
    run login
  }
  run get_data
}
```

### Explicit Cookie Setting

You can also set cookies manually in curl commands:

```httpx
request with_manual_cookie {
  curl ${base}/api \
    -b "manual_cookie=value123; another=value456"

  # This adds to automatically managed cookies
}
```

## Debugging Cookies

### View Cookies in Response

```bash
# Set VERBOSE to see all headers (including Set-Cookie)
VERBOSE=1 ./bin/httptool scenario run scenario.httpx
```

### Extract and Log Cookies

```httpx
request login {
  curl ${base}/login -d '...'

  extract {
    session = cookie:SESSION
    user_id = cookie:USER_ID
  }

  # These will be in the logs
  assert status == 200
}
```

## Cookie Lifecycle

```
1. First Request
   └─> Response includes: Set-Cookie: session=abc123
       └─> Cookie stored in jar

2. Second Request
   └─> Cookie: session=abc123 (added automatically)
       └─> Response may update: Set-Cookie: session=abc123; updated
           └─> Cookie updated in jar

3. Third Request
   └─> Cookie: session=abc123 (uses updated value)
       └─> Continues throughout scenario

4. Scenario Ends
   └─> Cookie jar cleared
```

## Important Notes

✅ **Automatic**: Cookies work automatically, no configuration needed
✅ **Per-VU**: Each virtual user has isolated cookies
✅ **Persistent**: Cookies last entire scenario duration
✅ **Standard HTTP**: Follows standard cookie behavior
✅ **Secure**: Cookies scoped to appropriate domains

❌ **Not Persistent**: Cookies don't persist between scenario runs
❌ **No Cookie File**: Doesn't save cookies to disk
❌ **No Manual Jar Management**: Jar is automatic only

## Examples

See:
- `examples/scenarios/with-cookies.httpx` - Complete cookie example
- `examples/scenarios/user-journey.httpx` - Realistic flow with cookies
- `TEMPLATE-YOUR-CURLS.httpx` - Template with cookie support

## Summary

**Cookies just work!** 🍪

1. Response sets cookies → Automatically stored
2. Next request → Cookies sent automatically
3. No configuration needed
4. Extract explicitly only if needed for other purposes

Perfect for:
- Session-based authentication
- Login flows
- CSRF protection
- OAuth flows with cookies
- Any session-based API testing
