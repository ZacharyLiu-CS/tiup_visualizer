# TiUP Visualizer - Project Structure

```
tiup-visualizer/
│

├── 📄 Makefile                    # Build, dev, deploy commands
├── 📄 README.md                   # 项目文档
├── 📄 QUICKSTART.md               # 快速开始指南
├── 📄 .gitignore                  # Git 忽略文件
├── 📄 Dockerfile                  # Docker 镜像构建（单二进制）
├── 📄 Dockerfile.nginx            # Docker 镜像构建（Nginx 模式）
├── 📄 docker-compose.yml          # Docker Compose 配置
├── 📄 docker-compose.nginx.yml    # Docker Compose Nginx 配置
├── 📄 nginx.conf.template         # Nginx location 配置模板
├── 📄 nginx.upstream.template     # Nginx upstream 配置模板
├── 📄 supervisord.conf            # Supervisor 配置（Docker Nginx 模式）
│
├── 📁 backend-go/                 # Go 后端（单二进制，嵌入前端）
│   ├── 📄 main.go                 # 应用入口，HTTP 服务器
│   ├── 📄 routes.go               # API 路由注册
│   ├── 📄 tiup_service.go         # TiUP 命令执行和解析
│   ├── 📄 auth.go                 # 认证（JWT）
│   ├── 📄 config.go               # 配置加载
│   ├── 📄 middleware.go           # HTTP 中间件
│   ├── 📄 models.go               # 数据模型
│   ├── 📄 terminal.go             # WebSocket 终端
│   ├── 📄 static.go               # go:embed 静态文件服务
│   ├── 📄 config.yaml.example     # 配置文件模板
│   └── 📁 static/                 # 前端构建产物（gitignored，由 make frontend 生成）
│
├── 📁 frontend/                   # Vue 3 前端
│   ├── 📄 package.json            # Node.js 依赖
│   ├── 📄 vite.config.js          # Vite 构建配置
│   ├── 📄 index.html              # HTML 入口
│   │
│   └── 📁 src/                    # 源代码
│       ├── 📄 main.js             # Vue 应用入口
│       ├── 📄 App.vue             # 根组件
│       ├── 📄 style.css           # 全局样式
│       │
│       ├── 📁 components/         # Vue 组件
│       │   ├── 📄 HostCard.vue             # 物理主机卡片
│       │   ├── 📄 ClusterCard.vue          # 集群卡片
│       │   ├── 📄 ClusterDetailModal.vue   # 集群详情弹窗
│       │   └── 📄 ConnectionLines.vue      # 连接线组件
│       │
│       ├── 📁 views/              # 页面视图
│       │   └── 📄 HomeView.vue    # 主页面
│       │
│       ├── 📁 services/           # API 服务
│       │   └── 📄 api.js          # Axios 配置和 API 调用
│       │
│       └── 📁 stores/             # 状态管理
│           └── 📄 cluster.js      # Pinia Store (集群状态)
│
└── 📁 scripts/                    # 工具脚本
    ├── 📄 deploy-nginx.sh         # Nginx 反向代理部署脚本
    ├── 📄 mock-tiup.sh            # TiUP 模拟数据脚本
    └── 📄 upload.sh               # 构建包上传脚本
```

## 核心文件说明

### Makefile 命令
- **`make dev`** - 开发模式，同时启动后端 (:8000) + 前端 (:5173)，支持热更新
- **`make dev-backend`** - 仅启动后端
- **`make dev-frontend`** - 仅启动前端（需后端已运行）
- **`make build`** - 完整构建：前端 + 后端 + 部署包
- **`make frontend`** - 仅构建前端
- **`make backend-only`** - 仅构建后端（假设 static/ 已存在）
- **`make clean`** - 清理构建产物

### 后端核心
- **main.go** - HTTP 服务器启动，路由配置
- **tiup_service.go** - 核心业务逻辑:
  - 执行 `tiup cluster list` 命令
  - 执行 `tiup cluster display <name>` 命令
  - 解析命令输出
  - 聚合主机和集群数据

### 前端核心
- **src/views/HomeView.vue** - 主页面:
  - 布局管理 (上下结构)
  - 交互逻辑 (点击、高亮)
  - 连接线计算和绘制
  
- **src/stores/cluster.js** - 状态管理:
  - 集群数据
  - 主机数据
  - 选择状态
  - API 调用

- **src/components/** - 可复用组件:
  - HostCard - 主机卡片显示
  - ClusterCard - 集群卡片显示
  - ClusterDetailModal - 详情弹窗
  - ConnectionLines - SVG 连接线

## 数据流

```
TiUP Commands → Go Backend → REST API → Frontend Store → Vue Components
     ↓              ↓             ↓            ↓              ↓
  tiup CLI     Parse Output  JSON Response   Pinia       UI Display
```

## API 端点

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/clusters` | 获取所有集群列表 |
| GET | `/api/v1/clusters/{name}` | 获取指定集群详情 |
| GET | `/api/v1/hosts` | 获取所有物理主机 |
| GET | `/api/v1/hosts/{ip}/clusters` | 获取主机上的集群 |
| GET | `/health` | 健康检查 |
