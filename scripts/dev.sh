#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Development Setup"
echo "======================================"

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)

# Setup backend
echo "Setting up backend..."
cd "$PROJECT_ROOT/backend"

if ! conda env list | grep -q "^env_tiup_visualizer "; then
    conda create --name env_tiup_visualizer python=3.8 -y
fi

eval "$(conda shell.bash hook)"
conda activate env_tiup_visualizer
pip install --upgrade pip
pip install -r requirements.txt

# Copy env file if not exists
if [ ! -f ".env" ]; then
    cp .env.example .env
fi

echo "Backend setup complete!"

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
echo '  Backend:  cd backend && eval "$(conda shell.bash hook)" && conda activate env_tiup_visualizer && python -m uvicorn app.main:app --reload'
echo "  Frontend: cd frontend && npm run dev"
echo ""
