# portpls

`portpls` is a CLI utility for automatic free port selection and allocation management.

## Build

```bash
go build -o portpls ./cmd/portpls
```

## Quick Usage

```bash
# Get default port for current directory
portpls get

# Named allocation
portpls get --name api

# List allocations
portpls list
```
