# Event Processor Testing Guide

## 🚀 Quick Start Testing

This guide will help you test the Event Processor service locally using Docker and LocalStack.

### Prerequisites

- Docker and Docker Compose installed
- `awslocal` CLI tool (will be installed automatically)
- Port 4566 (LocalStack) and 8080 (Event Processor) available

---

## ⚡ Quick Test (5 Minutes)

### One-Command Test
```bash
# Navigate to deployments directory
cd deployments

# Run automated quick test
../scripts/test-system.sh quick
```

### Manual Quick Test
```bash
# Navigate to deployments directory
cd deployments

# 1. Start the System
docker-compose up -d

# 2. Verify Health (30 seconds)
docker-compose ps
curl http://localhost:8080/health

# 3. Watch Events Flow (2 minutes)
docker-compose logs -f event-producer
# In another terminal:
docker-compose logs -f event-processor

# 4. Verify AWS Resources
awslocal sqs list-queues --endpoint-url=http://localhost:4566
awslocal dynamodb list-tables --endpoint-url=http://localhost:4566
```

### Expected Results ✅
- **3 containers running**: LocalStack, Event Processor, Event Producer
- **Health check**: Returns `{"healthy": true}`
- **Event flow**: Producer logs showing "Sent event X", Processor logs showing validation
- **AWS resources**: 2 SQS queues, 2 DynamoDB tables

---

## 📋 Test Scenarios

### 1. Basic System Startup

**Objective**: Verify all services start correctly

```bash
# Navigate to deployments directory
cd deployments

# Start all services
docker-compose up -d

# Verify services are running
docker-compose ps
```

**Expected Result**: All services should be `Up` status:
- `event-processor-localstack` (healthy)
- `event-processor-service` (healthy) 
- `event-producer` (running)

### 2. Health Check Verification

**Objective**: Confirm the Event Processor is healthy

```bash
# Test health endpoint
curl http://localhost:8080/health
```

**Expected Result**: JSON response with `"healthy": true` and database connectivity check.

### 3. AWS Resources Verification

**Objective**: Verify LocalStack resources are created

```bash
# Check SQS queues
awslocal sqs list-queues --endpoint-url=http://localhost:4566

# Check DynamoDB tables
awslocal dynamodb list-tables --endpoint-url=http://localhost:4566
```

**Expected Result**: 
- SQS: `event-queue` and `event-dlq` queues
- DynamoDB: `events` and `events-clients` tables

### 4. Event Production Testing

**Objective**: Verify events are being generated and queued

```bash
# Monitor producer logs (should show events being sent)
docker-compose logs -f event-producer

# Check queue messages
awslocal sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/event-queue \
  --attribute-names ApproximateNumberOfMessages \
  --endpoint-url=http://localhost:4566
```

**Expected Result**: 
- Producer logs showing "Sent event X" messages
- Queue attributes showing messages being processed

### 5. Event Processing Testing

**Objective**: Verify events are being consumed and processed

```bash
# Monitor processor logs
docker-compose logs -f event-processor

# Look for validation and processing messages
docker-compose logs event-processor | grep "Event validated"
```

**Expected Result**:
- Logs showing event validation
- Client permission checks
- Retry mechanisms for failed events

---

## 🔍 Advanced Testing Scenarios

### 6. Load Testing

**Objective**: Verify system performance under load

```bash
# Monitor system resources during event processing
docker stats

# Check queue depth
awslocal sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/event-queue \
  --attribute-names ApproximateNumberOfMessages \
  --endpoint-url=http://localhost:4566
```

**Expected Result**: System should handle 5 events/second without backlog

### 7. Error Handling Testing

**Objective**: Verify error handling and retry mechanisms

```bash
# Check dead letter queue for failed events
awslocal sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/event-dlq \
  --attribute-names ApproximateNumberOfMessages \
  --endpoint-url=http://localhost:4566

# Monitor processor logs for retry attempts
docker-compose logs event-processor | grep "retry"
```

**Expected Result**: Failed events should be moved to DLQ after retries

### 8. Multi-tenancy Testing

**Objective**: Verify client ID validation

```bash
# Check client configurations table
awslocal dynamodb scan \
  --table-name events-clients \
  --endpoint-url=http://localhost:4566

# Monitor logs for client validation
docker-compose logs event-processor | grep "ClientID"
```

**Expected Result**: Events should be validated against client permissions

---

## 🧪 Performance Testing

### Load Test Scenarios

#### Scenario 1: Normal Load
- **Rate**: 5 events/second (default)
- **Duration**: 5 minutes
- **Expected**: No backlog, < 100ms latency

#### Scenario 2: Peak Load
- **Rate**: 20 events/second
- **Duration**: 2 minutes
- **Expected**: Temporary backlog, < 500ms latency

#### Scenario 3: Sustained Load
- **Rate**: 10 events/second
- **Duration**: 15 minutes
- **Expected**: Stable performance, no memory leaks

### Performance Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Throughput** | 5+ events/sec | Events processed per second |
| **Latency** | < 100ms | End-to-end processing time |
| **Error Rate** | < 5% | Failed events percentage |
| **Memory Usage** | < 512MB | Container memory consumption |
| **CPU Usage** | < 80% | Container CPU utilization |

---

## 🚨 Troubleshooting

### Common Issues and Solutions

#### Issue: Services won't start
```bash
# Check port conflicts
lsof -i :4566 -i :8080

# Clean up and restart
docker-compose down --remove-orphans
docker-compose up -d
```

#### Issue: LocalStack not ready
```bash
# Check LocalStack health
curl http://localhost:4566/_localstack/health

# Restart LocalStack
docker-compose restart localstack
sleep 30  # Wait for initialization
```

#### Issue: No events in queue
```bash
# Check producer status
docker-compose ps event-producer

# Restart producer
docker-compose restart event-producer
```

#### Issue: Persistence errors
```bash
# Check DynamoDB table schema
awslocal dynamodb describe-table --table-name events --endpoint-url=http://localhost:4566

# Check processor logs for specific errors
docker-compose logs event-processor | grep "ValidationException"
```

#### Issue: High memory usage
```bash
# Monitor container resources
docker stats

# Check for memory leaks
docker-compose logs event-processor | grep "memory"

# Restart if necessary
docker-compose restart event-processor
```

#### Issue: Slow event processing
```bash
# Check queue depth
awslocal sqs get-queue-attributes \
  --queue-url http://localhost:4566/000000000000/event-queue \
  --attribute-names ApproximateNumberOfMessages \
  --endpoint-url=http://localhost:4566

# Monitor processor performance
docker-compose logs event-processor | grep "processing time"
```

---

## 🧹 Cleanup

### Stop Testing Environment

```bash
# Stop all services
docker-compose down

# Remove volumes (optional - cleans all data)
docker-compose down -v

# Clean up images (optional)
docker-compose down --rmi all
```

---

## 📈 Expected Test Results

### Successful Test Indicators

✅ **Startup**: All 3 containers running and healthy
✅ **Health Check**: Returns `{"healthy": true}` with sub-checks
✅ **Event Generation**: Producer sending 5 events/second by default
✅ **Event Processing**: Processor consuming and validating events
✅ **Multi-tenancy**: Client permission validation working
✅ **Error Handling**: Retry logic with exponential backoff
✅ **Observability**: Structured logging with correlation IDs



