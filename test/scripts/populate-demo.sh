#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🚀 NATS Demo Data Population Script"
echo "===================================="
echo ""

# Check if NATS is running
echo "📡 Checking if NATS is running on localhost:4222..."
if ! nc -z localhost 4222 2>/dev/null; then
    echo "❌ NATS server is not running on localhost:4222"
    echo ""
    echo "To start NATS with JetStream enabled, run:"
    echo "  docker run -d --name nats -p 4222:4222 nats:latest -js"
    echo ""
    echo "Or if using docker-compose:"
    echo "  cd test/docker && docker-compose up -d"
    exit 1
fi

echo "✅ NATS server is running!"
echo ""

# Build the utility
echo "🔨 Building populate-demo-data utility..."
cd "$SCRIPT_DIR"
go build -o populate-demo-data populate-demo-data.go

echo "✅ Build successful!"
echo ""

# Run the utility
echo "📝 Populating demo data..."
./populate-demo-data

echo ""
echo "✨ All done! Your NATS server now has demo data."
echo ""
echo "Next steps:"
echo "  1. Run your n9s/n2s tool to view the streams"
echo "  2. Capture screenshots or record videos for your README"
echo "  3. Clean up demo data when done with: nats stream purge --all"
echo ""

# Cleanup the binary
rm -f populate-demo-data

