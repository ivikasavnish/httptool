#!/usr/bin/env bun
/**
 * Custom Evaluator Example - Advanced Logic
 *
 * This demonstrates:
 * - Custom retry logic with exponential backoff
 * - Data extraction from responses
 * - Conditional branching
 * - Variable mutation for subsequent requests
 */

async function main() {
  const input = await Bun.stdin.text();
  const context = JSON.parse(input);

  const { ir, request, response, vars } = context;

  // Initialize decision
  const decision = {
    decision: "pass",
    reason: "",
    mutations: {
      headers: {},
      vars: {}
    },
    actions: {},
    metadata: {}
  };

  // Track performance
  decision.metadata.latency_ms = response.latency_ms;
  decision.metadata.size_bytes = response.size_bytes;

  // Handle network errors
  if (response.error) {
    const attempt = vars.attempt || 1;
    if (attempt < 3) {
      decision.decision = "retry";
      decision.reason = `Network error: ${response.error}`;
      decision.actions.retry_after_ms = Math.pow(2, attempt) * 1000;
      decision.actions.max_retries = 3;
    } else {
      decision.decision = "fail";
      decision.reason = `Network error after ${attempt} attempts: ${response.error}`;
    }
    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Authentication flow
  if (response.status === 401) {
    // Check if we have a refresh token
    if (vars.refresh_token) {
      decision.decision = "branch";
      decision.reason = "Unauthorized - refreshing token";
      decision.actions.goto = "refresh_auth";
      decision.mutations.vars.original_request = ir;
    } else {
      decision.decision = "fail";
      decision.reason = "Unauthorized and no refresh token available";
    }
    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Rate limiting with intelligent backoff
  if (response.status === 429) {
    const retryAfter = response.headers["retry-after"] || response.headers["x-ratelimit-reset"];
    const attempt = vars.attempt || 1;

    if (attempt < 5) {
      decision.decision = "retry";
      decision.reason = "Rate limited";

      if (retryAfter) {
        // Use server-provided retry time
        const waitTime = parseInt(retryAfter);
        decision.actions.retry_after_ms = isNaN(waitTime) ? 5000 : waitTime * 1000;
      } else {
        // Exponential backoff: 2s, 4s, 8s, 16s
        decision.actions.retry_after_ms = Math.pow(2, attempt) * 1000;
      }

      decision.metadata.backoff_strategy = "exponential";
    } else {
      decision.decision = "fail";
      decision.reason = `Rate limited after ${attempt} attempts`;
    }
    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Server errors with retry
  if (response.status >= 500) {
    const attempt = vars.attempt || 1;

    if (attempt < 3) {
      decision.decision = "retry";
      decision.reason = `Server error ${response.status}`;
      decision.actions.retry_after_ms = 2000; // Fixed 2s for server errors
    } else {
      decision.decision = "fail";
      decision.reason = `Server error ${response.status} after ${attempt} attempts`;
    }
    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Client errors (non-retryable)
  if (response.status >= 400) {
    decision.decision = "fail";
    decision.reason = `Client error: ${response.status}`;
    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Success - extract useful data
  if (response.status >= 200 && response.status < 300) {
    decision.decision = "pass";
    decision.reason = `Success: ${response.status}`;

    // Extract authentication tokens
    if (response.body && typeof response.body === 'object') {
      if (response.body.access_token) {
        decision.mutations.vars.access_token = response.body.access_token;
        decision.mutations.headers["Authorization"] = `Bearer ${response.body.access_token}`;
      }

      if (response.body.refresh_token) {
        decision.mutations.vars.refresh_token = response.body.refresh_token;
      }

      // Extract pagination info
      if (response.body.next_page) {
        decision.mutations.vars.next_page = response.body.next_page;
      }

      // Extract user info
      if (response.body.user) {
        decision.mutations.vars.user_id = response.body.user.id;
        decision.mutations.vars.user_email = response.body.user.email;
      }

      // Extract IDs for subsequent requests
      if (response.body.id) {
        decision.mutations.vars.created_id = response.body.id;
      }
    }

    // Performance warnings
    if (response.latency_ms > 1000) {
      decision.metadata.performance_warning = "Slow response";
    }

    // Size warnings
    if (response.size_bytes > 1024 * 1024) { // 1MB
      decision.metadata.size_warning = "Large response";
    }

    console.log(JSON.stringify(decision, null, 2));
    return;
  }

  // Unexpected status
  decision.decision = "fail";
  decision.reason = `Unexpected status: ${response.status}`;
  console.log(JSON.stringify(decision, null, 2));
}

main();
