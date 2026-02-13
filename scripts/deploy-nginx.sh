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
#   --port PORT      Backend port (default: 8000)
#   --user USER      System user to run the service (default: current user)
#   --help           Show this help
#
# Examples:
#   ./deploy-nginx.sh                            # Deploy at /tiup-visualizer
#   ./deploy-nginx.sh --prefix /my-app           # Deploy at /my-app
#   ./deploy-nginx.sh --prefix /tools/tiup       # Deploy at /tools/tiup
#   ./deploy-nginx.sh --prefix /tiup --port 8001 # Custom port
#   ./deploy-nginx.sh --user www-data            # Run as www-data user
# ======================================================

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Defaults
PATH_PREFIX="/tiup-visualizer"
BACKEND_PORT="8000"
RUN_USER="$USER"

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
        --user)
            RUN_USER="$2"
            shift 2
            ;;
        --help)
            head -23 "$0" | tail -18
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
echo -e "Run as user:  ${GREEN}${RUN_USER}${NC}"
echo ""

BUILD_DIR=$(cd "$(dirname "$0")" && pwd)
APP_NAME="tiup-visualizer"
DEPLOY_DIR="/var/www${PATH_PREFIX}"
SERVICE_NAME="tiup-visualizer"
NGINX_SITE_NAME="$SERVICE_NAME"

# ---- Preflight checks ----
if ! command -v nginx &> /dev/null; then
    echo -e "${RED}Error: Nginx is not installed${NC}"
    echo "Install with: sudo apt install nginx  (Debian/Ubuntu)"
    echo "          or: sudo yum install nginx   (CentOS/RHEL)"
    exit 1
fi

if [ ! -f "$BUILD_DIR/tiup-visualizer" ]; then
    echo -e "${RED}Error: tiup-visualizer binary not found. Run this script from the build directory.${NC}"
    exit 1
fi

if ! id "$RUN_USER" &>/dev/null; then
    echo -e "${RED}Error: User '${RUN_USER}' does not exist${NC}"
    exit 1
fi

# ---- Stop existing service if running (to avoid "Text file busy") ----
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo -e "${YELLOW}Stopping running service ${SERVICE_NAME}...${NC}"
    sudo systemctl stop "$SERVICE_NAME"
    sleep 1
fi

# ---- Deploy files ----
echo -e "${YELLOW}Deploying files to ${DEPLOY_DIR}...${NC}"

sudo mkdir -p "$DEPLOY_DIR"
sudo mkdir -p "$DEPLOY_DIR/logs"
sudo chown "$RUN_USER":"$(id -gn "$RUN_USER")" "$DEPLOY_DIR/logs"

# Copy binary (remove first to avoid "Text file busy" if still held)
sudo rm -f "$DEPLOY_DIR/tiup-visualizer"
sudo cp "$BUILD_DIR/tiup-visualizer" "$DEPLOY_DIR/tiup-visualizer"
sudo chmod +x "$DEPLOY_DIR/tiup-visualizer"

# Copy config.yaml (only if not already deployed, to preserve user edits)
if [ ! -f "$DEPLOY_DIR/config.yaml" ]; then
    if [ -f "$BUILD_DIR/config.yaml" ]; then
        sudo cp "$BUILD_DIR/config.yaml" "$DEPLOY_DIR/config.yaml"
    fi
fi

# Copy frontend static files (for nginx to serve directly)
sudo rm -rf "$DEPLOY_DIR/static"
sudo cp -r "$BUILD_DIR/static" "$DEPLOY_DIR/static"

echo -e "${GREEN}Files deployed!${NC}"

# ---- Generate Nginx config from template ----
echo -e "${YELLOW}Configuring Nginx...${NC}"

if [ ! -f "$BUILD_DIR/nginx.conf.template" ]; then
    echo -e "${RED}Error: nginx.conf.template not found in build directory${NC}"
    exit 1
fi

if [ ! -f "$BUILD_DIR/nginx.upstream.template" ]; then
    echo -e "${RED}Error: nginx.upstream.template not found in build directory${NC}"
    exit 1
fi

# Generate upstream config (goes into http{} level via conf.d)
sed \
    -e "s|__BACKEND_PORT__|${BACKEND_PORT}|g" \
    "$BUILD_DIR/nginx.upstream.template" > /tmp/"$NGINX_SITE_NAME"-upstream.conf

