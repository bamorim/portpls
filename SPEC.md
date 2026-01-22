# portpls - Port Allocation CLI Specification

## Overview

`portpls` is a CLI utility for automatic free port selection and allocation management. It helps developers avoid port conflicts when running multiple projects, git worktrees, or parallel development environments.

The tool allocates free TCP ports from a configured range, persists them per directory (and optional name), and provides commands to manage these allocations.

## Key Features

- **Directory-based persistence**: Each directory automatically gets its own port(s)
- **Named allocations**: Support multiple services per directory (web, api, db, etc.)
- **Port locking**: Prevent reallocation of critical ports
- **Freeze period**: Prevent immediate reuse of recently released ports
- **TTL support**: Auto-expire stale allocations
- **Process detection**: Scan and record ports already in use
- **Docker awareness**: Detect ports used by Docker containers

## Installation Location

**Configuration file:**
- `~/.config/portpls/config.json` - configuration settings

**State file:**
- `~/.local/share/portpls/allocations.json` - allocation database (port assignments)

## Core Concepts

### Directory-based Allocation

Ports are allocated per directory path. Running `portpls get` from the same directory always returns the same port:

```bash
$ cd ~/projects/project-a
$ portpls get
3000

$ cd ~/projects/project-b  
$ portpls get
3001

$ cd ~/projects/project-a
$ portpls get
3000  # Same port as before
```

### Named Allocations

A single directory can have multiple named allocations for different services:

```bash
$ cd ~/myproject
$ portpls get --name web
3010

$ portpls get --name api
3011

$ portpls get --name db
3012

$ portpls get  # default name is "main"
3000
```

### Port Locking

Locked ports cannot be reallocated to other directories. Useful for long-running services:

```bash
$ portpls lock          # Lock current directory's port (allocates if needed)
$ portpls unlock        # Unlock current directory's port
```

### Freeze Period

After a port is allocated, it enters a "freeze period" where it won't be allocated to other directories. This prevents race conditions when services start slowly.

Default: 24 hours (configurable)

## Data Structure

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
    },
    "3011": {
      "directory": "/home/user/myproject",
      "name": "api",
      "assigned_at": "2026-01-21T10:01:00Z",
      "last_used_at": "2026-01-21T14:35:00Z",
      "locked": false
    }
  }
}
```

### Configuration Fields

**`port_start`** (integer, default: 20000)
- Start of port range to allocate from

**`port_end`** (integer, default: 22000)
- End of port range (inclusive)

**`freeze_period`** (duration string, default: "24h")
- Time period during which a port cannot be reallocated after release
- Formats: "24h" (hours), "30m" (minutes), "1d" (days)
- "0" disables freeze period

**`allocation_ttl`** (duration string, default: "0")
- Auto-expire allocations after this period of inactivity
- Formats: "30d" (days), "720h" (hours), "24h30m" (combined)
- "0" disables TTL (allocations never expire)

**`log_file`** (string, optional)
- Path to log file for recording allocation changes
- Example: "~/.local/share/portpls/portpls.log"
- Empty string disables logging

### Allocation Fields

**`directory`** (string, required)
- Absolute path to the directory that owns this port
- Automatically resolved from current working directory

**`name`** (string, required)
- Name of this allocation within the directory
- Default: "main"
- Allows multiple ports per directory

**`assigned_at`** (ISO 8601 timestamp, required)
- When this port was first allocated

**`last_used_at`** (ISO 8601 timestamp, required)
- Last time this port was requested (for TTL calculation)
- Updated on each `get` command

**`locked`** (boolean, required)
- Whether this port is locked (cannot be reallocated)
- Default: false

## CLI Design

Using [`urfave/cli`](https://github.com/urfave/cli) for argument parsing.

### Global Options

```
--config PATH       Path to config file (default: ~/.config/portpls/config.json)
--allocations PATH  Path to allocations file (default: ~/.local/share/portpls/allocations.json)
--directory PATH    Override current directory (useful for scripts)
--verbose           Enable debug output to stderr
--help, -h          Show help
--version, -v       Show version
```

### Commands

#### `portpls get`

Get a free port for the current directory (or reuse existing allocation).

**Usage:**
```bash
portpls get [options]
```

**Options:**
```
--name NAME, -n NAME    Named allocation (default: "main")
```

**Behavior:**
1. Check if directory already has an allocation for the given name
2. If yes and port is still free, update `last_used_at` and return it
3. If yes but port is taken, reallocate a new port and return it
4. If no, find first free port in range and allocate it
5. Output port number to stdout

**Examples:**
```bash
# Get default port for current directory
portpls get
# Output: 3000

