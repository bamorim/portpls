# Architecture

This document describes the internal architecture and implementation details of `portpls`.

## Project Structure

```
portpls/
├── cmd/
│   └── portpls/
│       └── main.go              # Entry point with urfave/cli app
├── internal/
│   ├── allocations/
│   │   └── allocations.go       # Allocation persistence and management
│   ├── config/
│   │   └── config.go            # Configuration struct and defaults
│   ├── port/
│   │   ├── checker.go           # Port availability checking
│   │   └── finder.go            # Port allocation algorithm
│   ├── process/
│   │   └── process.go           # Process information (PID, cwd, command)
│   ├── docker/
│   │   └── docker.go            # Docker container detection
│   └── logger/
│       └── logger.go            # Logging functionality
├── go.mod
├── go.sum
└── Makefile
```

## Technology Stack

- **Language:** Go
- **CLI Framework:** [urfave/cli](https://github.com/urfave/cli/tree/v2) v2
- **File locking:** `golang.org/x/sys/unix` flock
- **JSON handling:** Standard library `encoding/json`
- **Time parsing:** Support duration formats like "24h", "30d", "1h30m"

## Data Structures

### Config File

Location: `~/.config/portpls/config.json`

```json
{
  "port_start": 20000,
  "port_end": 22000,
  "freeze_period": "24h",
  "allocation_ttl": "0",
  "log_file": ""
}
```

**Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port_start` | integer | 20000 | Start of port range to allocate from |
| `port_end` | integer | 22000 | End of port range (inclusive) |
| `freeze_period` | duration | "24h" | Time period during which a port cannot be reallocated after release. Formats: "24h", "30m", "1d". "0" disables. |
| `allocation_ttl` | duration | "0" | Auto-expire allocations after this period of inactivity. "0" disables TTL. |
| `log_file` | string | "" | Path to log file. Empty string disables logging. |

### Allocations File

Location: `~/.local/share/portpls/allocations.json`

```json
{
  "version": 1,
  "last_issued_port": 3012,
  "allocations": {
    "3000": {
      "directory": "/home/user/code/project-a",
      "name": "main",
      "assigned_at": "2026-01-21T10:00:00Z",
      "last_used_at": "2026-01-21T14:00:00Z",
      "locked": true
    },
    "3010": {
      "directory": "/home/user/myproject",
      "name": "web",
      "assigned_at": "2026-01-21T10:00:00Z",
      "last_used_at": "2026-01-21T14:30:00Z",
      "locked": false
    }
  }
}
```

**Allocation fields:**

| Field | Type | Description |
|-------|------|-------------|
| `directory` | string | Absolute path to the directory that owns this port |
| `name` | string | Name of this allocation within the directory (default: "main") |
| `assigned_at` | ISO 8601 | When this port was first allocated |
| `last_used_at` | ISO 8601 | Last time this port was requested (for TTL calculation) |
| `locked` | boolean | Whether this port is locked (cannot be reallocated) |

## Port Allocation Algorithm

```
1. Load configuration from ~/.config/portpls/config.json
   - Create file with defaults if it doesn't exist

2. Load allocations from ~/.local/share/portpls/allocations.json
   - Create file with defaults if it doesn't exist

3. Get current directory absolute path (or use --directory flag)

4. Check if allocation exists for (directory, name):

   YES:
     a. Update last_used_at to current time
     b. Check if port is still free (attempt bind on 127.0.0.1:PORT)
        - If free: return port, save allocations
        - If taken: proceed to step 5 (allocate new port)

   NO:
     Proceed to step 5

5. Find free port:
   a. Start from last_issued_port + 1 (or port_start if not set)
   b. For each port in range:
      - Skip if port is in freeze period (assigned_at + freeze_period > now)
      - Skip if port is locked by another directory
      - Skip if port is already allocated to another (directory, name)
      - Attempt to bind to 127.0.0.1:PORT
      - If bind succeeds: port is free, go to step 6
      - If bind fails: port is busy, try next
   c. If reached port_end, wrap around to port_start
   d. If no free port found after full cycle: ERROR

6. Create allocation:
   - port: selected port
   - directory: current directory absolute path
   - name: from --name flag or "main"
   - assigned_at: current timestamp
   - last_used_at: current timestamp
   - locked: false

7. Update last_issued_port to selected port

8. Save allocations to file (atomic write: temp file + rename)

9. Output port to stdout

10. Exit 0
```

## Port Checking Logic

```go
func isPortFree(port int) bool {
    listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
    if err != nil {
        return false // Port is busy
    }
    listener.Close()
    return true // Port is free
}
```

## TTL and Cleanup

If `allocation_ttl` is set (non-zero):

1. On each run, before any operation:
   - Check all allocations
   - If `last_used_at + allocation_ttl < now`: remove allocation
   - Log removal if logging is enabled

2. The `get` command updates `last_used_at`, so actively used ports never expire

## File Locking

To prevent corruption from concurrent access:

1. Use file locking (flock on Unix) on allocations file
2. Lock before read-modify-write operations
3. Unlock after write completes
4. Handle lock timeout (e.g., 5 seconds) gracefully

## Logging

If `log_file` is configured, all allocation changes are logged.

**Log format:** Plain text, one event per line

```
2026-01-21T15:04:05Z ALLOC_ADD port=3001 dir=/home/user/project1 name=main
2026-01-21T15:04:10Z ALLOC_LOCK port=3001 locked=true
2026-01-21T15:05:00Z ALLOC_UPDATE port=3001 (reused)
2026-01-21T15:06:00Z ALLOC_DELETE port=3002 dir=/home/user/forgotten name=main
2026-01-21T15:07:00Z ALLOC_EXPIRE port=3003 dir=/home/user/old-project name=main ttl=30d
2026-01-21T15:08:00Z ALLOC_DELETE_ALL count=5
```

**Events:**
- `ALLOC_ADD` - new port allocated
- `ALLOC_UPDATE` - existing allocation reused (last_used_at updated)
- `ALLOC_LOCK` - port locked/unlocked
- `ALLOC_DELETE` - allocation removed (forget command)
- `ALLOC_DELETE_ALL` - all allocations removed (forget --all)
- `ALLOC_EXPIRE` - allocation expired by TTL

## Commands Implementation

Each command is implemented as a separate function that returns a `*cli.Command`:

```go
func getCommand() *cli.Command {
    return &cli.Command{
        Name:  "get",
        Usage: "Get a free port for current directory",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "name",
                Aliases: []string{"n"},
                Value:   "main",
                Usage:   "Named allocation",
            },
        },
        Action: func(c *cli.Context) error {
            // Implementation
            return nil
        },
    }
}
```

## Docker Detection

For the `scan` command, when a process is `docker-proxy`:

1. Attempt to find the actual project directory:
   - Check container labels for `com.docker.compose.project.working_dir`
   - Check bind mount sources
2. Requires `docker` CLI to be available

## Error Handling

- Use Go error wrapping with `fmt.Errorf("context: %w", err)`
- Return appropriate exit codes (0=success, 1=no ports/not found, 2=config error)
- With `--verbose`, output debug info to stderr
- Without `--verbose`, keep output minimal

## Testing Guidelines

Essential test cases:

### Port Allocation
- Allocate port for new directory
- Reuse port for same directory
- Multiple named allocations per directory

### Freeze Period
- Port in freeze period is not reallocated
- Port after freeze period can be reallocated

### Locking
- Locked port cannot be allocated to other directory
- Unlocking allows reallocation

### TTL
- Old allocations are expired
- Active allocations are not expired

### Concurrency
- Multiple processes calling `get` simultaneously
- File locking prevents corruption

### Edge Cases
- No free ports available
- All ports in freeze period
- Invalid configuration
- Missing allocations file (create with defaults)

## Default Files

First run creates both files with defaults:

**Config file** (`~/.config/portpls/config.json`):
```json
{
  "port_start": 20000,
  "port_end": 22000,
  "freeze_period": "24h",
  "allocation_ttl": "0",
  "log_file": ""
}
```

**Allocations file** (`~/.local/share/portpls/allocations.json`):
```json
{
  "version": 1,
  "last_issued_port": 0,
  "allocations": {}
}
```
