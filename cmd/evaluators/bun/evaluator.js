#!/usr/bin/env bun

/**
 * Bun JavaScript Evaluator - Reference Implementation
 *
 * This evaluator reads EvaluationContext from stdin and writes
 * EvaluatorDecision to stdout as JSON.
 *
 * Contract:
 * - Input: EvaluationContext JSON via stdin
 * - Output: EvaluatorDecision JSON via stdout
 * - Errors: Write to stderr and exit with non-zero code
 */

async function main() {
  try {
    // Read input from stdin
    const input = await Bun.stdin.text();
    const context = JSON.parse(input);

    // Extract key data
    const { ir, request, response, vars } = context;

    // Example evaluation logic
    const decision = evaluate(ir, request, response, vars);

    // Write decision to stdout
    console.log(JSON.stringify(decision, null, 2));
    process.exit(0);
  } catch (error) {
    console.error(`Evaluator error: ${error.message}`);
    process.exit(1);
  }
}

/**
 * Evaluation logic - customize this based on your needs
 */
function evaluate(ir, request, response, vars) {
  const decision = {
    decision: "pass",
    reason: "default pass",
    mutations: {},
    actions: {},
    metadata: {}
  };

  // Handle errors
  if (response.error) {
    decision.decision = "fail";
    decision.reason = `Request failed: ${response.error}`;
    return decision;
  }

  // Handle HTTP errors
  if (response.status >= 500) {
    decision.decision = "retry";
    decision.reason = `Server error: ${response.status}`;
    decision.actions.retry_after_ms = 1000; // 1 second
    decision.actions.max_retries = 3;
    return decision;
  }

  if (response.status >= 400) {
    decision.decision = "fail";
    decision.reason = `Client error: ${response.status}`;
    return decision;
  }

  // Handle slow responses
  if (response.latency_ms > 5000) {
    decision.metadata.slow_request = true;
    decision.reason = `Slow response: ${response.latency_ms}ms`;
  }

  // Example: Extract data from response
  if (response.body && typeof response.body === 'object') {
    if (response.body.token) {
      decision.mutations.vars = {
        auth_token: response.body.token
      };
    }

    if (response.body.user_id) {
      decision.mutations.vars = decision.mutations.vars || {};
      decision.mutations.vars.user_id = response.body.user_id;
    }
  }

  // Example: Conditional retry logic
  const attempt = vars.attempt || 1;
  if (response.status === 429 && attempt < 3) {
    decision.decision = "retry";
    decision.reason = "Rate limited";

    // Exponential backoff
    const retryAfter = response.headers["retry-after"];
    if (retryAfter) {
      decision.actions.retry_after_ms = parseInt(retryAfter) * 1000;
    } else {
      decision.actions.retry_after_ms = Math.pow(2, attempt) * 1000;
    }

    return decision;
  }

  // Example: Content validation
  if (response.body && ir.metadata?.tags?.validate_content) {
    if (typeof response.body === 'string' && response.body.includes('error')) {
      decision.decision = "fail";
      decision.reason = "Response contains error marker";
      return decision;
    }
  }

  return decision;
}

// Run main
main();