# Get named port
portpls get --name api
# Output: 3001

# Use in scripts
PORT=$(portpls get)
npm run dev -- --port $PORT
```

**Exit codes:**
- 0: Success, port outputted to stdout
- 1: No free ports available in range
- 2: Configuration or file system error

#### `portpls list`

List all port allocations.

**Usage:**
```bash
portpls list [options]
```

**Options:**
```
--format FORMAT, -f FORMAT    Output format: table, json (default: table)
```

**Behavior:**
1. Load all allocations
2. For each port, check if it's currently in use (optional, for STATUS column)
3. Display in requested format

**Table format example:**
```
PORT  DIRECTORY                 NAME   STATUS  LOCKED  ASSIGNED             LAST_USED
3000  ~/code/project-a          main   free    yes     2026-01-21 10:00     2026-01-21 14:00
3010  ~/myproject               web    busy    no      2026-01-21 10:00     2026-01-21 14:30
3011  ~/myproject               api    free    no      2026-01-21 10:01     2026-01-21 14:35
```

**JSON format example:**
```json
[
  {
    "port": 3000,
    "directory": "/home/user/code/project-a",
    "name": "main",
    "status": "free",
    "locked": true,
    "assigned_at": "2026-01-21T10:00:00Z",
    "last_used_at": "2026-01-21T14:00:00Z"
  }
]
```

**Exit codes:**
- 0: Success

#### `portpls lock`

Lock a port to prevent reallocation.

**Usage:**
```bash
portpls lock [options]
```

**Options:**
```
--name NAME, -n NAME    Named allocation to lock (default: "main")
```

**Behavior:**
- If allocation exists for current directory + name: lock it
- If no allocation exists: allocate a new port and lock it

**Examples:**
```bash
# Lock default allocation (allocates if needed)
portpls lock

# Lock named allocation (allocates if needed)
portpls lock --name web
```

**Exit codes:**
- 0: Success
- 1: No free ports available in range
- 2: Configuration or file system error

#### `portpls unlock`

Unlock a port.

**Usage:**
```bash
portpls unlock [options]
```

**Options:**
```
--name NAME, -n NAME    Named allocation to unlock (default: "main")
```

**Behavior:**
- Unlocks the allocation for current directory + name
- Does not remove the allocation, just unlocks it

**Examples:**
```bash
# Unlock default allocation
portpls unlock

# Unlock named allocation
portpls unlock --name web
```

**Exit codes:**
- 0: Success
- 1: Allocation not found

#### `portpls forget`

Remove port allocations.

**Usage:**
```bash
portpls forget [options]
```

**Options:**
```
--name NAME, -n NAME      Remove specific named allocation for current directory (default: "main")
--all                     Remove ALL allocations for current directory
--all-directories         When combined with --all, remove ALL allocations for ALL directories
```

**Behavior:**
- Requires either `--name` or `--all` (explicit is better than implicit)
- With `--name`: Remove specific named allocation for current directory
- With `--all` only: Remove ALL allocations for current directory
- With `--all --all-directories`: Remove ALL allocations for ALL directories (interactive confirmation)

**Examples:**
```bash
# Remove specific named allocation (default is "main")
portpls forget --name
# Output: Cleared allocation 'main' for /home/user/myproject (was port 3010)

# Remove specific named allocation explicitly
portpls forget --name web
# Output: Cleared allocation 'web' for /home/user/myproject (was port 3010)

