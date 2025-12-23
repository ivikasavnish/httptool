# 🍪 Cookie Support

## NEW FEATURE: Automatic Cookie Management!

Cookies are now **automatically retained and reused** across all requests in your scenarios!

## How It Works

### Zero Configuration Required

```httpx
request login {
  curl https://api.example.com/login -d '{"user":"test"}'
  # ✅ Cookies from response are automatically stored
}

request get_data {
  curl https://api.example.com/data
  # ✅ Cookies from login are automatically sent!
}

scenario test {
  load 5 vus for 30s
  run login -> get_data  # Cookies flow automatically
}
```

That's it! No setup, no configuration - cookies just work.

## What This Enables

### ✅ Session-Based Authentication
```httpx
run login -> get_profile -> update_settings
# Session cookie flows through automatically
```

### ✅ OAuth with Cookies
```httpx
request oauth {
  curl -X POST /oauth/token -d '...'
  extract token = $.access_token
  # OAuth session cookie stored automatically
}

request api_call {
  curl /api -H "Authorization: Bearer ${token}"
  # Cookie sent automatically too!
}
```

### ✅ CSRF Protection
```httpx
request get_token {
  curl /csrf
  extract csrf = cookie:CSRF-TOKEN
  # Cookie stored, but also extracted for header
}

request submit {
  curl -X POST /form \
    -H "X-CSRF-TOKEN: ${csrf}"
  # CSRF cookie sent automatically
}
```

### ✅ Multi-Step Flows
```httpx
run step1 {         # Sets cookies
  run step2 {       # Uses step1 cookies
    run step3 {     # Uses all previous cookies
      run step4     # Cookies persist throughout
    }
  }
}
```

## Cookie Extraction (Optional)

If you need the cookie **value** explicitly:

```httpx
request login {
  curl /login -d '...'

  extract {
    session = cookie:SESSION_ID
    user = cookie:USER_ID
  }

  # Cookies are stored automatically AND extracted to variables
}

# Use extracted value in headers or elsewhere
request api {
  curl /api -H "X-Session: ${session}"
}
```

## Per-VU Isolation

Each virtual user has its own cookie jar:

```httpx
scenario test {
  load 10 vus for 5m
  run login -> get_data
  # VU #1 has its own cookies
  # VU #2 has its own cookies
  # They don't interfere
}
```

## Example Scenarios

### Login → Profile → Update
```httpx
var base = "https://api.example.com"

request login {
  curl -X POST ${base}/login -d '...'
}

request profile {
  curl ${base}/profile
}

request update {
  curl -X PATCH ${base}/profile -d '...'
}

scenario user_flow {
  load 10 vus for 5m
  run login -> profile -> update
  # Cookies flow through all 3 requests
}
```

### E-commerce Flow
```httpx
request browse {
  curl /products
}

request add_to_cart {
  curl -X POST /cart -d '{"product_id":123}'
  # Cart cookie set
}

request checkout {
  curl -X POST /checkout -d '...'
  # Cart cookie used
}

scenario shopping {
  load 20 vus for 10m
  run browse -> add_to_cart -> checkout
}
```

## See Also

- **Full Documentation**: `docs/cookies.md`
- **Example Scenario**: `examples/scenarios/with-cookies.httpx`
- **Template**: `TEMPLATE-YOUR-CURLS.httpx` (supports cookies)

## Summary

**Cookies just work! 🍪**

- ✅ Automatic capture from responses
- ✅ Automatic sending in requests
- ✅ Per-VU isolation
- ✅ Persistent throughout scenario
- ✅ Extract explicitly only if needed

No configuration, no hassle - perfect for session-based APIs!
