#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - One-Click Start"
echo "======================================"

PROJECT_ROOT=$(cd "$(dirname "$0")" && pwd)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check Python
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: Python 3 is not installed${NC}"
    exit 1
fi

# Check Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}Error: Node.js is not installed${NC}"
    exit 1
fi

# Check TiUP
if ! command -v tiup &> /dev/null; then
    echo -e "${YELLOW}Warning: TiUP is not installed or not in PATH${NC}"
    echo "The application will start but may not work correctly without TiUP"
fi

echo -e "${YELLOW}Setting up backend...${NC}"
cd "$PROJECT_ROOT/backend"

# Setup Python virtual environment
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv venv
fi

source venv/bin/activate

# Install Python dependencies
if [ ! -f "venv/.installed" ]; then
    echo "Installing Python dependencies..."
    pip install -q --upgrade pip
    pip install -q -r requirements.txt
    touch venv/.installed
else
    echo "Python dependencies already installed"
fi

# Copy env file
if [ ! -f ".env" ]; then
    cp .env.example .env
fi

echo -e "${GREEN}Backend setup complete!${NC}"

echo -e "${YELLOW}Setting up frontend...${NC}"
cd "$PROJECT_ROOT/frontend"

# Install Node.js dependencies
if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install
else
    echo "Node.js dependencies already installed"
fi

echo -e "${GREEN}Frontend setup complete!${NC}"

# Create a cleanup function
cleanup() {
    echo -e "\n${YELLOW}Shutting down...${NC}"
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    exit 0
}

trap cleanup SIGINT SIGTERM

echo ""
echo "======================================"
echo -e "${GREEN}Starting TiUP Visualizer...${NC}"
echo "======================================"

# Start backend
echo -e "${YELLOW}Starting backend on http://localhost:8000${NC}"
cd "$PROJECT_ROOT/backend"
source venv/bin/activate
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 > /tmp/tiup-visualizer-backend.log 2>&1 &
BACKEND_PID=$!

# Wait for backend to start
echo "Waiting for backend to start..."
for i in {1..30}; do
    if curl -s http://localhost:8000/health > /dev/null 2>&1; then
        echo -e "${GREEN}Backend started successfully!${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}Backend failed to start. Check logs at /tmp/tiup-visualizer-backend.log${NC}"
        cleanup
    fi
    sleep 1
done

# Start frontend
echo -e "${YELLOW}Starting frontend on http://localhost:5173${NC}"
cd "$PROJECT_ROOT/frontend"
npm run dev > /tmp/tiup-visualizer-frontend.log 2>&1 &
FRONTEND_PID=$!

# Wait for frontend to start
echo "Waiting for frontend to start..."
sleep 3

echo ""
echo "======================================"
echo -e "${GREEN}TiUP Visualizer is running!${NC}"
echo "======================================"
echo ""
echo -e "Frontend: ${GREEN}http://localhost:5173${NC}"
echo -e "Backend API: ${GREEN}http://localhost:8000${NC}"
echo -e "API Docs: ${GREEN}http://localhost:8000/docs${NC}"
echo ""
echo "Backend logs: /tmp/tiup-visualizer-backend.log"
echo "Frontend logs: /tmp/tiup-visualizer-frontend.log"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Wait for user interrupt
wait
