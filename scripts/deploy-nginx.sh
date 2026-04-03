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

# ---- Ensure full PATH (critical when launched from nohup/systemd/cron) ----
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# ---- Detect OS/distro ----
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="${ID}"
        OS_ID_LIKE="${ID_LIKE:-}"
    elif [ -f /etc/redhat-release ]; then
        OS_ID="rhel"
        OS_ID_LIKE="rhel"
    elif [ -f /etc/debian_version ]; then
        OS_ID="debian"
        OS_ID_LIKE="debian"
    else
        OS_ID="unknown"
        OS_ID_LIKE=""
    fi
}

is_debian_like() {
    case "$OS_ID" in
        debian|ubuntu|linuxmint|pop|kali) return 0 ;;
    esac
    case "$OS_ID_LIKE" in
        *debian*|*ubuntu*) return 0 ;;
    esac
    return 1
}

is_rhel_like() {
    case "$OS_ID" in
        rhel|centos|fedora|rocky|almalinux|tencentos|anolis|openeuler|kylin) return 0 ;;
    esac
    case "$OS_ID_LIKE" in
        *rhel*|*fedora*|*centos*) return 0 ;;
    esac
    return 1
}

detect_os
echo "Detected OS: ${OS_ID} (like: ${OS_ID_LIKE:-none})"

# ---- Detect init system ----
has_systemd() {
    command -v systemctl &>/dev/null && systemctl list-units &>/dev/null 2>&1
}

# ---- Defaults ----
PATH_PREFIX="/tiup-visualizer"
BACKEND_PORT="8000"
# Fallback: USER may be empty in nohup/systemd context; use logname or whoami
RUN_USER="${USER:-$(logname 2>/dev/null || whoami 2>/dev/null || echo root)}"

# ---- Parse arguments ----
while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix) PATH_PREFIX="$2"; shift 2 ;;
        --port)   BACKEND_PORT="$2"; shift 2 ;;
        --user)   RUN_USER="$2"; shift 2 ;;
        --help)   head -23 "$0" | tail -18; exit 0 ;;
        *) echo -e "${RED}Unknown option: $1${NC}"; echo "Use --help for usage"; exit 1 ;;
    esac
done

# Normalize prefix
PATH_PREFIX="/${PATH_PREFIX#/}"
PATH_PREFIX="${PATH_PREFIX%/}"
if [ "$PATH_PREFIX" = "" ]; then
    echo -e "${RED}Error: prefix cannot be empty${NC}"
    exit 1
fi

echo "======================================"
echo "TiUP Visualizer - Nginx Deployment"
echo "======================================"
echo ""
echo -e "OS:           ${GREEN}${OS_ID}${NC}"
echo -e "Path prefix:  ${GREEN}${PATH_PREFIX}${NC}"
echo -e "Backend port: ${GREEN}${BACKEND_PORT}${NC}"
echo -e "Run as user:  ${GREEN}${RUN_USER}${NC}"
echo ""

BUILD_DIR=$(cd "$(dirname "$0")" && pwd)
SERVICE_NAME="tiup-visualizer"
NGINX_SITE_NAME="$SERVICE_NAME"
DEPLOY_DIR="/var/www${PATH_PREFIX}"
SUDOERS_FILE="/etc/sudoers.d/tiup-visualizer"

