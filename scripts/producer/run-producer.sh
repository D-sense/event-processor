#!/bin/bash

# run-producer.sh - Script to run the event producer

set -e

# Ensure this script is executable
if [ ! -x "$0" ]; then
    echo "Making script executable..."
    chmod +x "$0"
fi

echo "🚀 Starting Event Producer..."

# Color codes for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
PRODUCER_RATE=${PRODUCER_RATE:-5}
COMPOSE_FILE="../../deployments/docker-compose.yml"

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Function to run producer locally (requires Go)
run_local() {
    print_header "Running Producer Locally"
    
    if ! command -v go &> /dev/null; then
        print_warning "Go is not installed. Using Docker instead..."
        run_docker
        return
    fi
    
    # Set environment variables for local development
    export AWS_REGION=us-east-1
    export AWS_ACCESS_KEY_ID=test
    export AWS_SECRET_ACCESS_KEY=test
    export AWS_ENDPOINT_URL=http://localhost:4566
    export SQS_QUEUE_URL=http://localhost:4566/000000000000/event-queue
    export PRODUCER_RATE=$PRODUCER_RATE
    
    print_info "Producer rate: $PRODUCER_RATE events/second"
    print_info "Queue URL: $SQS_QUEUE_URL"
    
    cd ../../
    go run ./cmd/producer
}

# Function to run producer via Docker
run_docker() {
    print_header "Running Producer via Docker"
    
    if ! command -v docker-compose &> /dev/null; then
        print_warning "Docker Compose is not installed."
        exit 1
    fi
    
    print_info "Producer rate: $PRODUCER_RATE events/second"
    
    # Update producer rate in environment
    PRODUCER_RATE=$PRODUCER_RATE docker-compose -f "$COMPOSE_FILE" up event-producer
}

# Function to run producer in background
run_background() {
    print_header "Running Producer in Background"
    
    PRODUCER_RATE=$PRODUCER_RATE docker-compose -f "$COMPOSE_FILE" up -d event-producer
    
    print_info "Producer started in background"
    print_info "To view logs: docker-compose -f $COMPOSE_FILE logs -f event-producer"
    print_info "To stop: docker-compose -f $COMPOSE_FILE stop event-producer"
}

# Function to stop producer
stop_producer() {
    print_header "Stopping Producer"
    
    docker-compose -f "$COMPOSE_FILE" stop event-producer
    print_info "Producer stopped"
}

# Function to show producer logs
show_logs() {
    print_header "Producer Logs"
    
    docker-compose -f "$COMPOSE_FILE" logs -f event-producer
}

# Main execution
case "${1:-docker}" in
    "local")
        run_local
        ;;
    "docker")
        run_docker
        ;;
    "background"|"bg")
        run_background
        ;;
    "stop")
        stop_producer
        ;;
    "logs")
        show_logs
        ;;
    *)
        echo "Usage: $0 {local|docker|background|stop|logs}"
        echo ""
        echo "Commands:"
        echo "  local      - Run producer locally (requires Go)"
        echo "  docker     - Run producer via Docker (default)"
        echo "  background - Run producer in background"
        echo "  stop       - Stop background producer"
        echo "  logs       - Show producer logs"
        echo ""
        echo "Environment Variables:"
        echo "  PRODUCER_RATE - Events per second (default: 5)"
        exit 1
        ;;
esac
