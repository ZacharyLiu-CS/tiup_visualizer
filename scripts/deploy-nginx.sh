#!/bin/bash

set -e

# ======================================================
# TiUP Visualizer - Nginx Deployment Script
# Run from within the build directory (alongside this file)
#
# Usage:
#   ./deploy-nginx.sh [OPTIONS]
#
# Options:
#   --prefix PATH    URL path prefix (default: /tiup-visualizer)
#   --port PORT      Backend uvicorn port (default: 8000)
#   --help           Show this help
#
# Examples:
#   ./deploy-nginx.sh                            # Deploy at /tiup-visualizer
#   ./deploy-nginx.sh --prefix /my-app           # Deploy at /my-app
#   ./deploy-nginx.sh --prefix /tools/tiup       # Deploy at /tools/tiup
#   ./deploy-nginx.sh --prefix /tiup --port 8001 # Custom port
# ======================================================

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Defaults
PATH_PREFIX="/tiup-visualizer"
BACKEND_PORT="8000"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix)
            PATH_PREFIX="$2"
            shift 2
            ;;
        --port)
            BACKEND_PORT="$2"
            shift 2
            ;;
        --help)
            head -22 "$0" | tail -17
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage"
            exit 1
            ;;
    esac
done

# Normalize prefix: ensure leading slash, no trailing slash
PATH_PREFIX="/${PATH_PREFIX#/}"
PATH_PREFIX="${PATH_PREFIX%/}"

if [ "$PATH_PREFIX" = "" ]; then
    echo -e "${RED}Error: prefix cannot be empty. Use '/' is not supported, deploy at a sub-path like /tiup-visualizer${NC}"
    exit 1
fi

echo "======================================"
echo "TiUP Visualizer - Nginx Deployment"
echo "======================================"
echo ""
echo -e "Path prefix:  ${GREEN}${PATH_PREFIX}${NC}"
echo -e "Backend port: ${GREEN}${BACKEND_PORT}${NC}"
echo ""

BUILD_DIR=$(cd "$(dirname "$0")" && pwd)
APP_NAME="tiup-visualizer"
DEPLOY_DIR="/var/www${PATH_PREFIX}"
SERVICE_NAME="tiup-visualizer-$(echo "$PATH_PREFIX" | tr '/' '-' | sed 's/^-//')"
NGINX_SITE_NAME="$SERVICE_NAME"

# ---- Preflight checks ----
if ! command -v nginx &> /dev/null; then
    echo -e "${RED}Error: Nginx is not installed${NC}"
    echo "Install with: sudo apt install nginx"
    exit 1
fi

if ! command -v conda &> /dev/null; then
    echo -e "${RED}Error: conda is not installed${NC}"
    exit 1
fi

if [ ! -f "$BUILD_DIR/requirements.txt" ]; then
    echo -e "${RED}Error: requirements.txt not found. Run this script from the build directory.${NC}"
    exit 1
fi

# ---- Setup conda environment ----
echo -e "${YELLOW}Setting up Python environment...${NC}"
if ! conda env list | grep -q "^env_tiup_visualizer "; then
    conda create --name env_tiup_visualizer python=3.8 -y
fi

eval "$(conda shell.bash hook)"
conda activate env_tiup_visualizer
pip install -q --upgrade pip
pip install -q -r "$BUILD_DIR/requirements.txt"

echo -e "${GREEN}Python environment ready!${NC}"

# ---- Deploy files ----
echo -e "${YELLOW}Deploying files to ${DEPLOY_DIR}...${NC}"

sudo mkdir -p "$DEPLOY_DIR"

# Copy backend app
sudo cp -r "$BUILD_DIR/app" "$DEPLOY_DIR/"
sudo cp "$BUILD_DIR/requirements.txt" "$DEPLOY_DIR/"

if [ ! -f "$DEPLOY_DIR/.env" ]; then
    sudo cp "$BUILD_DIR/.env" "$DEPLOY_DIR/.env"
fi

