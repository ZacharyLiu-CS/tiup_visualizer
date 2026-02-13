GO_BIN ?= go
FRONTEND_DIR := frontend
BACKEND_DIR := backend-go
STATIC_DIR := $(BACKEND_DIR)/static
BINARY := $(BACKEND_DIR)/tiup-visualizer
BUILD_DIR := build

# Path prefix for sub-path deployment (default: /tiup-visualizer)
# Override with: BASE_PATH=/my-app make build
BASE_PATH ?= /tiup-visualizer

.PHONY: all clean build frontend backend backend-only dev dev-backend dev-frontend package upload ensure-static

all: build

# Ensure static directory exists with a placeholder (required by go:embed in static.go)
ensure-static:
	@mkdir -p $(STATIC_DIR)
	@test -n "$$(ls -A $(STATIC_DIR) 2>/dev/null)" || touch $(STATIC_DIR)/.gitkeep

# Build frontend and copy to backend static directory for embedding
frontend: ensure-static
	@echo "==> Building frontend (BASE_PATH=$(BASE_PATH))..."
	cd $(FRONTEND_DIR) && npm ci && BASE_PATH=$(BASE_PATH)/ npm run build
	@rm -rf $(STATIC_DIR)/*
	@cp -r $(FRONTEND_DIR)/dist/* $(STATIC_DIR)/
	@echo "==> Frontend built and copied to $(STATIC_DIR)/"

# Build Go binary with embedded static files (static linked, no libc dependency)
backend: frontend
	@echo "==> Building Go backend (static binary)..."
	cd $(BACKEND_DIR) && CGO_ENABLED=0 $(GO_BIN) build -ldflags="-s -w" -o tiup-visualizer .
	@echo "==> Built: $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Build backend only (assumes static/ already has frontend assets)
backend-only: ensure-static
	@echo "==> Building Go backend (static binary)..."
	cd $(BACKEND_DIR) && CGO_ENABLED=0 $(GO_BIN) build -ldflags="-s -w" -o tiup-visualizer .
	@echo "==> Built: $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Full production build: frontend + backend + deployment package
build: backend package
	@echo "==> Build complete! Deployment package in $(BUILD_DIR)/"

# Create deployment package in build/
package:
	@echo "==> Creating deployment package..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)
	@cp $(BINARY) $(BUILD_DIR)/
	@cp $(BACKEND_DIR)/config.yaml.example $(BUILD_DIR)/config.yaml
	@cp -r $(FRONTEND_DIR)/dist $(BUILD_DIR)/static
	@cp nginx.conf.template $(BUILD_DIR)/
	@cp nginx.upstream.template $(BUILD_DIR)/
	@cp scripts/deploy-nginx.sh $(BUILD_DIR)/
	@chmod +x $(BUILD_DIR)/deploy-nginx.sh
	@chmod +x $(BUILD_DIR)/tiup-visualizer
	@echo "==> Deployment package created in $(BUILD_DIR)/"

# Development: start backend (API :8000) + frontend Vite dev server (:5173) concurrently
# Access http://localhost:5173 for full-stack dev with hot reload
# Ctrl+C stops both processes
dev: ensure-static
	@echo "==> Starting backend (localhost:8000) and frontend dev server (localhost:5173)..."
	@echo "==> Open http://localhost:5173 in your browser"
	@trap 'kill 0' EXIT; \
		(cd $(BACKEND_DIR) && $(GO_BIN) run .) & \
		(cd $(FRONTEND_DIR) && npm run dev) & \
		wait

# Development: run backend only (API server on :8000)
dev-backend: ensure-static
	cd $(BACKEND_DIR) && $(GO_BIN) run .

# Development: run frontend only (Vite dev server on :5173, proxies API to :8000)
dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

# Upload build to artifact repository
upload:
	@./scripts/upload.sh

clean:
	rm -f $(BINARY)
	rm -rf $(STATIC_DIR)
	rm -rf $(BUILD_DIR)
