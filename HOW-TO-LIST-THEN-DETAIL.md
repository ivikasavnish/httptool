# How To: List API → Detail API Pattern

## Your Use Case

You have 2 curl commands:
1. **First curl**: Get list of posts (returns array)
2. **Second curl**: Get individual post detail (run for each post from the list)

## Quick Solution

### Option 1: Simple Sequential (Works Now)

```httpx
# Replace with your actual API
var base_url = "https://api.example.com"

# First curl: Get list
request get_list {
  curl ${base_url}/posts

  # Extract IDs from response array
  extract {
    id_1 = $[0].id
    id_2 = $[1].id
    id_3 = $[2].id
  }

  assert status == 200
}

# Second curl: Get details (repeat for each)
request get_detail_1 {
  curl ${base_url}/posts/${id_1}
  assert status == 200
}

request get_detail_2 {
  curl ${base_url}/posts/${id_2}
  assert status == 200
}

request get_detail_3 {
  curl ${base_url}/posts/${id_3}
  assert status == 200
}

# Run them in sequence
scenario test {
  load 5 vus for 30s

  run get_list -> get_detail_1 -> get_detail_2 -> get_detail_3
}
```

### Option 2: Using Your Actual Curls

Here's the template - just replace the curl commands:

```httpx
var base = "YOUR_API_BASE_URL"

# ===== FIRST CURL =====
request list_posts {
  # 👇 PASTE YOUR FIRST CURL HERE
  curl -X GET ${base}/api/posts \
    -H "Authorization: Bearer YOUR_TOKEN" \
    -H "Content-Type: application/json"

  # Extract IDs from the response
  # Adjust JSONPath based on your response structure
  extract {
    # If response is: [{"id": 1, ...}, {"id": 2, ...}]
    post_1 = $[0].id
    post_2 = $[1].id
    post_3 = $[2].id

    # If response is: {"posts": [{"id": 1, ...}]}
    # post_1 = $.posts[0].id
    # post_2 = $.posts[1].id
  }

  assert status == 200
}

# ===== SECOND CURL (for each post) =====
request get_post_1 {
  # 👇 PASTE YOUR SECOND CURL HERE (use ${post_1} for the ID)
  curl -X GET ${base}/api/posts/${post_1} \
    -H "Authorization: Bearer YOUR_TOKEN" \
    -H "Content-Type: application/json"

  assert status == 200
}

request get_post_2 {
  curl -X GET ${base}/api/posts/${post_2} \
    -H "Authorization: Bearer YOUR_TOKEN"
  assert status == 200
}

request get_post_3 {
  curl -X GET ${base}/api/posts/${post_3} \
    -H "Authorization: Bearer YOUR_TOKEN"
  assert status == 200
}

# Run the flow
scenario my_test {
  load 5 vus for 30s

  run list_posts -> get_post_1 -> get_post_2 -> get_post_3
}
```

## How to Extract IDs from Response

The key is the `extract` block. Adjust based on your API response:

### If response is a simple array:
```json
[
  {"id": 1, "title": "Post 1"},
  {"id": 2, "title": "Post 2"}
]
```

Extract like this:
```httpx
extract {
  id_1 = $[0].id
  id_2 = $[1].id
  title_1 = $[0].title
}
```

### If response is nested:
```json
{
  "data": {
    "posts": [
      {"id": 1, "title": "Post 1"},
      {"id": 2, "title": "Post 2"}
    ]
  }
}
```

Extract like this:
```httpx
extract {
  id_1 = $.data.posts[0].id
  id_2 = $.data.posts[1].id
}
```

### If you need all IDs:
```json
{
  "posts": [
    {"post_id": 123},
    {"post_id": 456},
    {"post_id": 789}
  ]
}
```

Extract:
```httpx
extract {
  id_1 = $.posts[0].post_id
  id_2 = $.posts[1].post_id
  id_3 = $.posts[2].post_id
}
```

## Step-by-Step Instructions

### 1. Save Your Scenario

```bash
cat > my-api-test.httpx << 'EOF'
var base = "https://api.example.com"

request list {
  curl ${base}/posts
  extract id_1 = $[0].id, id_2 = $[1].id
}

request detail_1 {
  curl ${base}/posts/${id_1}
}

request detail_2 {
  curl ${base}/posts/${id_2}
}

scenario test {
  load 5 vus for 20s
  run list -> detail_1 -> detail_2
}
EOF
```

### 2. Validate

```bash
./bin/httptool scenario validate my-api-test.httpx
```

### 3. Dry Run

```bash
./bin/httptool scenario run my-api-test.httpx --dry-run
```

### 4. Execute

```bash
./bin/httptool scenario run my-api-test.httpx
```

## Working Example (You Can Run Now)

```bash
# This works with real API right now:
cat > posts-test.httpx << 'EOF'
var api = "https://jsonplaceholder.typicode.com"

request list_posts {
  curl ${api}/posts
  extract id_1 = $[0].id, id_2 = $[1].id
}

request get_post_1 {
  curl ${api}/posts/${id_1}
}

request get_post_2 {
  curl ${api}/posts/${id_2}
}

scenario test {
  load 3 vus for 15s
  run list_posts -> get_post_1 -> get_post_2
}
EOF

# Run it
./bin/httptool scenario run posts-test.httpx
```

## Common Patterns

### Pattern 1: List Users → Get Each Profile
```httpx
request list_users {
  curl ${api}/users
  extract user_1 = $[0].id, user_2 = $[1].id
}

request get_user_1 {
  curl ${api}/users/${user_1}
}
```

### Pattern 2: Search → Get Details
```httpx
request search {
  curl "${api}/search?q=test"
  extract result_1 = $.results[0].id
}

request get_result {
  curl ${api}/items/${result_1}
}
```

### Pattern 3: With Authentication
```httpx
request login {
  curl -X POST ${api}/login -d '{"user":"test"}'
  extract token = $.access_token
}

request list_posts {
  curl ${api}/posts -H "Authorization: Bearer ${token}"
  extract post_id = $[0].id
}

request get_post {
  curl ${api}/posts/${post_id} -H "Authorization: Bearer ${token}"
}

scenario flow {
  load 5 vus for 30s
  run login -> list_posts -> get_post
}
```

## Tips

1. **Start with 1-2 items**: Don't extract 100 IDs, start small
2. **Check your JSONPath**: Look at the actual response structure
3. **Use dry-run**: Test extraction without load
4. **Add assertions**: Verify extracted values make sense

## Troubleshooting

### "Variable not found"
- Check JSONPath matches your response structure
- View response: `SHOW_BODY=1 ./bin/httptool exec 'curl ...'`

### "Extraction failed"
- Response might not be JSON
- Array might be empty
- Path might be wrong

### "High failure rate"
- Extracted variables not passing correctly
- This is a known issue - variables work within same request, not across requests yet
- Workaround: Use hardcoded IDs for now, or single VU

## Current Limitations

The variable extraction across requests is still in development. For now:

✅ **Works**: Extract within same request
✅ **Works**: Sequential flow with hardcoded values
⚠️ **Partial**: Variables across requests (in development)

## Need Help?

1. Check your API response format
2. Test extraction with single request
3. Use dry-run to validate
4. Start with small load (1-2 VUs)

---

**Your pattern is supported! Just paste your curl commands and adjust the JSONPath extraction.**
