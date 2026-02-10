#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Build Script"
echo "======================================"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
BUILD_DIR="$PROJECT_ROOT/build"

echo -e "${YELLOW}Building backend...${NC}"
cd "$PROJECT_ROOT/backend"

# Create virtual environment if not exists
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install dependencies
echo "Installing Python dependencies..."
pip install -q --upgrade pip
pip install -q -r requirements.txt

echo -e "${GREEN}Backend build complete!${NC}"

echo -e "${YELLOW}Building frontend...${NC}"
cd "$PROJECT_ROOT/frontend"

# Install Node.js dependencies
if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install
fi

# Build frontend
echo "Building frontend for production..."
npm run build

echo -e "${GREEN}Frontend build complete!${NC}"

# Create build directory
echo -e "${YELLOW}Creating deployment package...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Copy backend files
cp -r "$PROJECT_ROOT/backend/app" "$BUILD_DIR/"
cp "$PROJECT_ROOT/backend/requirements.txt" "$BUILD_DIR/"
cp "$PROJECT_ROOT/backend/.env.example" "$BUILD_DIR/.env"

# Copy frontend build
cp -r "$PROJECT_ROOT/frontend/dist" "$BUILD_DIR/static"

# Copy deployment script
cat > "$BUILD_DIR/start.sh" << 'EOF'
#!/bin/bash

set -e

echo "Starting TiUP Visualizer..."

# Check if virtual environment exists
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install dependencies
echo "Installing dependencies..."
pip install -q --upgrade pip
pip install -q -r requirements.txt

# Start the server
echo "Starting FastAPI server on http://0.0.0.0:8000"
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
EOF

chmod +x "$BUILD_DIR/start.sh"

# Create systemd service file
cat > "$BUILD_DIR/tiup-visualizer.service" << EOF
[Unit]
Description=TiUP Visualizer Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$BUILD_DIR
ExecStart=$BUILD_DIR/venv/bin/python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Create README
cat > "$BUILD_DIR/README.md" << 'EOF'
# TiUP Visualizer - Deployment

## Quick Start

### Method 1: Direct Run
```bash
./start.sh
```

### Method 2: Systemd Service (Production)
```bash
# Copy service file
sudo cp tiup-visualizer.service /etc/systemd/system/

# Edit the service file to update paths
sudo nano /etc/systemd/system/tiup-visualizer.service

# Reload systemd
sudo systemctl daemon-reload

# Start service
sudo systemctl start tiup-visualizer

# Enable on boot
sudo systemctl enable tiup-visualizer

# Check status
sudo systemctl status tiup-visualizer
```

## Access
- API: http://localhost:8000
- Web Interface: http://localhost:8000 (served via static files)

## Configuration
Edit `.env` file to change settings.

## Requirements
- Python 3.8+
- tiup command available in PATH
- Proper permissions to execute tiup commands
EOF

echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Build directory: $BUILD_DIR"
echo ""
echo "To deploy:"
echo "  1. Copy $BUILD_DIR to your server"
echo "  2. Run: ./start.sh"
echo ""
echo "Or use systemd service for production deployment."