# ---- Install sudoers rules (idempotent) ----
# Allows the service user to run deploy commands and systemd-run without a password,
# which is required for the self-update runner to work correctly.
install_sudoers() {
    local user="$1"
    local content
    content="# TiUP Visualizer - auto-update permissions
${user} ALL=(ALL) NOPASSWD: /usr/bin/systemd-run
${user} ALL=(ALL) NOPASSWD: /bin/systemctl
${user} ALL=(ALL) NOPASSWD: /usr/bin/systemctl
${user} ALL=(ALL) NOPASSWD: /usr/sbin/nginx
${user} ALL=(ALL) NOPASSWD: /usr/bin/nginx
${user} ALL=(ALL) NOPASSWD: /bin/cp
${user} ALL=(ALL) NOPASSWD: /bin/rm
${user} ALL=(ALL) NOPASSWD: /bin/mkdir
${user} ALL=(ALL) NOPASSWD: /bin/chown
${user} ALL=(ALL) NOPASSWD: /bin/chmod
${user} ALL=(ALL) NOPASSWD: /usr/bin/cp
${user} ALL=(ALL) NOPASSWD: /usr/bin/rm
${user} ALL=(ALL) NOPASSWD: /usr/bin/mkdir
${user} ALL=(ALL) NOPASSWD: /usr/bin/chown
${user} ALL=(ALL) NOPASSWD: /usr/bin/chmod
"
    echo "$content" > /tmp/tiup-visualizer-sudoers
    chmod 0440 /tmp/tiup-visualizer-sudoers
    if visudo -cf /tmp/tiup-visualizer-sudoers &>/dev/null; then
        sudo cp /tmp/tiup-visualizer-sudoers "$SUDOERS_FILE"
        echo -e "${GREEN}Sudoers rules installed for user '${user}' → ${SUDOERS_FILE}${NC}"
    else
        echo -e "${YELLOW}Warning: sudoers syntax check failed, skipping install${NC}"
    fi
    rm -f /tmp/tiup-visualizer-sudoers
}

if [ -d /etc/sudoers.d ]; then
    install_sudoers "$RUN_USER"
else
    echo -e "${YELLOW}Warning: /etc/sudoers.d not found, skipping sudoers install${NC}"
fi

# ---- Preflight checks ----
if ! command -v nginx &>/dev/null; then
    echo -e "${RED}Error: Nginx is not installed${NC}"
    if is_debian_like; then
        echo "Install with: sudo apt install -y nginx"
    elif is_rhel_like; then
        echo "Install with: sudo yum install -y nginx  (or dnf)"
    else
        echo "Please install nginx for your distribution."
    fi
    exit 1
fi

if [ ! -f "$BUILD_DIR/tiup-visualizer" ]; then
    echo -e "${RED}Error: tiup-visualizer binary not found in $BUILD_DIR${NC}"
    exit 1
fi

if ! id "$RUN_USER" &>/dev/null; then
    echo -e "${RED}Error: User '${RUN_USER}' does not exist${NC}"
    exit 1
fi

# ---- Stop existing service ----
if has_systemd; then
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        echo -e "${YELLOW}Stopping systemd service ${SERVICE_NAME}...${NC}"
        sudo systemctl stop "$SERVICE_NAME"
        sleep 1
    fi
else
    echo -e "${YELLOW}Warning: systemd not available, skipping service stop${NC}"
fi

# ---- Deploy files ----
echo -e "${YELLOW}Deploying files to ${DEPLOY_DIR}...${NC}"
sudo mkdir -p "$DEPLOY_DIR" "$DEPLOY_DIR/logs"
sudo chown "$RUN_USER":"$(id -gn "$RUN_USER")" "$DEPLOY_DIR/logs"

sudo rm -f "$DEPLOY_DIR/tiup-visualizer"
sudo cp "$BUILD_DIR/tiup-visualizer" "$DEPLOY_DIR/tiup-visualizer"
sudo chmod +x "$DEPLOY_DIR/tiup-visualizer"

[ -f "$BUILD_DIR/version" ] && sudo cp "$BUILD_DIR/version" "$DEPLOY_DIR/version"

if [ ! -f "$DEPLOY_DIR/config.yaml" ] && [ -f "$BUILD_DIR/config.yaml" ]; then
    sudo cp "$BUILD_DIR/config.yaml" "$DEPLOY_DIR/config.yaml"
fi

sudo rm -rf "$DEPLOY_DIR/static"
sudo cp -r "$BUILD_DIR/static" "$DEPLOY_DIR/static"
echo -e "${GREEN}Files deployed!${NC}"

