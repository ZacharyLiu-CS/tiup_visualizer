# ============================================
# Dockerfile - Single static binary deployment
# Produces a minimal image with just the Go binary
# ============================================

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go backend with embedded frontend
FROM golang:1.22-alpine AS backend-builder

WORKDIR /app

COPY backend-go/go.mod backend-go/go.sum ./
RUN go mod download

COPY backend-go/ ./

# Copy frontend build into static/ for embedding
COPY --from=frontend-builder /app/dist ./static/

# Build static binary
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o tiup-visualizer .

# Stage 3: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache bash openssh-client

WORKDIR /app

COPY --from=backend-builder /app/tiup-visualizer .
COPY backend-go/config.yaml.example ./config.yaml

EXPOSE 8000

CMD ["./tiup-visualizer"]
