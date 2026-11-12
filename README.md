<p align="center">
  <img src="docs/assets/proxez-logo.svg" alt="Proxez logo" width="420">
</p>

<p align="center">
  Simple Layer 7 proxy and load-balancer deployment.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/status-in%20development-orange" alt="Development status">
  <img src="https://img.shields.io/badge/protocol-Layer%207-blue" alt="Layer 7">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="MIT License">
</p>

# Proxez

Proxez is a Layer 7 reverse proxy and load-balancing platform for routing HTTP traffic and distributing requests across upstream services.

It provides configurable traffic routing, multiple load-balancing algorithms, upstream management, and a web interface for managing proxy deployments.

Proxez is designed for reliable service delivery across local environments, self-hosted infrastructure, and production deployments.

## Status

Proxez is under active development.

The core proxy and load-balancing features are being developed first. The web interface is also under development.

The configuration format, APIs, and user interface may change before the first stable release.

Do not use Proxez in production until a stable release is available.

## Features

- Layer 7 reverse proxying
- HTTP traffic routing
- Multiple upstream servers
- Smooth weighted round-robin load balancing
- Least-connections load balancing
- Upstream server configuration
- Proxy configuration management
- Developer-focused web interface
- Deployment workflow under development

## Load-Balancing Algorithms

### Smooth Weighted Round Robin

Smooth weighted round robin distributes traffic according to the configured weight of each upstream server.

Example:

| Upstream | Weight |
|---|---:|
| Server A | 5 |
| Server B | 3 |
| Server C | 2 |

The expected traffic distribution is approximately 50%, 30%, and 20%.

### Least Connections

Least connections sends each new request to the upstream server with the fewest active connections.

This algorithm is useful when requests have different processing times.

## Web Interface

The Proxez web interface is under development.

The planned interface will provide:

- Proxy creation and configuration
- Upstream server management
- Load-balancing algorithm selection
- Server weight configuration
- Service status
- Connection information
- Runtime configuration management
- Logs and basic diagnostics
- Deployment workflows

## Architecture

```text
                    ┌──────────────┐
Client Request ────▶│ Layer 7 Proxy│
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │ Load Balancer│
                    └──────┬───────┘
                           │
             ┌─────────────┼─────────────┐
             ▼             ▼             ▼
        Upstream A     Upstream B     Upstream C
