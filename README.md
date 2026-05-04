# Monitoring Agent

A monitoring agent built with Go using gopsutil for system metrics collection.

## Project Structure

```
├── cmd/
│   └── agent/          # Main agent application
├── internal/
│   ├── collector/      # Metrics collection logic
│   ├── config/         # Configuration management
│   └── exporter/       # Metrics exporting
├── deploy/
│   └── grafana/        # Grafana dashboards
└── go.mod
```

## Prerequisites

- Go 1.26.2 or higher
- gopsutil for system metrics

## Getting Started

```bash
go mod download
go run ./cmd/agent
```

## Development

To build the agent:

```bash
go build -o bin/agent ./cmd/agent
```


