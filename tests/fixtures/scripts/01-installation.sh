#!/bin/bash

set -e

echo "=== User Journey Test: Fresh Installation ==="
echo

# Test binary availability
echo "1. Checking if aps binary is installed..."
if ! command -v aps &> /dev/null; then
    echo "ERROR: aps binary not found in PATH"
    exit 1
fi
echo "✓ aps binary found: $(which aps)"
echo

# Test version command
echo "2. Checking aps version..."
aps version || aps --version
echo "✓ Version check passed"
echo

# Test help command
echo "3. Testing help command..."
aps --help > /dev/null
echo "✓ Help command works"
echo

echo "=== Installation Tests Passed ==="
