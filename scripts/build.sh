#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Build Script"
echo "======================================"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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

# ---- Check prerequisites ----
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Install from: https://go.dev/dl/"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo -e "${RED}Error: Node.js is not installed${NC}"
    exit 1
fi

# ---- Build frontend ----
echo -e "${YELLOW}Building frontend...${NC}"
cd "$PROJECT_ROOT/frontend"

if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm ci
fi

echo "Building frontend for production (base: ${BASE_PATH}/)..."
BASE_PATH="${BASE_PATH}/" npm run build

echo -e "${GREEN}Frontend build complete!${NC}"

# ---- Build Go backend ----
echo -e "${YELLOW}Building Go backend (static binary)...${NC}"
cd "$PROJECT_ROOT/backend-go"

# Copy frontend build to static/ for embedding
rm -rf static
cp -r "$PROJECT_ROOT/frontend/dist" static

# Build static binary (no libc dependency)
CGO_ENABLED=0 go build -ldflags="-s -w" -o tiup-visualizer .

BINARY_SIZE=$(du -h tiup-visualizer | cut -f1)
echo -e "${GREEN}Backend build complete! Binary size: ${BINARY_SIZE}${NC}"

# ---- Create deployment package ----
echo -e "${YELLOW}Creating deployment package...${NC}"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Copy binary
cp "$PROJECT_ROOT/backend-go/tiup-visualizer" "$BUILD_DIR/"

# Copy config
cp "$PROJECT_ROOT/backend-go/config.yaml.example" "$BUILD_DIR/config.yaml"

# Copy frontend build (for nginx to serve directly)
cp -r "$PROJECT_ROOT/frontend/dist" "$BUILD_DIR/static"

# Copy nginx templates and deploy script
cp "$PROJECT_ROOT/nginx.conf.template" "$BUILD_DIR/"
cp "$PROJECT_ROOT/nginx.upstream.template" "$BUILD_DIR/"
cp "$PROJECT_ROOT/scripts/deploy-nginx.sh" "$BUILD_DIR/"
chmod +x "$BUILD_DIR/deploy-nginx.sh"
chmod +x "$BUILD_DIR/tiup-visualizer"

echo -e "${GREEN}Build complete!${NC}"
echo ""
echo "Build directory: $BUILD_DIR"
echo "Build path prefix: $BASE_PATH"
echo ""
echo "Deployment options:"
echo ""
echo "  1. Direct run (single binary, simplest):"
echo "     cd $BUILD_DIR"
echo "     ./tiup-visualizer"
echo ""
echo "  2. Nginx reverse proxy (recommended for multi-site):"
echo "     cd $BUILD_DIR"
echo "     sudo ./deploy-nginx.sh --prefix $BASE_PATH"
echo ""
echo "  3. Systemd service (standalone, no nginx):"
echo "     sudo cp the binary to /usr/local/bin/"
echo "     Create a systemd unit pointing to the binary"
echo ""