# Remove all allocations for current directory
portpls forget --all
# Output: Cleared 3 allocation(s) for /home/user/myproject

# Remove all allocations everywhere
portpls forget --all --all-directories
# Prompt: This will remove ALL port allocations. Continue? [y/N]
# Output: Cleared 15 allocation(s)
```

**Exit codes:**
- 0: Success
- 1: User declined confirmation (for --all --all-directories)
- 2: No flags specified (error)

#### `portpls scan`

Scan port range and record busy ports.

**Usage:**
```bash
portpls scan [options]
```

**Behavior:**
1. Scan all ports in configured range
2. For each busy port:
   - Attempt to determine which process is using it
   - Try to determine the working directory of that process
   - If directory found: create allocation for that directory
   - If directory not found (e.g., root process): create allocation with marker like "(unknown:3005)"
3. Skip ports already in allocations
4. Report results

**Docker detection:**
- If process is `docker-proxy`, attempt to find the actual project directory:
  - Check container labels for `com.docker.compose.project.working_dir`
  - Check bind mount sources
  - Requires `docker` CLI to be available

**Examples:**
```bash
# Scan port range
portpls scan
# Output:
# Scanning ports 20000-22000...
# Port 20005: used by node (pid=12345, cwd=/home/user/projects/app-a) - recorded
# Port 20014: used by docker-proxy (container in /home/user/my-compose-app) - recorded
# Port 20020: already allocated to /home/user/other-project
# 
# Recorded 2 new allocation(s)

# Run with sudo for full process info
sudo -E portpls scan
```

**Exit codes:**
- 0: Success
- 1: Error during scan

#### `portpls config`

Show or modify configuration.

**Usage:**
```bash
portpls config [KEY] [VALUE]
```

**Behavior:**
- No arguments: Show all configuration
- KEY only: Show specific configuration value
- KEY and VALUE: Set configuration value

**Examples:**
```bash
# Show all config
portpls config
# Output:
# port_start: 20000
# port_end: 22000
# freeze_period: 24h
# allocation_ttl: 30d

# Show specific value
portpls config port_start
# Output: 20000

# Set value
portpls config port_start 5000
# Output: Set port_start to 5000

# Set freeze period
portpls config freeze_period 12h
# Output: Set freeze_period to 12h
```

**Exit codes:**
- 0: Success
- 1: Invalid configuration key or value

## Port Allocation Algorithm

```
1. Load configuration from ~/.config/portpls/config.json
   - Create file with defaults if it doesn't exist

2. Load allocations from ~/.local/share/portpls/allocations.json
   - Create file with defaults if it doesn't exist

3. Get current directory absolute path (or use --directory flag)

3. Check if allocation exists for (directory, name):
   
   YES: 
     a. Update last_used_at to current time
     b. Check if port is still free (attempt bind on 127.0.0.1:PORT)
        - If free: return port, save allocations
        - If taken: proceed to step 4 (allocate new port)
   
   NO:
     Proceed to step 4

4. Find free port:
   a. Start from last_issued_port + 1 (or port_start if not set)
   b. For each port in range:
      - Skip if port is in freeze period (assigned_at + freeze_period > now)
      - Skip if port is locked by another directory
      - Skip if port is already allocated to another (directory, name)
      - Attempt to bind to 127.0.0.1:PORT
      - If bind succeeds: port is free, go to step 5
      - If bind fails: port is busy, try next
   c. If reached port_end, wrap around to port_start
   d. If no free port found after full cycle: ERROR

5. Create allocation:
   - port: selected port
   - directory: current directory absolute path
   - name: from --name flag or "main"
   - assigned_at: current timestamp
   - last_used_at: current timestamp
   - locked: false

6. Update last_issued_port to selected port

7. Save allocations to file (atomic write: temp file + rename)

8. Output port to stdout

