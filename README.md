# Event Processor Service

A high-performance, scalable event processing service built with Go, designed to handle high-throughput event streams with multi-tenant support, dead letter queues, and comprehensive monitoring.

## 🚀 Quick Start

**New to this project?** Follow these steps:

1. **Prerequisites**: Docker 20.0+, Docker Compose 2.0+, Go 1.23+
2. **One-Command Setup**: `chmod +x ./scripts/setup.sh && ./scripts/setup.sh`
3. **Quick Test**: `chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh quick`

### Automated Setup (5 minutes)
```bash
# Clone and setup
git clone git@github.com:D-sense/event-processor.git
cd event-processor
chmod +x ./scripts/setup.sh && ./scripts/setup.sh
```

### Manual Setup
```bash
# Clone repository
git clone git@github.com:D-sense/event-processor.git
cd event-processor

# Start services
docker-compose -f deployments/docker-compose.yml up -d

# Wait for infrastructure setup (15-30 seconds)
# The event-processor service will automatically create:
# - DynamoDB tables (events, events-clients)
# - SQS queues (event-queue, event-dlq)
# - Sample client configurations
sleep 30

# Verify system
chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh quick
```

**Expected Result**: All tests should pass ✅

---

## 🏗️ Architecture Overview

### Core Components

- **Event Processor**: High-throughput event consumer with SQS integration
- **Event Producer**: Test event generator for development and testing
- **SQS Queues**: Main queue + Dead Letter Queue for failed events
- **DynamoDB**: Event persistence and client configuration storage
- **LocalStack**: Local AWS service emulation for development

### Key Features

- **Multi-tenancy**: Client ID validation and routing
- **Event Validation**: JSON Schema-based validation
- **Dead Letter Queues**: Failed event handling with retry logic
- **Health Monitoring**: Comprehensive health checks and metrics
- **Structured Logging**: Correlation IDs and structured log output
- **Graceful Shutdown**: Proper cleanup and resource management
- **Auto-Infrastructure**: Automatic creation of required AWS resources on startup
- **Security**: Client permission validation prevents unauthorized event processing
- **Centralized Logging**: Unified logging configuration with log level control

### Technology Stack

- **Language**: Go 1.23+
- **Containerization**: Docker + Docker Compose
- **Message Queue**: AWS SQS
- **Database**: AWS DynamoDB
- **Local Development**: LocalStack 3.1.0
- **AWS SDK**: AWS SDK for Go v2

---

## 📋 Prerequisites

### Required Software

