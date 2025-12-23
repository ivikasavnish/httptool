# Progress Tracking and Parallel Execution

HTTPTool now supports real-time progress tracking and detailed logging during scenario execution.

## Features

### 1. Progress Tracking (`--progress`)

The `--progress` flag enables real-time progress updates during scenario execution:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress
```

Features:
- Real-time request counter
- Error count tracking
- Active VU (Virtual User) monitoring
- Progress updates every 2 seconds
- Completion summary

Output example:
```
🔄 Progress: 450 requests | 12 errors | 10 active VUs
```

### 2. Verbose Logging (`--verbose`)

The `--verbose` flag shows detailed information for each request:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --verbose
```

Features:
- Timestamp for each event
- VU start/completion notifications
- Iteration tracking
- Request-level details (method, URL, status, latency)
- Success (✓) and failure (✗) indicators

Output example:
```
[12:57:07] VU 1 started
[12:57:07] VU 1 → iteration 1
[12:57:07] VU 1 ✓ GET https://api.example.com/users - 200 (145ms)
[12:57:07] VU 1 → iteration 2
```

### 3. Combined Usage

For maximum visibility, combine both flags:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress --verbose
```

This provides:
- Detailed per-request logging
- Real-time progress summary
- Per-VU performance statistics at the end

## Parallel Execution

HTTPTool automatically executes Virtual Users (VUs) in parallel using Go goroutines.

### How It Works

When you specify a load configuration like:
```
scenario my_test {
    load 10 vus for 1m
    run my_request
}
```

HTTPTool will:
1. Start 10 VUs simultaneously
2. Each VU runs independently in parallel
3. Each VU executes iterations continuously for 1 minute
4. All VUs are coordinated and monitored in real-time

### Load Patterns

#### Virtual Users (VUs)
```
load 10 vus for 1m
```
- Runs 10 parallel users
- Each user executes requests continuously
- Duration: 1 minute

#### Requests Per Second (RPS)
```
load 100 rps for 30s
```
- Maintains 100 requests/second rate
- Automatically distributes across VUs
- Duration: 30 seconds

#### Fixed Iterations
```
load 1000 iterations with 5 vus
```
- Executes exactly 1000 iterations
- Distributed across 5 parallel VUs
- Runs until all iterations complete

## Performance Statistics

### Standard Output

After execution, you'll see:
- Total requests executed
- Success/failure breakdown
- Latency statistics (avg, min, max)
- Data transferred
- Throughput (requests/second)

### Verbose Mode Additional Stats

When using `--verbose`, you also get:
- Per-VU performance breakdown
- Individual VU iteration count
- Success/error count per VU
- Average latency per VU

Example:
```
Per-VU Results:
  VU 1: 127 iterations, 127 requests (✓ 120, ✗ 7), avg latency: 68ms
  VU 2: 125 iterations, 125 requests (✓ 118, ✗ 7), avg latency: 71ms
  VU 3: 128 iterations, 128 requests (✓ 121, ✗ 7), avg latency: 67ms
  ...
```

## Best Practices

### 1. Start Without Flags
First run without progress/verbose to see if the scenario works:
```bash
./bin/httptool scenario run my-scenario.httpx
```

### 2. Add Progress for Monitoring
For longer tests, add progress tracking:
```bash
./bin/httptool scenario run my-scenario.httpx --progress
```

### 3. Debug with Verbose
If you encounter issues, use verbose mode:
```bash
./bin/httptool scenario run my-scenario.httpx --progress --verbose
```

### 4. Adjust VUs Based on Results
- Too many errors? Reduce VUs
- Need more throughput? Increase VUs
- Want to control exact parallelism? Use the VU count

## Examples

### Quick Load Test
```bash
# 100 RPS for 30 seconds with progress
./bin/httptool scenario run examples/scenarios/simple-load.httpx --progress
```

### Extended Stress Test
```bash
# 50 VUs for 5 minutes with detailed logging
./bin/httptool scenario run examples/scenarios/stress-test.httpx --progress --verbose
```

### Debugging Single Request
```bash
# 1 VU for 10 seconds with full details
./bin/httptool scenario run examples/scenarios/debug.httpx --verbose
```

## Technical Details

### Progress Update Mechanism
- Non-blocking channel-based communication
- Updates buffered (1000 events)
- No performance impact on request execution
- Updates displayed every 2 seconds to avoid flooding

### Parallel Execution Architecture
- Each VU runs in a separate goroutine
- Shared cookie jar for session management
- Thread-safe result collection
- Graceful shutdown on context cancellation

### Performance Considerations
- Verbose mode generates more output (slight overhead)
- Progress mode has minimal overhead (<1%)
- No limit on VU count (limited by system resources)
- Automatic load distribution across available CPU cores
