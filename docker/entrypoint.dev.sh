#!/bin/sh

echo "================================================"
echo "Easy Orders Development Container Starting..."
echo "================================================"

# Navigate to app directory
cd /app

# Generate Swagger documentation
echo ""
echo "Generating Swagger documentation..."
swag init -g cmd/server/main.go -o docs/
if [ $? -eq 0 ]; then
    echo "✅ Swagger documentation generated successfully"
else
    echo "❌ Failed to generate Swagger documentation"
    exit 1
fi

echo ""
echo "Starting Air for hot reload..."
echo "================================================"
echo ""

# Start Air for hot reload
exec air -c .air.toml
