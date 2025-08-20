#!/bin/bash

# setup-localstack.sh - Script to set up LocalStack environment for Event Processor

set -e

echo "🚀 Setting up LocalStack environment for Event Processor..."

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
COMPOSE_FILE="deployments/docker-compose.yml"
LOCALSTACK_URL="http://localhost:4566"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Check if docker and docker-compose are installed
check_dependencies() {
    print_header "Checking Dependencies"
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    print_status "Docker and Docker Compose are installed ✓"
}

# Start LocalStack
start_localstack() {
    print_header "Starting LocalStack"
    
    # Stop any existing containers
    docker-compose -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true
    
    # Start LocalStack service only
    docker-compose -f "$COMPOSE_FILE" up -d localstack
    
    print_status "LocalStack container started"
}

# Wait for LocalStack to be ready
wait_for_localstack() {
    print_header "Waiting for LocalStack to be Ready"
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$LOCALSTACK_URL/_localstack/health" | grep -q "running"; then
            print_status "LocalStack is ready! ✓"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "LocalStack failed to start within expected time"
    exit 1
}

# Verify AWS CLI local is available
check_awslocal() {
    print_header "Checking AWS CLI Local"
    
    if ! command -v awslocal &> /dev/null; then
        print_warning "awslocal not found. Installing via pip..."
        pip install awscli-local || {
            print_error "Failed to install awslocal. Please install manually: pip install awscli-local"
            exit 1
        }
    fi
    
    print_status "awslocal is available ✓"
}

# Verify resources were created
verify_resources() {
    print_header "Verifying Created Resources"
    
    # Check SQS queues
    print_status "Checking SQS queues..."
    local queues=$(awslocal sqs list-queues --endpoint-url="$LOCALSTACK_URL" --output text)
    if echo "$queues" | grep -q "event-queue" && echo "$queues" | grep -q "event-dlq"; then
        print_status "SQS queues created successfully ✓"
    else
        print_error "SQS queues not found"
        return 1
    fi
    
    # Check DynamoDB tables
    print_status "Checking DynamoDB tables..."
    local tables=$(awslocal dynamodb list-tables --endpoint-url="$LOCALSTACK_URL" --output text)
    if echo "$tables" | grep -q "events" && echo "$tables" | grep -q "events-clients"; then
        print_status "DynamoDB tables created successfully ✓"
    else
        print_error "DynamoDB tables not found"
        return 1
    fi
}

# Display helpful information
show_info() {
    print_header "Environment Information"
    
    echo -e "${BLUE}LocalStack Endpoint:${NC} $LOCALSTACK_URL"
    echo -e "${BLUE}Health Check:${NC} $LOCALSTACK_URL/_localstack/health"
    echo -e "${BLUE}SQS Queue URL:${NC} $LOCALSTACK_URL/000000000000/event-queue"
    echo -e "${BLUE}SQS DLQ URL:${NC} $LOCALSTACK_URL/000000000000/event-dlq"
    echo -e "${BLUE}DynamoDB Endpoint:${NC} $LOCALSTACK_URL"
    
    echo ""
    echo -e "${GREEN}Environment Variables:${NC}"
    echo "export AWS_ENDPOINT_URL=$LOCALSTACK_URL"
    echo "export AWS_ACCESS_KEY_ID=test"
    echo "export AWS_SECRET_ACCESS_KEY=test"
    echo "export AWS_DEFAULT_REGION=us-east-1"
    
    echo ""
    echo -e "${GREEN}Next Steps:${NC}"
    echo "1. Run the event processor: docker-compose -f $COMPOSE_FILE up event-processor"
    echo "2. Run the event producer: docker-compose -f $COMPOSE_FILE up event-producer"
    echo "3. Or run all services: docker-compose -f $COMPOSE_FILE up"
}

# Cleanup function
cleanup() {
    print_header "Cleaning Up"
    docker-compose -f "$COMPOSE_FILE" down --remove-orphans
    print_status "Environment cleaned up"
}

# Main execution
main() {
    # Handle cleanup on script exit
    trap cleanup EXIT
    
    # Parse command line arguments
    case "${1:-start}" in
        "start")
            check_dependencies
            check_awslocal
            start_localstack
            wait_for_localstack
            sleep 5  # Give init script time to run
            verify_resources
            show_info
            ;;
        "stop")
            print_header "Stopping LocalStack"
            docker-compose -f "$COMPOSE_FILE" down
            print_status "LocalStack stopped"
            ;;
        "restart")
            print_header "Restarting LocalStack"
            docker-compose -f "$COMPOSE_FILE" restart localstack
            wait_for_localstack
            verify_resources
            show_info
            ;;
        "status")
            print_header "LocalStack Status"
            if curl -s "$LOCALSTACK_URL/_localstack/health" | grep -q "running"; then
                print_status "LocalStack is running ✓"
                verify_resources
            else
                print_warning "LocalStack is not running"
            fi
            ;;
        "logs")
            print_header "LocalStack Logs"
            docker-compose -f "$COMPOSE_FILE" logs -f localstack
            ;;
        *)
            echo "Usage: $0 {start|stop|restart|status|logs}"
            echo ""
            echo "Commands:"
            echo "  start   - Start LocalStack and initialize resources"
            echo "  stop    - Stop LocalStack"
            echo "  restart - Restart LocalStack"
            echo "  status  - Check LocalStack status"
            echo "  logs    - Show LocalStack logs"
            exit 1
            ;;
    esac
}

# Don't cleanup on normal exit for start command
if [ "${1:-start}" == "start" ]; then
    trap - EXIT
fi

main "$@"
