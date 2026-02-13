GO_BIN ?= go
FRONTEND_DIR := ../frontend
STATIC_DIR := static
BINARY := tiup-visualizer

.PHONY: all clean build frontend backend dev

all: build

# Build frontend and copy to static directory
frontend:
	@echo "==> Building frontend..."
	cd $(FRONTEND_DIR) && npm ci && npm run build
	@rm -rf $(STATIC_DIR)
	@cp -r $(FRONTEND_DIR)/dist $(STATIC_DIR)
	@echo "==> Frontend built and copied to $(STATIC_DIR)/"

# Build Go binary with embedded static files (static linked, no libc dependency)
backend: frontend
	@echo "==> Building Go backend (static binary)..."
	CGO_ENABLED=0 $(GO_BIN) build -ldflags="-s -w" -o $(BINARY) .
	@echo "==> Built: $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Build for production: frontend + backend in one step
build: backend
	@echo "==> Build complete! Single binary: $(BINARY)"

# Build backend only (assumes static/ already exists)
backend-only:
	@echo "==> Building Go backend (static binary)..."
	CGO_ENABLED=0 $(GO_BIN) build -ldflags="-s -w" -o $(BINARY) .
	@echo "==> Built: $(BINARY) ($$(du -h $(BINARY) | cut -f1))"

# Development: run with go run
dev:
	$(GO_BIN) run . 

clean:
	rm -f $(BINARY)
	rm -rf $(STATIC_DIR)
