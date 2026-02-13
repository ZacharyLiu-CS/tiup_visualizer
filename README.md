# TiUP Visualizer

A web-based visualization tool for TiUP cluster management, built with Go and Vue 3.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)
![Node](https://img.shields.io/badge/node-18+-green.svg)
![Vue](https://img.shields.io/badge/Vue-3.4-42b883.svg)

## 📖 Overview

TiUP Visualizer provides an intuitive web interface to visualize and manage your TiKV clusters deployed with TiUP. It displays physical hosts and clusters in an interactive dashboard with real-time status information.

**Key Features:**
- 🖥️ Visual representation of physical hosts
- 🔷 Interactive TiKV cluster cards
- 🔗 Connection visualization between hosts and clusters
- 📊 Detailed component information (IP, ports, status, directories)
- 🔄 Real-time data from TiUP commands
- 🎨 Modern, responsive UI with Vue 3

## 🚀 Quick Start

See [QUICKSTART.md](QUICKSTART.md) for detailed instructions.

### Development

```bash
cd tiup-visualizer
make dev
```

这会同时启动 Go 后端 (`:8000`) 和 Vite 前端开发服务器 (`:5173`)，支持前端热更新。

- 前端页面: http://localhost:5173
- 后端 API: http://localhost:8000

也可以分别启动：
```bash
make dev-backend    # 仅启动后端
make dev-frontend   # 仅启动前端（需要后端已运行）
```

## Project Structure

```
tiup-visualizer/
├── Makefile                 # Build, dev, deploy commands
├── backend-go/              # Go backend (single binary with embedded frontend)
│   ├── main.go
│   ├── routes.go
│   ├── tiup_service.go
│   ├── auth.go
│   ├── config.go
│   ├── static.go            # Embedded frontend via go:embed
│   └── static/              # Frontend build output (gitignored)
├── frontend/                # Vue 3 frontend
│   ├── src/
│   │   ├── components/      # Vue components
│   │   ├── views/           # Page views
│   │   ├── stores/          # Pinia stores
│   │   └── services/        # API services
│   └── package.json
└── scripts/                 # Deployment and utility scripts
```

## Requirements

- Go 1.22+
- Node.js 18+
- TiUP installed and available in PATH

## Quick Start

### Development

```bash
make dev
```

### Production Build

```bash
make build
```

This will:
- Build the Vue 3 frontend
- Embed it into the Go binary
- Create a deployment package in `build/`

Deploy the `build/` directory to your server and run:

```bash
cd build
./tiup-visualizer
```

### Nginx 反向代理部署（推荐用于多站点）

适用于在同一台服务器上部署多个站点的场景，通过 Nginx 反向代理将不同 URL 前缀路由到不同应用。

#### 方式一：使用部署脚本

1. 构建项目：
```bash
make build
```

2. 进入 build 目录，运行部署脚本：
```bash
cd build

# 使用默认配置（前缀 /tiup-visualizer，端口 8000）
./deploy-nginx.sh

# 自定义路径前缀
./deploy-nginx.sh --prefix /my-app

# 自定义路径前缀和后端端口
./deploy-nginx.sh --prefix /tools/tiup --port 8001
```

部署脚本会自动完成：
- 创建 conda 环境并安装依赖
- 将文件部署到 `/var/www/<prefix>/`
- 从模板生成 Nginx 配置并加载
- 创建并启动 systemd 服务

部署完成后访问：
- Web 界面：`http://your-server/tiup-visualizer/`
- API 文档：`http://your-server/tiup-visualizer/docs`

#### 方式二：使用 Docker Compose

```bash
# 使用默认路径前缀 /tiup-visualizer
docker-compose -f docker-compose.nginx.yml up -d

# 自定义路径前缀（修改 docker-compose.nginx.yml 中的 BASE_PATH）
```

`docker-compose.nginx.yml` 中可配置的参数：
```yaml
args:
  BASE_PATH: /tiup-visualizer  # 修改为你需要的路径前缀
```

#### 管理服务

```bash
# 查看 Nginx 状态
sudo systemctl status nginx

# 查看后端服务状态（服务名基于路径前缀生成）
sudo systemctl status tiup-visualizer-tiup-visualizer

# 重启后端服务
sudo systemctl restart tiup-visualizer-tiup-visualizer

# 查看后端日志
sudo journalctl -u tiup-visualizer-tiup-visualizer -f

# 查看 Nginx 日志
tail -f /var/log/nginx/tiup-visualizer-access.log
tail -f /var/log/nginx/tiup-visualizer-error.log
```

#### 多站点部署示例

在同一台机器上部署多个实例，使用不同前缀和端口：

```bash
# 实例 1
./deploy-nginx.sh --prefix /cluster-a --port 8001

# 实例 2
./deploy-nginx.sh --prefix /cluster-b --port 8002
```

### Docker Deployment

```bash
# Build and run with Docker Compose
docker-compose up -d

# Or build manually
docker build -t tiup-visualizer .
docker run -p 8000:8000 -v /root/.tiup:/root/.tiup:ro tiup-visualizer
```

## Scripts

`scripts/` 目录下提供了部署和测试相关的脚本工具。项目的构建和开发使用根目录的 `Makefile` 管理。

### Makefile 命令

```bash
# 开发模式：同时启动后端 (:8000) + 前端 (:5173)，支持热更新
make dev

# 仅启动后端
make dev-backend

# 仅启动前端（需后端已运行）
make dev-frontend

# 完整构建：前端 + 后端 + 部署包
make build

# 仅构建前端
make frontend

# 仅构建后端（假设 static/ 已存在）
make backend-only

# 上传构建包
make upload

# 清理构建产物
make clean
```

### `scripts/deploy-nginx.sh` — Nginx 反向代理部署

在服务器上配置 Nginx 反向代理 + systemd 服务，支持自定义路径前缀和端口，适合多站点共存部署。需要先运行 `make build` 生成部署包。

```bash
cd build

# 默认部署（前缀 /tiup-visualizer，端口 8000）
./deploy-nginx.sh

# 自定义路径前缀和端口
./deploy-nginx.sh --prefix /tools/tiup --port 8001

# 查看帮助
./deploy-nginx.sh --help
```

### `scripts/mock-tiup.sh` — TiUP 模拟数据

在没有真实 TiUP 环境时提供模拟数据，用于开发和测试。支持 `cluster list` 和 `cluster display` 两个子命令。

```bash
# 模拟 tiup cluster list
./scripts/mock-tiup.sh cluster list

# 模拟 tiup cluster display <cluster-name>
./scripts/mock-tiup.sh cluster display eg3-cicd-proxy
```

### `scripts/upload.sh` — 构建包上传

将 `build/` 目录打包为 `.tar.gz` 并上传到制品仓库。需要先运行 `make build` 完成构建。

```bash
# 先构建，再上传
make build
./scripts/upload.sh
```

## API Endpoints

- `GET /api/v1/clusters` - Get all clusters
- `GET /api/v1/clusters/{cluster_name}` - Get cluster details
- `GET /api/v1/hosts` - Get all physical hosts
- `GET /api/v1/hosts/{host_ip}/clusters` - Get clusters on a specific host

## Usage

1. The main page displays two sections:
   - **Top Section**: Physical hosts with server icons
   - **Bottom Section**: TiKV clusters

2. **Click on a Host**: Highlights all clusters deployed on that host with connection lines

3. **Click on a Cluster**: Opens a detailed modal showing:
   - Cluster information (type, version, URLs)
   - All components with their details (IP, ports, status, directories)
   - Highlights the physical hosts where the cluster is deployed

4. **Clear Selection**: Click the "Clear Selection" button or click the same item again

## Configuration

### 用户名和密码

认证配置在 `backend-go/config.yaml` 中：

```yaml
auth:
  username: "admin"          # 登录用户名
  password: "easygraph"      # 登录密码
  secret_key: "tiup-visualizer-secret-key-change-me-in-production"  # JWT 密钥，生产环境务必修改
  token_expire_hours: 24     # Token 过期时间（小时）
```

**开发模式 (`make dev`)**：直接编辑 `backend-go/config.yaml`，重启服务即可生效。

**生产构建 (`build/` 目录)**：`make build` 会将 `backend-go/config.yaml.example` 复制为 `build/config.yaml`。部署前直接编辑它：
```bash
vim build/config.yaml    # 修改用户名、密码、secret_key
```

**Nginx 部署模式 (`deploy-nginx.sh`)**：
- 部署前：先修改 `build/config.yaml`（或修改源文件 `backend-go/config.yaml.example` 后重新 `make build`）
- 部署后：直接修改部署目录的配置文件并重启服务：
  ```bash
  sudo vim /var/www/tiup-visualizer/config.yaml
  sudo systemctl restart tiup-visualizer
  ```

**Docker 模式**：在 `docker-compose.nginx.yml` 中挂载自定义配置文件：
```yaml
volumes:
  - ./my-config.yaml:/app/config.yaml
```

配置文件查找顺序：
1. 环境变量 `TIUP_VISUALIZER_CONFIG` 指定的路径
2. `backend-go/config.yaml`（开发模式默认）
3. `/etc/tiup-visualizer/config.yaml`

### 其他配置

`backend-go/config.yaml` 中的应用配置：
```
APP_NAME="TiUP Visualizer"
DEBUG=True
API_PREFIX="/api/v1"
```

## Technologies Used

### Backend
- **Go 1.22+**: Single static binary with embedded frontend
- **net/http**: Standard library HTTP server
- **embed**: Frontend assets embedded at compile time

### Frontend
- **Vue 3**: Progressive JavaScript framework
- **Pinia**: State management
- **Axios**: HTTP client
- **Vite**: Frontend build tool

## License

MIT
