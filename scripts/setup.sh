#!/bin/bash

# setup.sh - One-command setup script for Event Processor

set -e

# Ensure this script is executable
if [ ! -x "$0" ]; then
    echo "Making script executable..."
    chmod +x "$0"
fi

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    local all_good=true
    
    # Check Docker
    if command -v docker &> /dev/null; then
        local docker_version=$(docker --version | cut -d' ' -f3 | cut -d',' -f1)
        print_success "Docker found (version $docker_version)"
    else
        print_error "Docker not found. Please install Docker first."
        print_info "Visit: https://docs.docker.com/get-docker/"
        all_good=false
    fi
    
    # Check Docker Compose
    if command -v docker-compose &> /dev/null; then
        local compose_version=$(docker-compose --version | cut -d' ' -f4 | cut -d',' -f1)
        print_success "Docker Compose found (version $compose_version)"
    else
        print_error "Docker Compose not found. Please install Docker Compose first."
        print_info "Visit: https://docs.docker.com/compose/install/"
        all_good=false
    fi
    
    # Check Git
    if command -v git &> /dev/null; then
        print_success "Git found"
    else
        print_error "Git not found. Please install Git first."
        all_good=false
    fi
    
    # Check curl
    if command -v curl &> /dev/null; then
        print_success "curl found"
    else
        print_warning "curl not found. Some tests may not work."
    fi
    
    # Check ports
    print_info "Checking port availability..."
    local ports_in_use=()
    
    if lsof -i :4566 &> /dev/null; then
        ports_in_use+=(4566)
    fi
    
    if lsof -i :8080 &> /dev/null; then
        ports_in_use+=(8080)
    fi
    
    if lsof -i :8081 &> /dev/null; then
        ports_in_use+=(8081)
    fi
    
    if [ ${#ports_in_use[@]} -eq 0 ]; then
        print_success "All required ports (4566, 8080, 8081) are available"
    else
        print_warning "Ports in use: ${ports_in_use[*]}"
        print_info "You may need to stop services using these ports"
    fi
    
    if [ "$all_good" = false ]; then
        print_error "Prerequisites not met. Please install missing software and try again."
        exit 1
    fi
    
    print_success "All prerequisites met!"
}

# Setup project
setup_project() {
    print_header "Setting Up Project"
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -f "deployments/docker-compose.yml" ]; then
        print_error "Please run this script from the event-processor project root directory"
        print_info "Expected files: go.mod, deployments/docker-compose.yml"
        exit 1
    fi
    
    print_success "Project structure verified"
    
    # Make scripts executable
    print_info "Making scripts executable..."
    chmod +x scripts/*.sh 2>/dev/null || true
    chmod +x scripts/*/*.sh 2>/dev/null || true
    print_success "Scripts made executable"
}

# Start services
start_services() {
    print_header "Starting Services"
    
    cd deployments
    
    print_info "Pulling Docker images..."
    docker-compose pull --ignore-pull-failures
    
    print_info "Building application images..."
    docker-compose build
    
    print_info "Starting services..."
    docker-compose up -d
    
    print_success "Services started"
    cd ..
}

# Wait for services
wait_for_services() {
    print_header "Waiting for Services to Initialize"
    
    print_info "Waiting for LocalStack to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:4566/_localstack/health &> /dev/null; then
            print_success "LocalStack is ready"
            break
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        print_error "LocalStack failed to start within expected time"
        print_info "Check logs: docker-compose -f deployments/docker-compose.yml logs localstack"
        exit 1
    fi
    
    print_info "Waiting for Event Processor to be ready..."
    sleep 10
    
    local processor_attempts=15
    local processor_attempt=1
    
    while [ $processor_attempt -le $processor_attempts ]; do
        if curl -s http://localhost:8080/health &> /dev/null; then
            print_success "Event Processor is ready"
            break
        fi
        
        echo -n "."
        sleep 2
        processor_attempt=$((processor_attempt + 1))
    done
    
    if [ $processor_attempt -gt $processor_attempts ]; then
        print_warning "Event Processor may not be fully ready yet"
        print_info "Check logs: docker-compose -f deployments/docker-compose.yml logs event-processor"
    fi
}

# Verify installation
verify_installation() {
    print_header "Verifying Installation"
    
    # Run automated tests
    print_info "Running automated verification..."
    if chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh quick; then
        print_success "Installation verification passed!"
    else
        print_error "Installation verification failed"
        print_info "Check the output above for specific issues"
        return 1
    fi
}

# Show next steps
show_next_steps() {
    print_header "🎉 Setup Complete!"
    
    echo -e "${GREEN}Your Event Processor is now running!${NC}"
    echo ""
    echo -e "${BLUE}Services:${NC}"
    echo "  • LocalStack (AWS simulation): http://localhost:4566"
    echo "  • Event Processor: http://localhost:8080"
    echo "  • Event Producer: Running in background"
    echo ""
    echo -e "${BLUE}Useful Commands:${NC}"
    echo "  • Health check: curl http://localhost:8080/health"
    echo "  • View logs: docker-compose -f deployments/docker-compose.yml logs -f"
    echo "  • Stop services: docker-compose -f deployments/docker-compose.yml down"
    echo "  • Run tests: chmod +x ./scripts/test-system.sh && ./scripts/test-system.sh"
    echo ""
    echo -e "${BLUE}Documentation:${NC}"
    echo "  • Quick testing: cat QUICK_TEST.md"
    echo "  • Full testing guide: cat TESTING_GUIDE.md"
    echo "  • Architecture: cat README.md"
    echo ""
    echo -e "${GREEN}Happy coding! 🚀${NC}"
}

# Cleanup on failure
cleanup_on_failure() {
    print_error "Setup failed. Cleaning up..."
    cd deployments 2>/dev/null || true
    docker-compose down &> /dev/null || true
    cd .. 2>/dev/null || true
}

# Main execution
main() {
    print_header "Event Processor Setup"
    print_info "This script will set up the Event Processor application locally"
    
    # Set trap for cleanup on failure
    trap cleanup_on_failure ERR
    
    check_prerequisites
    setup_project
    start_services
    wait_for_services
    
    if verify_installation; then
        show_next_steps
    else
        print_error "Setup completed but verification failed"
        print_info "Services are running but may need troubleshooting"
        print_info "Check logs: docker-compose -f deployments/docker-compose.yml logs"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-setup}" in
    "setup")
        main
        ;;
    "clean")
        print_header "Cleaning Up"
        cd deployments 2>/dev/null || true
        docker-compose down -v --remove-orphans
        docker system prune -f
        print_success "Cleanup complete"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  setup    - Set up Event Processor (default)"
        echo "  clean    - Clean up Docker resources"
        echo "  help     - Show this help message"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