| Software | Version | Purpose | Installation |
|----------|---------|---------|-------------|
| **Docker** | 20.0+ | Container runtime | [Install Docker](https://docs.docker.com/get-docker/) |
| **Docker Compose** | 2.0+ | Multi-container orchestration | [Install Compose](https://docs.docker.com/compose/install/) |
| **Git** | 2.0+ | Source code management | [Install Git](https://git-scm.com/downloads) |
| **curl** | Any | API testing | Usually pre-installed |

### Optional Tools

| Tool | Purpose | Installation |
|------|---------|-------------|
| **awslocal** | AWS CLI for LocalStack | `pip install awscli-local` |
| **jq** | JSON processing | `brew install jq` (macOS) / `apt install jq` (Ubuntu) |
| **Go 1.23+** | Local development | [Install Go](https://golang.org/doc/install) |

### System Requirements

- **Memory**: 4GB+ RAM available for Docker
- **Disk**: 2GB+ free space
- **Ports**: 4566, 8080 available
- **OS**: macOS, Linux, or Windows with WSL2

---

## 📖 Detailed Installation Steps

### Step 1: Environment Preparation

#### Check Docker Installation
```bash
# Verify Docker is running
docker --version
docker-compose --version

# Test Docker functionality
docker run hello-world
```

#### Check Port Availability
```bash
# Check if required ports are free
lsof -i :4566 -i :8080

# If ports are in use, stop conflicting services
# Kill processes using these ports if necessary
```

### Step 2: Service Startup

#### Start All Services
```bash
# Start services in background
docker-compose -f deployments/docker-compose.yml up -d

# Monitor startup
docker-compose -f deployments/docker-compose.yml logs -f
```

#### Wait for Initialization
```bash
# Wait for LocalStack to be ready (15-30 seconds)
# The event-processor will automatically create infrastructure
sleep 30

# Check service status
docker-compose -f deployments/docker-compose.yml ps
```

### Step 3: Verification

#### Run Health Checks
```bash
# Test Event Processor health
curl http://localhost:8080/health

# Check LocalStack health
curl http://localhost:4566/_localstack/health
```

#### Run Automated Tests
```bash
# Quick system test
chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh quick

# Full system test
chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh
```

### Step 4: Logging Configuration

#### Log Level Control
The system supports configurable log levels via the `LOG_LEVEL` environment variable:

```bash
# Set log level in docker-compose.yml
environment:
  - LOG_LEVEL=debug  # Options: debug, info, warn, error, fatal, panic

# Or override at runtime
docker-compose -f deployments/docker-compose.yml up -e LOG_LEVEL=debug
```

#### Available Log Levels
- **debug**: Detailed debug information
- **info**: General information (default)
- **warn**: Warning messages
- **error**: Error messages
- **fatal**: Fatal errors (terminates service)
- **panic**: Panic errors (terminates service)

#### Structured Logging
All logs are in JSON format with consistent fields:
- `timestamp`: ISO 8601 timestamp
- `level`: Log level
- `message`: Log message
- `service`: Service name
- `version`: Service version
- `component`: Component name (when applicable)
- `correlation_id`: Request correlation ID
- `event_id`, `event_type`, `client_id`: Event context

---

## 🧪 Quick Testing

### 5-Minute Verification

#### 1. Verify System Health
```bash
# Check all services are running
docker-compose -f deployments/docker-compose.yml ps

# Test health endpoint
curl http://localhost:8080/health
```

#### 2. Watch Events Flow
```bash
# Watch events being produced
docker-compose -f deployments/docker-compose.yml logs -f event-producer

# In another terminal, watch events being processed
docker-compose -f deployments/docker-compose.yml logs -f event-processor
```

#### 3. Verify AWS Resources
```bash
# Check queues
docker exec event-processor-localstack awslocal sqs list-queues

# Check tables
docker exec event-processor-localstack awslocal dynamodb list-tables

# Check event count
docker exec event-processor-localstack awslocal dynamodb scan --table-name events --select COUNT

# Fetch last 10 processed records from events table
docker exec event-processor-localstack awslocal dynamodb scan --table-name events --limit 10
```


### Expected Results ✅

- **3 containers running**: event-processor-localstack, event-processor-service, event-producer-service
- **Health check**: Returns `{"healthy": true}`
- **Event flow**: Producer logs showing "✅ Sent event X", Processor logs showing validation
- **AWS resources**: 2 SQS queues, 2 DynamoDB tables
- **Events processing**: Continuous event generation and processing

---

## 📁 Project Structure

```
event-processor/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── producer/
│       └── main.go
├── internal/
│   ├── config/
│   ├── consumer/
│   ├── validator/
│   ├── processor/
│   ├── persistence/
│   └── health/
├── pkg/
│   ├── aws/
│   ├── schema/
│   └── models/
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile.processor
│   │   └── Dockerfile.producer
│   ├── docker-compose.yml
├── schemas/
│   └── event-schema.json
├── scripts/
│   ├── setup-localstack.sh
│   ├── test-simple.sh
│   ├── setup.sh
│   ├── test-system.sh
│   └── producer/
│       └── run-producer.sh
├── TESTING_GUIDE.md
└── README.md

```

---

## 📚 Documentation

| Document | Purpose | Audience |
|----------|---------|----------|
| [`TESTING_GUIDE.md`](TESTING_GUIDE.md) | Comprehensive testing scenarios | QA and testing teams |
| [`deployments/docker/README.md`](deployments/docker/README.md) | Docker configuration guide | All developers |

### Scripts
- [`scripts/setup.sh`](scripts/setup.sh) - One-command installation and setup
- [`scripts/test-system.sh`](scripts/test-system.sh) - Automated test suite
- [`scripts/setup-localstack.sh`](scripts/setup-localstack.sh) - LocalStack management
- [`scripts/producer/run-producer.sh`](scripts/producer/run-producer.sh) - Producer utilities

---

## 🚨 Troubleshooting

### Common Issues

#### Services Won't Start
```bash
# Check ports
lsof -i :4566 -i :8080

# Clean restart
docker-compose -f deployments/docker-compose.yml down && docker-compose -f deployments/docker-compose.yml up -d
```

#### LocalStack Not Ready
```bash
# Check health
curl http://localhost:4566/_localstack/health

# Restart LocalStack
docker-compose -f deployments/docker-compose.yml restart localstack
sleep 30
```

#### No Events Flowing
```bash
# Check producer status
docker-compose -f deployments/docker-compose.yml ps event-producer

# Restart producer
docker-compose -f deployments/docker-compose.yml restart event-producer
```

#### Permission Validation Errors
```bash
# Check client configurations
docker exec event-processor-localstack awslocal dynamodb scan --table-name events-clients

# Verify event types are allowed for specific clients
# Current: All clients can send all event types (integration, monitoring, transaction, user_action)
```

### Getting Help

1. **Check Logs**: `docker-compose -f deployments/docker-compose.yml logs [service-name]`
2. **Run Diagnostics**: `chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh`
3. **Clean Installation**: `docker-compose -f deployments/docker-compose.yml down -v && docker-compose -f deployments/docker-compose.yml up -d`

---

## 🧹 Cleanup

### Stop Development Environment
```bash
# Stop all services
docker-compose -f deployments/docker-compose.yml down

# Remove volumes (optional - cleans all data)
docker-compose -f deployments/docker-compose.yml down -v

# Clean up images (optional)
docker-compose -f deployments/docker-compose.yml down --rmi all
```