9. Exit 0
```

## Port Checking Logic

When checking if a port is free:

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

## Logging

If `log_file` is configured, all allocation changes are logged:

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

## File Locking

To prevent corruption from concurrent access:

1. Use file locking (flock on Unix) on allocations file
2. Lock before read-modify-write operations
3. Unlock after write completes
4. Handle lock timeout (e.g., 5 seconds) gracefully

## Implementation Notes

### Language and Framework

- **Language:** Go
- **CLI Framework:** [urfave/cli](https://github.com/urfave/cli/tree/v2) v2
- **File locking:** Use `golang.org/x/sys/unix` flock or similar
- **JSON handling:** Standard library `encoding/json`
- **Time parsing:** Support duration formats like "24h", "30d", "1h30m"

### Project Structure

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
├── Makefile
└── README.md
```

### Commands Implementation

Each command should be implemented as a separate function that returns an `cli.Command`:

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

### Error Handling

- Use Go error wrapping with `fmt.Errorf("context: %w", err)`
- Return appropriate exit codes
- With `--verbose`, output debug info to stderr
- Without `--verbose`, keep output minimal

### Testing

Essential test cases:

1. **Port allocation:**
   - Allocate port for new directory
   - Reuse port for same directory
   - Multiple named allocations per directory

2. **Freeze period:**
   - Port in freeze period is not reallocated
   - Port after freeze period can be reallocated

3. **Locking:**
   - Locked port cannot be allocated to other directory
   - Unlocking allows reallocation

4. **TTL:**
   - Old allocations are expired
   - Active allocations are not expired

5. **Concurrency:**
   - Multiple processes calling `get` simultaneously
   - File locking prevents corruption

6. **Edge cases:**
   - No free ports available
   - All ports in freeze period
   - Invalid configuration
   - Missing allocations file (create with defaults)

## Default Configuration

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

## Integration Examples

### With Docker Compose

```bash
# .envrc (with direnv)
export WEB_PORT=$(portpls get --name web)
export API_PORT=$(portpls get --name api)
export DB_PORT=$(portpls get --name db)
```

```yaml
# docker-compose.yml
services:
  web:
    ports:
      - "${WEB_PORT}:3000"
  api:
    ports:
      - "${API_PORT}:4000"
  db:
    ports:
      - "${DB_PORT}:5432"
```

### With npm scripts

```json
{
  "scripts": {
    "dev": "PORT=$(portpls get) next dev -p $PORT",
    "dev:api": "PORT=$(portpls get --name api) node server.js"
  }
}
```

### With AI Agents

Add to CLAUDE.md or similar:

```markdown
## Running dev server

Always use portpls before starting dev server:

​```bash
PORT=$(portpls get) npm run dev -- --port $PORT
​```

For multiple services:

​```bash
WEB_PORT=$(portpls get --name web)
API_PORT=$(portpls get --name api)
docker-compose up
​```
```

## Future Enhancements (Optional)

These are not required for the initial implementation but could be added later:

1. **Port ranges per directory pattern:**
   - Allow specifying different port ranges for different directory patterns
   - Example: `~/work/*` uses 4000-5000, `~/personal/*` uses 5000-6000

2. **Export command:**
   - Export allocations to JSON for backup/restore
   - Import allocations from JSON

3. **Doctor command:**
   - Check for issues (overlapping locks, stale allocations, etc.)
   - Suggest fixes

4. **Watch command:**
   - Monitor port usage in real-time
   - Alert when allocated ports become busy

5. **Remote allocations:**
   - Support for shared allocation database (e.g., Redis)
   - Useful for team environments

## Summary

`portpls` is a focused CLI tool that:

- Automatically allocates free ports from a configured range
- Maintains persistent allocations per directory and optional name
- Prevents conflicts through port locking and freeze periods
- Provides simple commands: `get`, `list`, `lock`, `unlock`, `forget`, `scan`, `config`
- Uses `urfave/cli` for clean command structure
- Stores config in `~/.config/portpls/config.json` and state in `~/.local/share/portpls/allocations.json`

The design is simple, focused, and solves the specific problem of port conflicts in multi-worktree development environments.
