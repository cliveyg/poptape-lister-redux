#!/bin/bash

# Test script for poptape-lister-redux API with new authentication middleware
# This tests the API endpoints with the new X-Access-Token authentication

BASE_URL="http://localhost:8400"
ACCESS_TOKEN="test-token-123"

echo "=== Testing poptape-lister-redux API with new authentication ==="
echo "Base URL: $BASE_URL"
echo "Using Access Token: $ACCESS_TOKEN"
echo ""

# Test system status (no auth required)
echo "1. Testing system status (no auth required)..."
curl -s -w "\nResponse Code: %{http_code}\n" "$BASE_URL/list/status"
echo ""

# Test watching count (no auth required, will fail without MongoDB but tests routing)
echo "2. Testing watching count (no auth required)..."
curl -s -w "\nResponse Code: %{http_code}\n" "$BASE_URL/list/watching/803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
echo ""

# Test authenticated endpoints with missing token
echo "3. Testing watchlist (GET) - missing auth token..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     "$BASE_URL/list/watchlist"
echo ""

# Test authenticated endpoints with X-Access-Token (will fail without poptape-authy service)
echo "4. Testing watchlist (GET) - with X-Access-Token..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -H "X-Access-Token: $ACCESS_TOKEN" \
     "$BASE_URL/list/watchlist"
echo ""

echo "5. Testing watchlist (POST) - with X-Access-Token..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -X POST \
     -H "Content-Type: application/json" \
     -H "X-Access-Token: $ACCESS_TOKEN" \
     -d '{"uuid":"803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"}' \
     "$BASE_URL/list/watchlist"
echo ""

echo "6. Testing with wrong Content-Type..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -X POST \
     -H "Content-Type: text/plain" \
     -H "X-Access-Token: $ACCESS_TOKEN" \
     -d '{"uuid":"803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"}' \
     "$BASE_URL/list/watchlist"
echo ""

echo "7. Testing favourites (GET) - with X-Access-Token..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -H "X-Access-Token: $ACCESS_TOKEN" \
     "$BASE_URL/list/favourites"
echo ""

echo "8. Testing invalid endpoint..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     "$BASE_URL/list/invalid"
echo ""

echo "=== Test completed ==="