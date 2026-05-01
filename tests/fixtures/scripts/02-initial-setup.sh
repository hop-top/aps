#!/bin/bash

set -e

echo "=== User Journey Test: Initial Setup ==="
echo

# Check XDG config directory
echo "1. Checking XDG configuration directory..."
XDG_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}"
APS_CONFIG="$XDG_CONFIG/aps"

if [ ! -d "$APS_CONFIG" ]; then
    echo "✓ APS config directory will be created on first use"
else
    echo "✓ APS config directory exists: $APS_CONFIG"
fi
echo

# Create first profile
echo "2. Creating first profile..."
PROFILE_ID="test-profile-$$"

if aps profile list | grep -q "$PROFILE_ID"; then
    echo "Profile $PROFILE_ID already exists, deleting..."
    aps profile delete "$PROFILE_ID" --force || true
fi

aps profile create "$PROFILE_ID" --display-name "Test Agent"
echo "✓ Profile created: $PROFILE_ID"
echo

# List profiles
echo "3. Listing profiles..."
aps profile list
echo

# Show profile details
echo "4. Showing profile details..."
aps profile show "$PROFILE_ID"
echo

echo "=== Initial Setup Tests Passed ==="

# Cleanup
echo "Cleaning up test profile..."
aps profile delete "$PROFILE_ID" --force || true