# ---- Generate Nginx configs ----
echo -e "${YELLOW}Configuring Nginx...${NC}"

for tpl in nginx.conf.template nginx.upstream.template; do
    if [ ! -f "$BUILD_DIR/$tpl" ]; then
        echo -e "${RED}Error: $tpl not found in $BUILD_DIR${NC}"
        exit 1
    fi
done

sed -e "s|__BACKEND_PORT__|${BACKEND_PORT}|g" \
    "$BUILD_DIR/nginx.upstream.template" > /tmp/"$NGINX_SITE_NAME"-upstream.conf

sed -e "s|__PATH_PREFIX__|${PATH_PREFIX}|g" \
    -e "s|__BACKEND_PORT__|${BACKEND_PORT}|g" \
    -e "s|__STATIC_DIR__|${DEPLOY_DIR}/static|g" \
    "$BUILD_DIR/nginx.conf.template" > /tmp/"$NGINX_SITE_NAME"-locations.conf

# Install upstream config into conf.d (universal, exists on both Debian & RHEL)
sudo mkdir -p /etc/nginx/conf.d
sudo rm -f /etc/nginx/conf.d/"$NGINX_SITE_NAME".conf  # remove legacy single-file config
sudo cp /tmp/"$NGINX_SITE_NAME"-upstream.conf /etc/nginx/conf.d/"$NGINX_SITE_NAME"-upstream.conf
echo -e "${GREEN}Upstream config → /etc/nginx/conf.d/${NGINX_SITE_NAME}-upstream.conf${NC}"

# Install location blocks — strategy differs by distro
if [ -d /etc/nginx/default.d ]; then
    # RHEL/CentOS style: location snippets go into default.d/
    sudo cp /tmp/"$NGINX_SITE_NAME"-locations.conf /etc/nginx/default.d/"$NGINX_SITE_NAME".conf
    echo -e "${GREEN}Location config → /etc/nginx/default.d/${NGINX_SITE_NAME}.conf${NC}"
    # Remove any leftover Debian-style config
    sudo rm -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME" \
               /etc/nginx/sites-available/"$NGINX_SITE_NAME" \
               /etc/nginx/snippets/"$NGINX_SITE_NAME".conf 2>/dev/null || true

elif [ -d /etc/nginx/sites-available ]; then
    # Debian/Ubuntu style: use snippets + include in default site
    sudo mkdir -p /etc/nginx/snippets
    sudo cp /tmp/"$NGINX_SITE_NAME"-locations.conf /etc/nginx/snippets/"$NGINX_SITE_NAME".conf
    echo -e "${GREEN}Location config → /etc/nginx/snippets/${NGINX_SITE_NAME}.conf${NC}"

    DEFAULT_SITE="/etc/nginx/sites-available/default"
    INCLUDE_LINE="include /etc/nginx/snippets/${NGINX_SITE_NAME}.conf;"
    if [ -f "$DEFAULT_SITE" ] && ! grep -qF "$INCLUDE_LINE" "$DEFAULT_SITE"; then
        echo -e "${YELLOW}Injecting include into ${DEFAULT_SITE}...${NC}"
        sudo sed -i "/^[[:space:]]*server[[:space:]]*{/,/^[[:space:]]*}/ {
            /^[[:space:]]*}/ i\\    ${INCLUDE_LINE}
        }" "$DEFAULT_SITE"
        echo -e "${GREEN}Include added to default site${NC}"
    fi
    sudo rm -f /etc/nginx/sites-enabled/"$NGINX_SITE_NAME" \
               /etc/nginx/sites-available/"$NGINX_SITE_NAME" 2>/dev/null || true

