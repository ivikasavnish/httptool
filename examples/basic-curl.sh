#!/bin/bash
# Basic examples of using httptool with curl commands

echo "=== Example 1: Simple GET request ==="
httptool exec 'curl https://httpbin.org/get'

echo ""
echo "=== Example 2: POST with JSON data ==="
httptool exec 'curl -X POST https://httpbin.org/post \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"test\",\"value\":123}"'

echo ""
echo "=== Example 3: Convert curl to IR ==="
httptool convert 'curl -X POST https://httpbin.org/post \
  -H "Authorization: Bearer token123" \
  -H "Content-Type: application/json" \
  -d "{\"action\":\"create\"}"' > /tmp/request.json

echo "Saved to /tmp/request.json"
cat /tmp/request.json

echo ""
echo "=== Example 4: Execute from IR file ==="
httptool run /tmp/request.json

echo ""
echo "=== Example 5: With authentication ==="
httptool exec 'curl -u username:password https://httpbin.org/basic-auth/username/password'

echo ""
echo "=== Example 6: With custom headers ==="
httptool exec 'curl https://httpbin.org/headers \
  -H "X-Custom-Header: value1" \
  -H "X-Another-Header: value2"'

echo ""
echo "=== Example 7: Form data ==="
httptool exec 'curl -X POST https://httpbin.org/post \
  -d "field1=value1&field2=value2"'

echo ""
echo "=== Example 8: With cookies ==="
httptool exec 'curl https://httpbin.org/cookies \
  -b "session=abc123; user_id=456"'

echo ""
echo "=== Example 9: Insecure TLS ==="
httptool exec 'curl -k https://self-signed.badssl.com/'

echo ""
echo "=== Example 10: With timeout ==="
httptool exec 'curl -m 5 https://httpbin.org/delay/2'
