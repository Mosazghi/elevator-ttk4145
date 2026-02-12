[![Go Tests](https://github.com/Mosazghi/elevator-ttk4145/actions/workflows/test.yml/badge.svg)](https://github.com/Mosazghi/elevator-ttk4145/actions/workflows/test.yml)

 TTK4145 Real-Time Programming - Distributed Systems Project
Distributed elevator control system with network synchronization.

## Quick Start

```bash
# Run single elevator
make run

# Run multiple elevators
make run-multi

# Run tests
make test
make race
```

## Environment Variables

- `ENV=production` or `ENV=prod` - Enable production mode (filters UDP echo messages)
- Default (unset) - Development mode (receives all messages including echoes)

```bash
# Production mode
ENV=production make run
```

## Manual Run

```bash
go run ./cmd/elevator --id=<elevator_id> --port=<port>
```

## Manual Tests

```bash
go test <package_name> -v
``` 

For example, to test *only* the statesync package:

```bash
go test ./internal/statesync -v
```
