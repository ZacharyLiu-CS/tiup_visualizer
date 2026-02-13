#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Development Setup"
echo "======================================"

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)

# Setup frontend
echo "Setting up frontend..."
cd "$PROJECT_ROOT/frontend"

if [ ! -d "node_modules" ]; then
    npm install
fi

echo "Frontend setup complete!"

echo ""
echo "======================================"
echo "Development environment ready!"
echo "======================================"
echo ""
echo "To start development:"
echo "  Backend:  cd backend-go && go run ."
echo "  Frontend: cd frontend && npm run dev"
echo ""
