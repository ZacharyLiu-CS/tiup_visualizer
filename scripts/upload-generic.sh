#!/bin/bash

set -e

echo "======================================"
echo "TiUP Visualizer - Upload Script"
echo "======================================"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
BUILD_DIR="$PROJECT_ROOT/build"

# Upload configuration
USERNAME="zacharyzliu"
TOKEN="${UPLOAD_TOKEN:-}"
REPO="${UPLOAD_REPO:-easygraph-tiup-visualizer}"

# Check if build directory exists
if [ ! -d "$BUILD_DIR" ]; then
    echo -e "${RED}Error: Build directory not found at $BUILD_DIR${NC}"
    echo "Please run 'make build' first"
    exit 1
fi

# Read version from version file
VERSION_FILE="$PROJECT_ROOT/version"
if [ ! -f "$VERSION_FILE" ]; then
    echo -e "${RED}Error: version file not found at $VERSION_FILE${NC}"
    exit 1
fi
VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
ZIP_FILENAME="tiup-visualizer-${VERSION}.tar.gz"
ZIP_PATH="$PROJECT_ROOT/$ZIP_FILENAME"

# Create zip file
echo -e "${YELLOW}Compressing build directory...${NC}"
cd "$PROJECT_ROOT"
tar -czf "$ZIP_FILENAME" --transform 's/^build/tiup-visualizer/' build

if [ ! -f "$ZIP_PATH" ]; then
    echo -e "${RED}Error: Failed to create zip file${NC}"
    exit 1
fi

echo -e "${GREEN}Zip file created: $ZIP_FILENAME${NC}"

# Upload file
echo -e "${YELLOW}Uploading to repository...${NC}"
UPLOAD_URL="https://mirrors.tencent.com/repository/generic/${REPO}/"

if curl --request PUT -u "${USERNAME}:${TOKEN}" \
    --url "$UPLOAD_URL" \
    --upload-file "$ZIP_FILENAME"; then

    echo ""
    echo -e "${GREEN}Upload successful!${NC}"
    echo ""
    echo "Upload URL: $UPLOAD_URL$ZIP_FILENAME"
    echo ""
else
    echo -e "${RED}Upload failed${NC}"
    exit 1
fi

#  Clean up local zip file
rm "$ZIP_PATH"
echo -e "${GREEN}Local zip file removed${NC}"

echo -e "${GREEN}Done!${NC}"
