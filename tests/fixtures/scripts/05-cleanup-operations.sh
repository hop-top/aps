#!/bin/bash

set -e

echo "=== User Journey Test: Cleanup Operations ==="
echo

# Create multiple test profiles
echo "1. Creating test profiles..."
for i in 1 2 3; do
    PROFILE_ID="cleanup-test-$$-$i"
    aps profile new "$PROFILE_ID" --display-name "Cleanup Test $i"
    echo "✓ Created profile: $PROFILE_ID"
done
echo

# List all profiles
echo "2. Listing all profiles..."
aps profile list
echo

# Delete profiles by removing their directories
echo "3. Deleting profiles..."
DATA_DIR="${APS_DATA_PATH:-${XDG_DATA_HOME:-$HOME/.local/share}/aps}"
for i in 1 2 3; do
    PROFILE_ID="cleanup-test-$$-$i"
    rm -rf "$DATA_DIR/profiles/$PROFILE_ID"
    echo "✓ Deleted profile: $PROFILE_ID"
done
echo

# Verify deletion
echo "4. Verifying profiles were deleted..."
REMAINING=$(aps profile list | grep "cleanup-test-$$-" || true)
if [ -z "$REMAINING" ]; then
    echo "✓ All test profiles deleted successfully"
else
    echo "ERROR: Some profiles remain: $REMAINING"
    exit 1
fi
echo

echo "=== Cleanup Operations Tests Passed ==="
