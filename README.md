<p align="center">
  <img src="docs/assets/proxez-logo.svg" alt="Proxez logo" width="420">
</p>

<p align="center">
  Simple Layer 7 proxy and load-balancer deployment.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/status-in%20development-orange" alt="Development status">
  <img src="https://img.shields.io/badge/protocol-Layer%207-blue" alt="Layer 7">
  <img src="https://img.shields.io/badge/license-Apache2.0-green" alt="MIT License">
</p>

# Proxez

Proxez is a Layer 7 HTTP reverse proxy and load balancer, written in Go with
no third-party dependencies.

It routes incoming HTTP requests to healthy upstream services using:

- Round robin
- Smooth weighted round robin
- Least connections *(planned)*

> **Work in progress.** Proxez is a learning project that is being built in
> the open. Check the progress list below before relying on a feature.

## Status

Proxez is under active development.

### Proxy core
- [x] HTTP/1.1 request forwarding
- [x] Backend selection: round robin
- [x] Backend selection: smooth weighted round robin
- [ ] Backend selection: least connections
- [x] Shared backend state (health, active requests, failures)
- [x] Health checks *(temporary: calls `/health` on each backend)*
- [ ] Failure handling (retry, mark unhealthy on errors)
- [ ] Graceful shutdown
- [ ] Multiple pools, each with its own backends and strategy
- [ ] Routing by host and path prefix

### Configuration storage
- [x] Commands and validation (`Validate`)
- [x] Applying commands to the in-memory config (`Apply`)
- [x] Binary encoding of commands
- [x] Write-ahead log with checksums and sync to disk
- [x] Crash recovery (replay the log, cut a half-written tail)
- [x] Exclusive lock on the data folder
- [ ] Snapshots and log compaction
- [ ] Log segments and rotation

### Admin
- [ ] Admin page served from the binary (`go:embed`)
- [ ] Add and remove pools, backends and routes from the page
- [ ] Live health and request counters
- [ ] Metrics and logs

### Not planned (for now)
- HTTPS termination
- WebSocket proxying
- HTTP/2 upstreams
- Automatic service discovery

## Requirements

- Go 1.23 or later
- Linux or macOS *(Windows is not supported yet: the data folder lock uses
  `flock`)*
- Network access to the configured upstream services

## Install

```bash
go install github.com/cptcupnoodles/proxez/cmd/proxez@latest
```

Or build from source:

```bash
git clone https://github.com/cptcupnoodles/proxez
cd proxez
go build -o bin/proxez ./cmd/proxez
```

## Run

```bash
./bin/proxez -data ./data
```

| Flag | Default | Meaning |
|---|---|---|
| `-data` | `./data` | Folder for the config log (and snapshots, later) |

Only one Proxez process can use a data folder at a time.

## How it works

Configuration changes are written as small commands to an append-only log
before they are applied in memory. On startup the log is replayed to rebuild
the config.

```
admin change -> Validate -> append to log and sync -> Apply -> live pools
```

Live data such as health and request counters is not saved. It is rebuilt
on every start.

## Development

```bash
go vet ./...
go test -race -count=1 ./...
```