# Progress Tracking & Parallel Execution Demo

## Summary of Enhancements

HTTPTool now includes comprehensive progress tracking and parallel execution capabilities for load testing scenarios.

## New Features Implemented

### 1. Real-Time Progress Tracking (`--progress`)

Track execution progress in real-time:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress
```

**Features:**
- Live request counter updates every 2 seconds
- Error tracking
- Active VU (Virtual User) count
- Non-blocking progress updates
- Final summary on completion

**Example Output:**
```
🔄 Progress: 450 requests | 12 errors | 10 active VUs
✓ Completed: 6000 requests | 150 errors
```

### 2. Detailed Verbose Logging (`--verbose`)

Get detailed per-request information:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --verbose
```

**Features:**
- Timestamps for each event
- VU lifecycle events (start, iteration, completion)
- Request-level details (method, URL, status code, latency)
- Success ✓ and failure ✗ indicators
- Per-VU performance statistics

**Example Output:**
```
[12:57:07] VU 1 started
[12:57:07] VU 1 → iteration 1
[12:57:07] VU 1 ✓ GET https://api.example.com/users - 200 (145ms)
[12:57:07] VU 1 ✗ GET https://api.example.com/error - 401 (200ms)
[12:57:07] VU 1 → iteration 2
```

### 3. Combined Progress & Verbose Mode

Maximum visibility into test execution:

```bash
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress --verbose
```

Provides both detailed logging and periodic progress summaries.

### 4. Automatic Parallel Execution

HTTPTool automatically runs Virtual Users in parallel using Go goroutines:

- **VU-based load**: Each VU runs in its own goroutine
- **RPS-based load**: Requests distributed across parallel workers
- **Iteration-based load**: Iterations spread across multiple VUs
- **Thread-safe**: All result collection is synchronized
- **Cookie sharing**: Shared cookie jar across VUs for session management

### 5. Enhanced Per-VU Statistics (Verbose Mode)

When using `--verbose`, get detailed per-VU breakdown:

```
Per-VU Results:
  VU 1: 127 iterations, 127 requests (✓ 120, ✗ 7), avg latency: 68ms
  VU 2: 125 iterations, 125 requests (✓ 118, ✗ 7), avg latency: 71ms
  VU 3: 128 iterations, 128 requests (✓ 121, ✗ 7), avg latency: 67ms
  ...
```

## Usage Examples

### Example 1: Basic Progress Tracking

```bash
# Run voter outreach scenario with progress updates
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress
```

Output shows periodic progress updates while running 10 VUs for 1 minute.

### Example 2: Debugging with Verbose Mode

```bash
# Run with detailed logging to see every request
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --verbose
```

Shows timestamp, VU ID, status, and latency for each request.

### Example 3: Full Monitoring

```bash
# Combine both for complete visibility
./bin/httptool scenario run examples/scenarios/voter_outreach_user.httpx --progress --verbose
```

Get detailed request logs plus periodic progress summaries.

### Example 4: RPS Load Testing

```bash
# Run 100 requests per second with progress
./bin/httptool scenario run examples/scenarios/simple-load.httpx --progress
```

Maintains consistent RPS rate with automatic VU distribution.

## Parallel Execution Details

### How It Works

1. **VU Mode** (`load 10 vus for 1m`):
   - Starts 10 goroutines simultaneously
   - Each VU runs iterations continuously for 1 minute
   - Independent execution per VU

2. **RPS Mode** (`load 100 rps for 30s`):
   - Uses ticker to maintain consistent rate
   - Dynamically spawns goroutines for each request
   - Automatically tracks which VU handled each request

3. **Iteration Mode** (`load 1000 iterations with 5 vus`):
   - Distributes 1000 iterations across 5 VUs
   - Each VU executes ~200 iterations
   - Runs until all iterations complete

### Performance Characteristics

- **Zero blocking**: Progress updates don't slow down execution
- **Buffered channel**: 1000 event buffer prevents message loss
- **Concurrent-safe**: Mutex-protected result aggregation
- **Resource efficient**: Progress updates batched every 2 seconds

## Technical Implementation

### Progress Update Architecture

```go
type ProgressUpdate struct {
    Type        string // "vu_start", "iteration_start", "request", "vu_done"
    VUID        int
    Iteration   int
    RequestName string
    Status      int
    Latency     time.Duration
    Error       string
    Timestamp   time.Time
}
```

Updates are sent via non-blocking channel from executor goroutines to a dedicated progress printer goroutine.

### Parallel Execution Pattern

Each VU runs in a goroutine:

```go
for vu := 1; vu <= vus; vu++ {
    wg.Add(1)
    go func(vuID int) {
        defer wg.Done()
        // Execute iterations
        for time.Now().Before(deadline) {
            executeIteration(vuID, iteration)
            iteration++
        }
    }(vu)
}
wg.Wait()
```

## Files Modified

1. **pkg/scenario/executor.go**
   - Added `ProgressUpdate` struct
   - Added `EnableProgress()` method
   - Added `sendProgress()` helper
   - Instrumented VU execution with progress updates

2. **cmd/httptool/scenario.go**
   - Added `--progress` and `--verbose` flags
   - Added `printProgress()` function
   - Enhanced `printScenarioResults()` with per-VU stats
   - Updated CLI usage documentation

3. **pkg/scenario/parser.go**
   - Added RPS shorthand syntax support: `load 100 rps for 30s`

4. **examples/scenarios/voter_outreach_user.httpx**
   - Fixed scenario to actually run the request

## Documentation

- **docs/progress-tracking.md**: Complete guide to progress tracking and parallel execution
- **PROGRESS-DEMO.md**: This file - summary of features and usage

## Benefits

1. **Visibility**: See what's happening during long-running tests
2. **Debugging**: Identify failing requests immediately with verbose mode
3. **Performance**: Parallel execution maximizes throughput
4. **Monitoring**: Track active VUs and error rates in real-time
5. **Analysis**: Per-VU statistics help identify bottlenecks
6. **Flexibility**: Choose level of detail based on needs

## Next Steps

Users can now:
- Run load tests with real-time monitoring
- Debug failing scenarios with verbose logging
- Analyze per-VU performance characteristics
- Scale tests with automatic parallel execution
- Monitor progress of long-running tests

All features work seamlessly with existing scenario files!
