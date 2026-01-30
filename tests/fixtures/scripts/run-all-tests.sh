#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXTURES_DIR="$(dirname "$SCRIPT_DIR")"

echo "======================================"
echo "APS User Journey Test Suite"
echo "======================================"
echo

# Run all test scripts in sequence
TEST_SCRIPTS=(
    "01-installation.sh"
    "02-initial-setup.sh"
    "03-workflow-execution.sh"
    "04-configuration-management.sh"
    "05-cleanup-operations.sh"
)

FAILED_TESTS=()
PASSED_TESTS=()

for script in "${TEST_SCRIPTS[@]}"; do
    SCRIPT_PATH="$FIXTURES_DIR/scripts/$script"
    if [ ! -f "$SCRIPT_PATH" ]; then
        echo "ERROR: Test script not found: $script"
        exit 1
    fi

    echo "--------------------------------------"
    echo "Running: $script"
    echo "--------------------------------------"

    if bash "$SCRIPT_PATH"; then
        PASSED_TESTS+=("$script")
        echo "✓ $script PASSED"
    else
        FAILED_TESTS+=("$script")
        echo "✗ $script FAILED"
    fi

    echo
done

# Summary
echo "======================================"
echo "Test Summary"
echo "======================================"
echo "Total Tests: ${#TEST_SCRIPTS[@]}"
echo "Passed: ${#PASSED_TESTS[@]}"
echo "Failed: ${#FAILED_TESTS[@]}"
echo

if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    echo "✓ ALL TESTS PASSED"
    exit 0
else
    echo "✗ FAILED TESTS:"
    for test in "${FAILED_TESTS[@]}"; do
        echo "  - $test"
    done
    exit 1
fi