else
    # Fallback: create conf.d only, put locations there wrapped in a server block hint
    echo -e "${YELLOW}Warning: neither default.d nor sites-available found.${NC}"
    echo -e "${YELLOW}Installing locations into conf.d as snippet — you may need to include it manually.${NC}"
    sudo cp /tmp/"$NGINX_SITE_NAME"-locations.conf /etc/nginx/conf.d/"$NGINX_SITE_NAME"-locations.conf
fi

rm -f /tmp/"$NGINX_SITE_NAME"-upstream.conf /tmp/"$NGINX_SITE_NAME"-locations.conf

# ---- Validate & reload Nginx ----
if sudo nginx -t; then
    echo -e "${GREEN}Nginx config valid${NC}"
else
    echo -e "${RED}Nginx config test failed — aborting${NC}"
    exit 1
fi
sudo nginx -s reload 2>/dev/null || sudo systemctl reload nginx
echo -e "${GREEN}Nginx reloaded${NC}"

# ---- Systemd service ----
RUN_USER_HOME=$(eval echo "~$RUN_USER")
TIUP_BIN_DIR="$RUN_USER_HOME/.tiup/bin"
[ -f "$TIUP_BIN_DIR/tiup" ] || echo -e "${YELLOW}Warning: tiup not found at ${TIUP_BIN_DIR}/tiup${NC}"

if has_systemd; then
    echo -e "${YELLOW}Creating systemd service: ${SERVICE_NAME}...${NC}"
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
Environment="PATH=${TIUP_BIN_DIR}:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
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
    echo -e "${GREEN}Systemd service restarted${NC}"
else
    # Fallback: supervisord or plain nohup
    if command -v supervisorctl &>/dev/null && [ -d /etc/supervisor ]; then
        echo -e "${YELLOW}systemd not available, using supervisord...${NC}"
        SUPERVISOR_CONF="/etc/supervisor/conf.d/${SERVICE_NAME}.conf"
        cat > /tmp/"${SERVICE_NAME}"-supervisor.conf << EOF
[program:${SERVICE_NAME}]
command=${DEPLOY_DIR}/tiup-visualizer
directory=${DEPLOY_DIR}
user=${RUN_USER}
autostart=true
autorestart=true
environment=LISTEN_ADDR="127.0.0.1:${BACKEND_PORT}",ROOT_PATH="${PATH_PREFIX}",HOME="${RUN_USER_HOME}",PATH="${TIUP_BIN_DIR}:/usr/local/bin:/usr/bin:/bin"
stdout_logfile=${DEPLOY_DIR}/logs/tiup-visualizer.log
stderr_logfile=${DEPLOY_DIR}/logs/tiup-visualizer.log
EOF
        sudo cp /tmp/"${SERVICE_NAME}"-supervisor.conf "$SUPERVISOR_CONF"
        sudo supervisorctl reread && sudo supervisorctl update
        sudo supervisorctl restart "$SERVICE_NAME" || sudo supervisorctl start "$SERVICE_NAME"
        echo -e "${GREEN}Supervisord service started${NC}"
    else
        echo -e "${YELLOW}Neither systemd nor supervisord available — starting with nohup...${NC}"
        pkill -f "$DEPLOY_DIR/tiup-visualizer" 2>/dev/null || true
        sleep 1
        export LISTEN_ADDR="127.0.0.1:${BACKEND_PORT}"
        export ROOT_PATH="${PATH_PREFIX}"
        export HOME="${RUN_USER_HOME}"
        nohup "$DEPLOY_DIR/tiup-visualizer" >> "$DEPLOY_DIR/logs/tiup-visualizer.log" 2>&1 &
        echo -e "${GREEN}Service started via nohup (pid=$!)${NC}"
    fi
fi

# ---- Verify ----
echo ""
echo -e "${YELLOW}Waiting for backend to start...${NC}"
sleep 2
if curl -sf "http://127.0.0.1:${BACKEND_PORT}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}Backend is running!${NC}"
else
    echo -e "${RED}Backend health check failed — check logs${NC}"
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
