# Event Processor Service

A high-performance, scalable event processing service built with Go, designed to handle high-throughput event streams with multi-tenant support, dead letter queues, and comprehensive monitoring.

## 🚀 Quick Start

**New to this project?** Follow these steps:

1. **Prerequisites**: Docker 20.0+, Docker Compose 2.0+, Go 1.23+
2. **One-Command Setup**: `./scripts/setup.sh`
3. **Quick Test**: `cd deployments && ../scripts/test-system.sh quick`

### Automated Setup (5 minutes)
```bash
# Clone and setup
git clone [repository-url]
cd event-processor
./scripts/setup.sh
```

### Manual Setup
```bash
# Clone repository
git clone [repository-url]
cd event-processor

# Start services
cd deployments
docker-compose up -d

# Wait for infrastructure setup (15-30 seconds)
# The event-processor service will automatically create:
# - DynamoDB tables (events, events-clients)
# - SQS queues (event-queue, event-dlq)
# - Sample client configurations
sleep 30

# Verify system
../scripts/test-system.sh quick
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
- **Ports**: 4566, 8080, 8081 available
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
lsof -i :4566 -i :8080 -i :8081

# If ports are in use, stop conflicting services
# Kill processes using these ports if necessary
```

### Step 2: Service Startup

#### Start All Services
```bash
# Navigate to deployments
cd deployments

# Start services in background
docker-compose up -d

# Monitor startup
docker-compose logs -f
```

#### Wait for Initialization
```bash
# Wait for LocalStack to be ready (15-30 seconds)
# The event-processor will automatically create infrastructure
sleep 30

# Check service status
docker-compose ps
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
../scripts/test-system.sh quick

# Full system test
../scripts/test-system.sh
```

---

## 🧪 Quick Testing

### 5-Minute Verification

#### 1. Verify System Health
```bash
# Check all services are running
docker-compose ps

# Test health endpoint
curl http://localhost:8080/health
```

#### 2. Watch Events Flow
```bash
# Watch events being produced
docker-compose logs -f event-producer

# In another terminal, watch events being processed
docker-compose logs -f event-processor
```

#### 3. Verify AWS Resources
```bash
# Check queues
docker exec event-processor-localstack awslocal sqs list-queues

# Check tables
docker exec event-processor-localstack awslocal dynamodb list-tables

# Check event count
docker exec event-processor-localstack awslocal dynamodb scan --table-name events --select COUNT
```

### Expected Results ✅

- **3 containers running**: LocalStack, Event Processor, Event Producer
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
│   └── producer/
|   └── run-producer.sh
|   └── test-simple.sh
|   └── setup.sh
|   └── test-system.sh
├── TESTING_GUIDE.md
└── README.md

```

---

## 🔧 Development Workflow

### Local Development
1. **Start Services**: `cd deployments && docker-compose up -d`
2. **Test**: `../scripts/test-system.sh quick`

### Testing
- **Unit Tests**: `go test ./...`
- **Integration Tests**: `../scripts/test-system.sh`
- **Load Tests**: See `TESTING_GUIDE.md`

### System Restart Test
```bash
# Test complete system restart
docker-compose down
docker-compose up -d

# Wait for auto-infrastructure setup
sleep 30

# Verify all services are healthy
docker-compose ps
curl http://localhost:8080/health
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
docker-compose down && docker-compose up -d
```

#### LocalStack Not Ready
```bash
# Check health
curl http://localhost:4566/_localstack/health

# Restart LocalStack
docker-compose restart localstack
sleep 30
```

#### No Events Flowing
```bash
# Check producer status
docker-compose ps event-producer

# Restart producer
docker-compose restart event-producer
```

#### Permission Validation Errors
```bash
# Check client configurations
docker exec event-processor-localstack awslocal dynamodb scan --table-name events-clients

# Verify event types are allowed for specific clients
# Default: client-001: [integration], client-002: [transaction, integration], client-003: [transaction, integration]
```

### Getting Help

1. **Check Logs**: `docker-compose logs [service-name]`
2. **Run Diagnostics**: `../scripts/test-system.sh`
3. **Clean Installation**: `docker-compose down -v && docker-compose up -d`

---

## 🧹 Cleanup

### Stop Development Environment
```bash
# Stop all services
docker-compose down

# Remove volumes (optional - cleans all data)
docker-compose down -v

# Clean up images (optional)
docker-compose down --rmi all
```



