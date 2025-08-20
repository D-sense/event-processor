#!/bin/bash

# test-system.sh - Automated test script for Event Processor

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
LOCALSTACK_URL="http://localhost:4566"
SERVICE_URL="http://localhost:8080"
TEST_TIMEOUT=60

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Test 1: Check Docker services
test_docker_services() {
    print_header "Testing Docker Services"
    
    local services=("event-processor-localstack" "event-processor-service" "event-producer")
    local all_running=true
    
    for service in "${services[@]}"; do
        if docker ps --format "table {{.Names}}" | grep -q "$service"; then
            print_success "$service is running"
        else
            print_error "$service is not running"
            all_running=false
        fi
    done
    
    if [ "$all_running" = true ]; then
        print_success "All Docker services are running"
        return 0
    else
        print_error "Some Docker services are not running"
        return 1
    fi
}

# Test 2: Check LocalStack health
test_localstack_health() {
    print_header "Testing LocalStack Health"
    
    if curl -s "$LOCALSTACK_URL/_localstack/health" | grep -q "running"; then
        print_success "LocalStack is healthy"
        return 0
    else
        print_error "LocalStack is not healthy"
        return 1
    fi
}

# Test 3: Check Event Processor health
test_service_health() {
    print_header "Testing Event Processor Health"
    
    local response=$(curl -s "$SERVICE_URL/health" 2>/dev/null)
    if echo "$response" | grep -q '"healthy":true'; then
        print_success "Event Processor is healthy"
        
        # Check sub-components
        if echo "$response" | grep -q '"database":{"healthy":true'; then
            print_success "Database connectivity is healthy"
        else
            print_warning "Database connectivity issue detected"
        fi
        return 0
    else
        print_error "Event Processor health check failed"
        return 1
    fi
}

# Test 4: Check AWS resources
test_aws_resources() {
    print_header "Testing AWS Resources"
    
    # Check SQS queues
    print_info "Checking SQS queues..."
    local queues=$(awslocal sqs list-queues --endpoint-url="$LOCALSTACK_URL" --output text 2>/dev/null || echo "")
    
    if echo "$queues" | grep -q "event-queue" && echo "$queues" | grep -q "event-dlq"; then
        print_success "SQS queues are created"
    else
        print_error "SQS queues are missing"
        return 1
    fi
    
    # Check DynamoDB tables
    print_info "Checking DynamoDB tables..."
    local tables=$(awslocal dynamodb list-tables --endpoint-url="$LOCALSTACK_URL" --output text 2>/dev/null || echo "")
    
    if echo "$tables" | grep -q "events" && echo "$tables" | grep -q "events-clients"; then
        print_success "DynamoDB tables are created"
    else
        print_error "DynamoDB tables are missing"
        return 1
    fi
    
    return 0
}

# Test 5: Check event production
test_event_production() {
    print_header "Testing Event Production"
    
    print_info "Checking if producer is generating events..."
    
    # Get recent producer logs
    local producer_logs=$(docker-compose logs --tail=10 event-producer 2>/dev/null || echo "")
    
    if echo "$producer_logs" | grep -q "Sent event"; then
        local event_count=$(echo "$producer_logs" | grep -c "Sent event" || echo "0")
        print_success "Producer is generating events (found $event_count recent events)"
        return 0
    else
        print_error "No events found in producer logs"
        return 1
    fi
}

# Test 6: Check event processing
test_event_processing() {
    print_header "Testing Event Processing"
    
    print_info "Checking if processor is consuming events..."
    
    # Get recent processor logs
    local processor_logs=$(docker-compose logs --tail=20 event-processor 2>/dev/null || echo "")
    
    if echo "$processor_logs" | grep -q "Event validated"; then
        print_success "Processor is validating events"
    else
        print_warning "No event validation logs found recently"
    fi
    
    if echo "$processor_logs" | grep -q "correlation_id"; then
        print_success "Structured logging with correlation IDs is working"
    else
        print_warning "No correlation IDs found in logs"
    fi
    
    if echo "$processor_logs" | grep -q "retry_count"; then
        print_success "Retry mechanism is active"
    else
        print_info "No retry attempts detected (this is normal if processing is successful)"
    fi
    
    return 0
}

# Test 7: Check queue activity
test_queue_activity() {
    print_header "Testing Queue Activity"
    
    print_info "Checking queue message counts..."
    
    # Check main queue
    local queue_attrs=$(awslocal sqs get-queue-attributes \
        --queue-url "$LOCALSTACK_URL/000000000000/event-queue" \
        --attribute-names ApproximateNumberOfMessages \
        --endpoint-url="$LOCALSTACK_URL" 2>/dev/null || echo "")
    
    if [ -n "$queue_attrs" ]; then
        local msg_count=$(echo "$queue_attrs" | grep -o '"ApproximateNumberOfMessages": "[^"]*"' | cut -d'"' -f4 || echo "0")
        print_info "Main queue has $msg_count messages"
        
        if [ "$msg_count" -gt 0 ]; then
            print_success "Messages are flowing through the queue"
        else
            print_info "Queue is empty (messages being processed quickly)"
        fi
    else
        print_error "Could not retrieve queue attributes"
        return 1
    fi
    
    return 0
}

# Run all tests
run_all_tests() {
    print_header "Event Processor System Test Suite"
    print_info "Testing Event Processor service components..."
    
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    # Array of test functions
    local tests=(
        "test_docker_services"
        "test_localstack_health" 
        "test_service_health"
        "test_aws_resources"
        "test_event_production"
        "test_event_processing"
        "test_queue_activity"
    )
    
    # Run each test
    for test_func in "${tests[@]}"; do
        total_tests=$((total_tests + 1))
        echo ""
        
        if $test_func; then
            passed_tests=$((passed_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Summary
    print_header "Test Results Summary"
    echo -e "Total Tests: $total_tests"
    echo -e "${GREEN}Passed: $passed_tests${NC}"
    echo -e "${RED}Failed: $failed_tests${NC}"
    
    if [ $failed_tests -eq 0 ]; then
        print_success "All tests passed! 🎉"
        echo -e "\n${GREEN}✨ Event Processor system is working correctly!${NC}"
        return 0
    else
        print_error "Some tests failed"
        echo -e "\n${YELLOW}💡 Check the TESTING_GUIDE.md for troubleshooting steps${NC}"
        return 1
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  test     - Run all system tests (default)"
    echo "  quick    - Run quick health checks only"
    echo "  help     - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Run all tests"
    echo "  $0 test         # Run all tests"
    echo "  $0 quick        # Quick health check"
}

# Quick test mode
run_quick_tests() {
    print_header "Quick System Health Check"
    
    local quick_tests=(
        "test_docker_services"
        "test_service_health"
        "test_localstack_health"
    )
    
    local all_passed=true
    
    for test_func in "${quick_tests[@]}"; do
        echo ""
        if ! $test_func; then
            all_passed=false
        fi
    done
    
    echo ""
    if [ "$all_passed" = true ]; then
        print_success "Quick health check passed! ✅"
        return 0
    else
        print_error "Quick health check failed! ❌"
        return 1
    fi
}

# Main execution
main() {
    case "${1:-test}" in
        "test")
            run_all_tests
            ;;
        "quick")
            run_quick_tests
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            echo "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
    print_error "docker-compose.yml not found. Please run this script from the deployments directory."
    print_info "Expected path: event-processor/deployments/"
    exit 1
fi

main "$@"
