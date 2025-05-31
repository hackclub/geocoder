#!/bin/bash

# Integration Test Runner for Hack Club Geocoder
# This script sets up the environment and runs the full integration test suite

set -e

echo "🧪 Hack Club Geocoder Integration Test Suite"
echo "============================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must be run from the project root directory"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    exit 1
fi

# Set up test environment variables
export CGO_ENABLED=0
export GOOGLE_GEOCODING_API_KEY=${GOOGLE_GEOCODING_API_KEY:-"test-key"}
export IPINFO_API_KEY=${IPINFO_API_KEY:-""}
export ADMIN_USERNAME=${ADMIN_USERNAME:-"admin"}
export ADMIN_PASSWORD=${ADMIN_PASSWORD:-"admin"}

# Check if Docker is available for database tests
DOCKER_AVAILABLE=false
if command -v docker &> /dev/null && docker info &> /dev/null; then
    DOCKER_AVAILABLE=true
    echo "✅ Docker is available - will test with real database"
else
    echo "⚠️  Docker not available - will use mock database"
fi

# Start test database if Docker is available
if [ "$DOCKER_AVAILABLE" = true ]; then
    echo "🐳 Starting test database..."
    
    # Stop any existing test containers
    docker stop geocoder-test-postgres 2>/dev/null || true
    docker rm geocoder-test-postgres 2>/dev/null || true
    
    # Start fresh test database
    docker run -d \
        --name geocoder-test-postgres \
        -e POSTGRES_DB=geocoder_test \
        -e POSTGRES_USER=test \
        -e POSTGRES_PASSWORD=test \
        -p 5433:5432 \
        postgres:16-alpine
    
    # Wait for database to be ready
    echo "⏳ Waiting for database to be ready..."
    sleep 5
    
    # Set database URL for tests
    export DATABASE_URL="postgres://test:test@localhost:5433/geocoder_test?sslmode=disable"
    
    # Function to cleanup on exit
    cleanup() {
        echo "🧹 Cleaning up test database..."
        docker stop geocoder-test-postgres 2>/dev/null || true
        docker rm geocoder-test-postgres 2>/dev/null || true
    }
    trap cleanup EXIT
fi

echo "🏃 Running integration tests..."
echo ""

# Run the integration tests with verbose output
if go test -tags=integration -v ./tests/; then
    echo ""
    echo "✅ All integration tests passed!"
    
    if [ "$DOCKER_AVAILABLE" = true ]; then
        echo "📊 Test database logs:"
        docker logs geocoder-test-postgres --tail 10
    fi
else
    echo ""
    echo "❌ Some integration tests failed!"
    
    if [ "$DOCKER_AVAILABLE" = true ]; then
        echo "📊 Test database logs:"
        docker logs geocoder-test-postgres --tail 20
    fi
    
    exit 1
fi

echo ""
echo "🎉 Integration test suite completed successfully!"
