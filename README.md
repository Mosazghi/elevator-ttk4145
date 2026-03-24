[![Go Tests](https://github.com/Mosazghi/elevator-ttk4145/actions/workflows/test.yml/badge.svg)](https://github.com/Mosazghi/elevator-ttk4145/actions/workflows/test.yml)

# Distributed Elevator Controller

Real-time distributed elevator control for TTK4145, with shared state sync and distributed hall-call assignment.

## Requirements

- Go 1.20+
- Make

## Run


```bash
go run ./cmd/elevator --id=<id> --port=<port> --floors=<n> --loglevel=<level>
```

Flags:

- `-id`: node ID (default `1`)
- `-port`: elevator server port (default `30000`)
- `-floors`: number of floors (default `4`)
- `-loglevel`: slog level (`debug=-4`, `info=0`, `warn=4`, `error=8`). Default `debug`.

## Environment

- `ENV=production` or `ENV=prod`: production mode
- Unset `ENV`: development mode

This is used primarily for filtering echos in the network.

Example:

```bash
ENV=production make run
```

## Test

```bash
make test
make race
```

Run one package:

```bash
go test ./internal/statesync -v
```

## Documentation

For a more interactive experience, you can use `pkgsite` to serve documentation locally:

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest

pkgsite
```

## Project Layout

- `cmd/elevator`: application entrypoint
- `internal/controller`: local elevator decision/state transitions
- `internal/statesync`: distributed worldview sync and reconciliation
- `internal/order_handler`: hall-call assignment integration
- `internal/network`: message transport
- `internal/orchestrator`: event loop wiring driver, controller, and sync
- `pkg/elevio`: elevator driver abstraction and TCP-backed implementation
- `executables/`: executable binaries and tools
