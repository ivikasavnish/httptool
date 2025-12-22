#!/usr/bin/env python3
"""
Python Evaluator - Reference Implementation

This evaluator reads EvaluationContext from stdin and writes
EvaluatorDecision to stdout as JSON.

Contract:
- Input: EvaluationContext JSON via stdin
- Output: EvaluatorDecision JSON via stdout
- Errors: Write to stderr and exit with non-zero code
"""

import json
import sys
from typing import Any, Dict, Optional


def evaluate(ir: Dict, request: Dict, response: Dict, vars: Dict) -> Dict:
    """
    Evaluation logic - customize this based on your needs
    """
    decision = {
        "decision": "pass",
        "reason": "default pass",
        "mutations": {},
        "actions": {},
        "metadata": {}
    }

    # Handle errors
    if response.get("error"):
        decision["decision"] = "fail"
        decision["reason"] = f"Request failed: {response['error']}"
        return decision

    status = response.get("status", 0)

    # Handle HTTP errors
    if status >= 500:
        decision["decision"] = "retry"
        decision["reason"] = f"Server error: {status}"
        decision["actions"]["retry_after_ms"] = 1000  # 1 second
        decision["actions"]["max_retries"] = 3
        return decision

    if status >= 400:
        decision["decision"] = "fail"
        decision["reason"] = f"Client error: {status}"
        return decision

    # Handle slow responses
    latency = response.get("latency_ms", 0)
    if latency > 5000:
        decision["metadata"]["slow_request"] = True
        decision["reason"] = f"Slow response: {latency}ms"

    # Example: Extract data from response
    body = response.get("body")
    if body and isinstance(body, dict):
        if "token" in body:
            decision["mutations"]["vars"] = {
                "auth_token": body["token"]
            }

        if "user_id" in body:
            if "vars" not in decision["mutations"]:
                decision["mutations"]["vars"] = {}
            decision["mutations"]["vars"]["user_id"] = body["user_id"]

    # Example: Conditional retry logic
    attempt = vars.get("attempt", 1)
    if status == 429 and attempt < 3:
        decision["decision"] = "retry"
        decision["reason"] = "Rate limited"

        # Exponential backoff
        retry_after = response.get("headers", {}).get("retry-after")
        if retry_after:
            decision["actions"]["retry_after_ms"] = int(retry_after) * 1000
        else:
            decision["actions"]["retry_after_ms"] = (2 ** attempt) * 1000

        return decision

    # Example: Content validation
    if body and ir.get("metadata", {}).get("tags", {}).get("validate_content"):
        if isinstance(body, str) and "error" in body:
            decision["decision"] = "fail"
            decision["reason"] = "Response contains error marker"
            return decision

    return decision


def main():
    try:
        # Read input from stdin
        input_data = sys.stdin.read()
        context = json.loads(input_data)

        # Extract key data
        ir = context.get("ir", {})
        request = context.get("request", {})
        response = context.get("response", {})
        vars = context.get("vars", {})

        # Evaluate
        decision = evaluate(ir, request, response, vars)

        # Write decision to stdout
        print(json.dumps(decision, indent=2))
        sys.exit(0)

    except Exception as error:
        print(f"Evaluator error: {error}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
