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

# Path prefix for sub-path deployment (default: /tiup-visualizer)
# Override with: BASE_PATH=/my-app bash scripts/build.sh
BASE_PATH="${BASE_PATH:-/tiup-visualizer}"
# Normalize: ensure leading slash, no trailing slash
BASE_PATH="/${BASE_PATH#/}"
BASE_PATH="${BASE_PATH%/}"

echo -e "Build path prefix: ${GREEN}${BASE_PATH}${NC}"
echo ""

echo -e "${YELLOW}Building backend...${NC}"
cd "$PROJECT_ROOT/backend"

# Create virtual environment if not exists
if ! conda env list | grep -q "^env_tiup_visualizer "; then
    echo "Creating conda virtual environment..."
    conda create --name env_tiup_visualizer python=3.8 -y
fi

# Activate virtual environment
eval "$(conda shell.bash hook)"
conda activate env_tiup_visualizer

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

# Build frontend with sub-path base
# Vite uses BASE_PATH env to set the base URL for all assets
echo "Building frontend for production (base: ${BASE_PATH}/)..."
BASE_PATH="${BASE_PATH}/" npm run build

echo -e "${GREEN}Frontend build complete!${NC}"

# Create build directory
echo -e "${YELLOW}Creating deployment package...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Copy backend files
cp -r "$PROJECT_ROOT/backend/app" "$BUILD_DIR/"
cp "$PROJECT_ROOT/backend/requirements.txt" "$BUILD_DIR/"
cp "$PROJECT_ROOT/backend/.env.example" "$BUILD_DIR/.env"
cp "$PROJECT_ROOT/backend/config.yaml.example" "$BUILD_DIR/config.yaml"

# Copy frontend build
cp -r "$PROJECT_ROOT/frontend/dist" "$BUILD_DIR/static"

# Copy nginx templates and deploy script
cp "$PROJECT_ROOT/nginx.conf.template" "$BUILD_DIR/"
cp "$PROJECT_ROOT/nginx.upstream.template" "$BUILD_DIR/"
cp "$PROJECT_ROOT/scripts/deploy-nginx.sh" "$BUILD_DIR/"
chmod +x "$BUILD_DIR/deploy-nginx.sh"

# Copy deployment script (direct run without nginx)
cat > "$BUILD_DIR/start.sh" << 'EOF'
#!/bin/bash

set -e

echo "Starting TiUP Visualizer..."

# Check if virtual environment exists
if ! conda env list | grep -q "^env_tiup_visualizer "; then
    echo "Creating conda virtual environment..."
    conda create --name env_tiup_visualizer python=3.8 -y
fi

# Activate virtual environment
eval "$(conda shell.bash hook)"
conda activate env_tiup_visualizer

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
ExecStart=/bin/bash -c 'eval "\$(conda shell.bash hook)" && conda activate env_tiup_visualizer && python -m uvicorn app.main:app --host 0.0.0.0 --port 8000'
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Build directory: $BUILD_DIR"
echo "Build path prefix: $BASE_PATH"
echo ""
echo "Deployment options:"
echo ""
echo "  1. Nginx reverse proxy (recommended for multi-site):"
echo "     cd $BUILD_DIR"
echo "     ./deploy-nginx.sh --prefix $BASE_PATH"
echo ""
echo "  2. Direct run (standalone, no nginx):"
echo "     cd $BUILD_DIR"
echo "     ./start.sh"
echo ""
echo "  3. Systemd service (standalone, no nginx):"
echo "     sudo cp $BUILD_DIR/tiup-visualizer.service /etc/systemd/system/"
echo "     sudo systemctl daemon-reload && sudo systemctl start tiup-visualizer"
echo ""
