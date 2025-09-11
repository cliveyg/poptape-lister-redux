#!/bin/bash

# Test script for poptape-lister-redux API
# This tests the API endpoints without requiring MongoDB

BASE_URL="http://localhost:8400"
PUBLIC_ID="2a99371f-4188-49b8-a628-85e946540364"

echo "=== Testing poptape-lister-redux API ==="
echo "Base URL: $BASE_URL"
echo "Using Public ID: $PUBLIC_ID"
echo ""

# Test system status (no auth required)
echo "1. Testing system status..."
curl -s -w "\nResponse Code: %{http_code}\n" "$BASE_URL/list/status"
echo ""

# Test watching count (no auth required, will fail without MongoDB but tests routing)
echo "2. Testing watching count..."
curl -s -w "\nResponse Code: %{http_code}\n" "$BASE_URL/list/watching/803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
echo ""

# Test authenticated endpoints (will fail without MongoDB but tests routing and auth)
echo "3. Testing watchlist (GET) - requires auth..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -H "X-Public-ID: $PUBLIC_ID" \
     "$BASE_URL/list/watchlist"
echo ""

echo "4. Testing watchlist (POST) - requires auth..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -X POST \
     -H "Content-Type: application/json" \
     -H "X-Public-ID: $PUBLIC_ID" \
     -d '{"uuid":"803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"}' \
     "$BASE_URL/list/watchlist"
echo ""

echo "5. Testing favourites (GET) - requires auth..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -H "X-Public-ID: $PUBLIC_ID" \
     "$BASE_URL/list/favourites"
echo ""

echo "6. Testing viewed (GET) - requires auth..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     -H "X-Public-ID: $PUBLIC_ID" \
     "$BASE_URL/list/viewed"
echo ""

echo "7. Testing invalid endpoint..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     "$BASE_URL/list/invalid"
echo ""

echo "8. Testing endpoint without auth header..."
curl -s -w "\nResponse Code: %{http_code}\n" \
     "$BASE_URL/list/watchlist"
echo ""

echo "=== Test completed ==="