# Set ROOT_PATH in .env for FastAPI to know its prefix
sudo sed -i "s|^ROOT_PATH=.*|ROOT_PATH=\"${PATH_PREFIX}\"|" "$DEPLOY_DIR/.env"

# Copy frontend static files
sudo rm -rf "$DEPLOY_DIR/static"
sudo cp -r "$BUILD_DIR/static" "$DEPLOY_DIR/static"

echo -e "${GREEN}Files deployed!${NC}"

# ---- Generate Nginx config from template ----
echo -e "${YELLOW}Configuring Nginx...${NC}"

if [ -f "$BUILD_DIR/nginx.conf.template" ]; then
    sed \
        -e "s|__PATH_PREFIX__|${PATH_PREFIX}|g" \
        -e "s|__BACKEND_PORT__|${BACKEND_PORT}|g" \
        -e "s|__STATIC_DIR__|${DEPLOY_DIR}/static|g" \
        "$BUILD_DIR/nginx.conf.template" > /tmp/"$NGINX_SITE_NAME".conf

    sudo cp /tmp/"$NGINX_SITE_NAME".conf /etc/nginx/sites-available/"$NGINX_SITE_NAME"
    sudo ln -sf /etc/nginx/sites-available/"$NGINX_SITE_NAME" /etc/nginx/sites-enabled/
else
    echo -e "${RED}Error: nginx.conf.template not found in build directory${NC}"
    exit 1
fi

# Test nginx config
if sudo nginx -t; then
    echo -e "${GREEN}Nginx config is valid!${NC}"
else
    echo -e "${RED}Nginx config test failed!${NC}"
    exit 1
fi

sudo systemctl reload nginx

# ---- Create systemd service for backend ----
echo -e "${YELLOW}Creating systemd service: ${SERVICE_NAME}...${NC}"

CONDA_PREFIX_PATH=$(conda info --base)
cat > /tmp/"$SERVICE_NAME".service << EOF
[Unit]
Description=TiUP Visualizer Backend (${PATH_PREFIX})
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$DEPLOY_DIR
ExecStart=/bin/bash -c 'eval "\$(${CONDA_PREFIX_PATH}/bin/conda shell.bash hook)" && conda activate env_tiup_visualizer && python -m uvicorn app.main:app --host 127.0.0.1 --port ${BACKEND_PORT} --workers 2'
Restart=on-failure
RestartSec=10
Environment="PATH=${CONDA_PREFIX_PATH}/bin:/usr/local/bin:/usr/bin:/bin"

[Install]
WantedBy=multi-user.target
EOF

sudo cp /tmp/"$SERVICE_NAME".service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl restart "$SERVICE_NAME"

# ---- Verify ----
echo ""
echo -e "${YELLOW}Waiting for backend to start...${NC}"
sleep 3

if curl -s "http://127.0.0.1:${BACKEND_PORT}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}Backend is running!${NC}"
else
    echo -e "${RED}Backend may not be running. Check: sudo systemctl status ${SERVICE_NAME}${NC}"
fi

echo ""
echo "======================================"
echo -e "${GREEN}Deployment complete!${NC}"
echo "======================================"
echo ""
echo -e "Web Interface: ${GREEN}http://localhost${PATH_PREFIX}/${NC}"
echo -e "API Endpoint:  ${GREEN}http://localhost${PATH_PREFIX}/api/v1${NC}"
echo -e "API Docs:      ${GREEN}http://localhost${PATH_PREFIX}/docs${NC}"
echo -e "Health Check:  ${GREEN}http://localhost${PATH_PREFIX}/health${NC}"
echo ""
echo "Manage services:"
echo "  sudo systemctl status nginx"
echo "  sudo systemctl status ${SERVICE_NAME}"
echo "  sudo systemctl restart ${SERVICE_NAME}"
echo ""
echo "Logs:"
echo "  Nginx:   /var/log/nginx/tiup-visualizer-access.log"
echo "  Backend: sudo journalctl -u ${SERVICE_NAME} -f"
echo ""
