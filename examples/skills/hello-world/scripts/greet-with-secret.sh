#!/bin/bash
# Demonstrates secret placeholder replacement

# This placeholder will be replaced by the SecretReplacer
API_KEY="${SECRET:API_KEY}"

echo "Greeting with authentication!"
echo "API Key (first 5 chars): ${API_KEY:0:5}..."
echo "Secret replacement working!"
