#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Production Deploy"
echo "======================================"

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}Building frontend...${NC}"
cd "$PROJECT_ROOT/frontend"

if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

npm run build
echo -e "${GREEN}Frontend built successfully!${NC}"

echo -e "${YELLOW}Setting up backend...${NC}"
cd "$PROJECT_ROOT/backend"

if [ ! -d "venv" ]; then
    python3 -m venv venv
fi

source venv/bin/activate
pip install -q --upgrade pip
pip install -q -r requirements.txt

# Copy frontend build to backend static
rm -rf static
cp -r "$PROJECT_ROOT/frontend/dist" static

if [ ! -f ".env" ]; then
    cp .env.example .env
fi

echo -e "${GREEN}Setup complete!${NC}"

# Create cleanup function
cleanup() {
    echo -e "\n${YELLOW}Shutting down...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

echo ""
echo "======================================"
echo -e "${GREEN}Starting TiUP Visualizer (Production Mode)${NC}"
echo "======================================"
echo ""
echo -e "Access: ${GREEN}http://localhost:8000${NC}"
echo -e "API Docs: ${GREEN}http://localhost:8000/docs${NC}"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Start server
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 &
SERVER_PID=$!

wait
