# portpls

`portpls` is a CLI utility for automatic free port selection and allocation management. It helps developers avoid port conflicts when running multiple projects, git worktrees, or parallel development environments.

## Key Features

- **Directory-based persistence**: Each directory automatically gets its own port(s)
- **Named allocations**: Support multiple services per directory (web, api, db, etc.)
- **Port locking**: Prevent reallocation of critical ports
- **Freeze period**: Prevent immediate reuse of recently released ports
- **TTL support**: Auto-expire stale allocations
- **Process detection**: Scan and record ports already in use
- **Docker awareness**: Detect ports used by Docker containers

## Installation

### Using mise (recommended)

Install globally using [mise](https://mise.jdx.dev/):

```bash
mise use -g github:bamorim/portpls
```

This will download the latest release binary and make it available in your PATH.

### Using go install

Install the latest version:

```bash
go install github.com/bamorim/portpls@latest
```

Make sure `~/go/bin` is in your `PATH`:

```bash
export PATH=$PATH:~/go/bin
```

### Download binary from releases

Download the latest binary for your platform from the [GitHub Releases](https://github.com/bamorim/portpls/releases) page and add it to your PATH.

### Configuration

Configuration and state files are created automatically on first run:

- **Config**: `~/.config/portpls/config.json`
- **State**: `~/.local/share/portpls/allocations.json`

## Quick Start

```bash
# Get a port for the current directory
$ portpls get
20000

# Running from the same directory always returns the same port
$ portpls get
20000

# Different directories get different ports
$ cd ../other-project
$ portpls get
20001
```

## Commands

### `portpls get`

Get a free port for the current directory (or reuse existing allocation).

```bash
# Get default port for current directory
portpls get

# Get named port for a specific service
portpls get --name api

# Use in scripts
PORT=$(portpls get)
npm run dev -- --port $PORT
```

**Options:**
- `--name, -n NAME` - Named allocation (default: "main")

### `portpls list`

List all port allocations.

```bash
# Table format (default)
portpls list

# JSON format
portpls list --format json

# Filter to a specific directory
portpls list --directory .
```

**Output example:**
```
PORT  DIRECTORY                 NAME   STATUS  LOCKED  ASSIGNED             LAST_USED
3000  ~/code/project-a          main   free    yes     2026-01-21 10:00     2026-01-21 14:00
3010  ~/myproject               web    busy    no      2026-01-21 10:00     2026-01-21 14:30
```

**Options:**
- `--format, -f FORMAT` - Output format: table, json (default: table)
- `--directory PATH` - Filter allocations by directory

### `portpls lock` / `portpls unlock`

Lock a port to prevent reallocation. Useful for long-running services.

```bash
# Lock current directory's port (allocates if needed)
portpls lock

# Lock a named allocation
portpls lock --name web

# Unlock
portpls unlock
portpls unlock --name web
portpls unlock --directory ../archived-worktree
```

**Options:**
- `--name, -n NAME` - Named allocation (default: "main")
- `--directory PATH` - Override directory

### `portpls forget`

Remove port allocations.

```bash
# Remove default allocation for current directory
portpls forget

# Remove specific named allocation
portpls forget --name web

# Remove all allocations for current directory
portpls forget --all

# Remove all allocations everywhere (prompts for confirmation)
portpls forget --all --all-directories

# Remove allocation for a different directory
portpls forget --directory ../archived-worktree --name web
```

**Options:**
- `--name, -n NAME` - Named allocation to remove (default: "main")
- `--all` - Remove all allocations for current directory
- `--all-directories` - Combined with --all, remove everything
- `--directory PATH` - Override directory

### `portpls scan`

Scan port range and record busy ports. Attempts to determine which process is using each port and its working directory.

```bash
portpls scan
# Output:
# Scanning ports 20000-22000...
# Port 20005: used by node (pid=12345, cwd=/home/user/projects/app-a) - recorded
# Port 20014: used by docker-proxy (container in /home/user/my-compose-app) - recorded
# Recorded 2 new allocation(s)
```

### `portpls config`

Show or modify configuration.

```bash
# Show all config
portpls config

# Show specific value
portpls config port_start

# Set value
portpls config port_start 5000
portpls config freeze_period 12h
```

**Configuration options:**
- `port_start` - Start of port range (default: 20000)
- `port_end` - End of port range (default: 22000)
- `freeze_period` - Time before released port can be reallocated (default: "24h")
- `allocation_ttl` - Auto-expire inactive allocations after this period (default: "0" = disabled)
- `log_file` - Path to log file (default: "" = disabled)

## Global Options

```
--config PATH       Path to config file (default: ~/.config/portpls/config.json)
--allocations PATH  Path to allocations file (default: ~/.local/share/portpls/allocations.json)
--directory PATH    Override current directory (useful for scripts)
--verbose           Enable debug output to stderr
--help, -h          Show help
--version, -v       Show version
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

Add to your CLAUDE.md or agent instructions:

```markdown
## Running dev server

Always use portpls before starting dev server:

PORT=$(portpls get) npm run dev -- --port $PORT

For multiple services:

WEB_PORT=$(portpls get --name web)
API_PORT=$(portpls get --name api)
docker-compose up
```

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

$ portpls get  # default name is "main"
3000
```

### Port Locking

Locked ports cannot be reallocated to other directories:

```bash
$ portpls lock          # Lock current directory's port
$ portpls unlock        # Unlock it
```

### Freeze Period

After a port is allocated, it enters a "freeze period" where it won't be allocated to other directories. This prevents race conditions when services start slowly. Default: 24 hours.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | No free ports / allocation not found / user declined |
| 2 | Configuration or file system error |

## Credits

Inspired by [port-selector](https://github.com/dapi/port-selector)
