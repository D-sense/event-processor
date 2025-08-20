# Docker Configuration

This directory contains all Docker-related files for the Event Processor services.

## Files

| File | Purpose | Service |
|------|---------|---------|
| `Dockerfile.processor` | Event Processor service container | Main event processing service |
| `Dockerfile.producer` | Event Producer service container | Test event generator |

## Usage

### Build Individual Services
```bash
# From project root
docker build -f deployments/docker/Dockerfile.processor -t event-processor .
docker build -f deployments/docker/Dockerfile.producer -t event-producer .
```

### Build via Docker Compose
```bash
# From deployments/ directory
docker-compose build
```

## Container Details

### Event Processor Container
- **Base Image**: `golang:1.23-alpine` → `alpine:3.18`
- **Port**: 8080
- **Binary**: `/app/event-processor`
- **User**: Non-root (`appuser:appgroup`)
- **Health Check**: Built-in health endpoint
- **Volumes**: Schema files mounted from host

### Event Producer Container  
- **Base Image**: `golang:1.23-alpine` → `alpine:latest`
- **Port**: 8081 (exposed, not published)
- **Binary**: `/root/producer`
- **Purpose**: Generate test events for development
- **Volumes**: Schema files mounted from host

## Build Context

Both Dockerfiles use the **project root** as build context:
```yaml
build:
  context: ..  # Project root
  dockerfile: deployments/docker/Dockerfile.processor
```

This allows access to:
- `go.mod` and `go.sum` 
- Source code in `cmd/`, `internal/`, `pkg/`
- Schema files in `schemas/`

## Multi-Stage Builds

Both containers use multi-stage builds for:
- **Smaller final images** (Alpine-based)
- **Security** (no build tools in final image)
- **Performance** (optimized Go binaries)

## Development

### Local Changes
```bash
# Rebuild after code changes
docker-compose build event-processor
docker-compose up -d event-processor

# View logs
docker-compose logs -f event-processor
```

### Debugging
```bash
# Run with shell access
docker run -it --entrypoint /bin/sh event-processor

# Check container contents
docker exec -it event-processor-service ls -la /app
```

## Best Practices Applied

- ✅ **Multi-stage builds** for optimized images
- ✅ **Non-root user** for security
- ✅ **Minimal base images** (Alpine)
- ✅ **Layer caching** optimization
- ✅ **Health checks** for monitoring
- ✅ **Proper file organization** in dedicated directory
