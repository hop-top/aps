#!/bin/bash

set -e

echo "=== User Journey Test: Workflow Execution ==="
echo

PROFILE_ID="exec-test-$$"

# Create profile
echo "1. Creating profile for execution test..."
aps profile new "$PROFILE_ID" --display-name "Execution Test"
echo "✓ Profile created"
echo

# Test basic command execution
echo "2. Testing basic command execution..."
OUTPUT=$(aps run "$PROFILE_ID" -- echo "Hello from APS")
if [ "$OUTPUT" = "Hello from APS" ]; then
    echo "✓ Command execution successful"
else
    echo "ERROR: Expected 'Hello from APS', got '$OUTPUT'"
    exit 1
fi
echo

# Test environment isolation
echo "3. Testing environment isolation..."
ENV_OUTPUT=$(aps run "$PROFILE_ID" -- env)
if echo "$ENV_OUTPUT" | grep -q "APS_PROFILE_ID=$PROFILE_ID"; then
    echo "✓ Profile ID environment variable set"
else
    echo "ERROR: APS_PROFILE_ID not found in environment"
    exit 1
fi
echo

# Test multiple commands
echo "4. Testing multiple commands in sequence..."
aps run "$PROFILE_ID" -- sh -c "echo 'First' && echo 'Second'"
echo "✓ Multiple commands executed"
echo

echo "=== Workflow Execution Tests Passed ==="

# Cleanup
echo "Cleaning up test profile..."
aps profile delete "$PROFILE_ID" --force || true
