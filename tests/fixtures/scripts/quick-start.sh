#!/bin/bash

set -e

echo "=== Quick Start: Docker Testing Environment ==="
echo
echo "This script will set up and run the Docker testing environment."
echo

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running. Please start Docker and try again."
    exit 1
fi

echo "✓ Docker is running"
echo

# Step 1: Build binary
echo "Step 1: Building APS binary..."
if [ ! -f "bin/aps" ]; then
    make build
else
    echo "Binary already exists at bin/aps"
fi
echo "✓ Binary ready"
echo

# Step 2: Build Docker test image
echo "Step 2: Building Docker test image..."
make docker-build-test
echo "✓ Docker image ready"
echo

# Step 3: Install binary in test container
echo "Step 3: Installing binary in test container..."
make docker-test-install
echo "✓ Binary installed in test container"
echo

# Step 4: Run quick tests
echo "Step 4: Running quick verification tests..."
echo
echo "Testing binary availability..."
docker compose -f docker-compose.test.yml run --rm test-env \
    /usr/local/bin/aps --help > /dev/null
echo "✓ Binary is accessible"
echo

echo "Testing profile creation..."
docker compose -f docker-compose.test.yml run --rm test-env \
    /usr/local/bin/aps profile create quick-test --display-name "Quick Test"
echo "✓ Profile creation works"
echo

echo "Cleaning up test profile..."
docker compose -f docker-compose.test.yml run --rm test-env \
    /usr/local/bin/aps profile delete quick-test --force
echo "✓ Cleanup works"
echo

echo "=== Quick Start Complete ==="
echo
echo "To run the full user journey test suite:"
echo "  make docker-test-e2e-user"
echo
echo "To enter an interactive test environment:"
echo "  make docker-test-shell"
echo
echo "For more information, see docs/DOCKER_TESTING.md"