# Generate location-only config (goes inside an existing server{} block)
sed \
    -e "s|__PATH_PREFIX__|${PATH_PREFIX}|g" \
    -e "s|__BACKEND_PORT__|${BACKEND_PORT}|g" \
    -e "s|__STATIC_DIR__|${DEPLOY_DIR}/static|g" \
    "$BUILD_DIR/nginx.conf.template" > /tmp/"$NGINX_SITE_NAME"-locations.conf

# Install upstream to conf.d (works on both Debian and RHEL)
if [ -d /etc/nginx/conf.d ]; then
    if [ -f /etc/nginx/conf.d/"$NGINX_SITE_NAME".conf ]; then
        echo -e "${YELLOW}Removing old conf.d/${NGINX_SITE_NAME}.conf...${NC}"
        sudo rm -f /etc/nginx/conf.d/"$NGINX_SITE_NAME".conf
    fi
    sudo cp /tmp/"$NGINX_SITE_NAME"-upstream.conf /etc/nginx/conf.d/"$NGINX_SITE_NAME"-upstream.conf
    echo -e "${GREEN}Upstream config installed to conf.d/${NGINX_SITE_NAME}-upstream.conf${NC}"
else
    sudo mkdir -p /etc/nginx/conf.d
    sudo cp /tmp/"$NGINX_SITE_NAME"-upstream.conf /etc/nginx/conf.d/"$NGINX_SITE_NAME"-upstream.conf
fi

# Install location blocks into the default server{} block
if [ -d /etc/nginx/default.d ]; then
    sudo cp /tmp/"$NGINX_SITE_NAME"-locations.conf /etc/nginx/default.d/"$NGINX_SITE_NAME".conf
    echo -e "${GREEN}Location config installed to default.d/${NGINX_SITE_NAME}.conf${NC}"

    if [ -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME" ]; then
        sudo rm -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME"
        sudo rm -f /etc/nginx/sites-available/"$NGINX_SITE_NAME"
    fi
elif [ -d /etc/nginx/sites-available ]; then
    sudo mkdir -p /etc/nginx/snippets
    sudo cp /tmp/"$NGINX_SITE_NAME"-locations.conf /etc/nginx/snippets/"$NGINX_SITE_NAME".conf
    echo -e "${GREEN}Location config installed to snippets/${NGINX_SITE_NAME}.conf${NC}"

    DEFAULT_SITE="/etc/nginx/sites-available/default"
    INCLUDE_LINE="include /etc/nginx/snippets/${NGINX_SITE_NAME}.conf;"
    if [ -f "$DEFAULT_SITE" ] && ! grep -qF "$INCLUDE_LINE" "$DEFAULT_SITE"; then
        echo -e "${YELLOW}Adding include directive to ${DEFAULT_SITE}...${NC}"
        sudo sed -i "/^[[:space:]]*server[[:space:]]*{/,/^[[:space:]]*}/ {
            /^[[:space:]]*}/ i\\    ${INCLUDE_LINE}
        }" "$DEFAULT_SITE"
        echo -e "${GREEN}Include directive added to default site config${NC}"
    fi

    if [ -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME" ]; then
        sudo rm -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME"
        sudo rm -f /etc/nginx/sites-available/"$NGINX_SITE_NAME"
        echo -e "${YELLOW}Removed old sites-available/${NGINX_SITE_NAME}${NC}"
    fi
else
    echo -e "${RED}Error: Cannot find /etc/nginx/default.d or /etc/nginx/sites-available${NC}"
    exit 1
fi

# Clean up temp files
rm -f /tmp/"$NGINX_SITE_NAME"-upstream.conf /tmp/"$NGINX_SITE_NAME"-locations.conf

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

RUN_USER_HOME=$(eval echo "~$RUN_USER")
TIUP_BIN_DIR="$RUN_USER_HOME/.tiup/bin"

if [ ! -f "$TIUP_BIN_DIR/tiup" ]; then
    echo -e "${YELLOW}Warning: tiup not found at ${TIUP_BIN_DIR}/tiup${NC}"
    echo -e "${YELLOW}The service user '${RUN_USER}' may not have tiup installed.${NC}"
fi

cat > /tmp/"$SERVICE_NAME".service << EOF
[Unit]
Description=TiUP Visualizer Backend (${PATH_PREFIX})
After=network.target

[Service]
Type=simple
User=$RUN_USER
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/tiup-visualizer
Restart=on-failure
RestartSec=10
Environment="PATH=${TIUP_BIN_DIR}:/usr/local/bin:/usr/bin:/bin"
Environment="HOME=${RUN_USER_HOME}"
Environment="LISTEN_ADDR=127.0.0.1:${BACKEND_PORT}"
Environment="ROOT_PATH=${PATH_PREFIX}"

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
sleep 2

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
