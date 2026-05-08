# TiUP Visualizer

A web-based visualization and management tool for TiUP cluster operations, built with Go and Vue 3.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)
![Node](https://img.shields.io/badge/node-18+-green.svg)
![Vue](https://img.shields.io/badge/Vue-3.4-42b883.svg)

## 📖 Overview

TiUP Visualizer 提供直观的 Web 界面，用于可视化和管理通过 TiUP 部署的 TiKV 集群。交互式仪表板实时展示物理主机与集群的拓扑关系，并集成了多种运维工具。

---

## ✨ 功能一览

### 🖥️ 集群可视化
- 物理主机与集群的拓扑展示，通过连线直观显示主机与集群的对应关系
- 集群卡片实时显示健康状态（healthy / partial / unhealthy）
- 点击主机或集群，高亮显示关联关系

### 📋 集群详情
- 完整组件列表（TiKV、PD、Grafana、Prometheus 等）
- 每个组件显示 IP、端口、状态、数据目录、部署目录
- 一键访问 Dashboard URL 和 Grafana URL

### ⚡ 集群运维操作
- **Start / Stop**：启动或停止集群
- **Clean**：清理集群数据（`--all`）
- **Destroy**：销毁集群（双重确认防误操作）

### ⚙️ 在线 Config 编辑（新功能）
点击集群卡片上的 **Config** 按钮，打开配置编辑面板：
- **在线编辑** `meta.yaml`，支持完整拓扑配置修改
- **Save & Reload**：保存修改并执行 `tiup cluster reload`，使配置生效
- **Export as Template**：将当前集群拓扑导出为可直接用于部署新集群的 YAML 模板（仅包含 topology 部分，不含 `user`/`tidb_version` 等元数据），自动保存到"创建集群历史"中
- **Discard Changes**：一键放弃所有未保存的修改，恢复原始状态
- 修改检测：有未保存修改时显示 `Modified` 标记

### 🚀 创建集群
- 右侧滑动面板，支持粘贴 YAML、上传文件、从历史选择配置三种方式
- 实时 SSE 流式显示部署输出
- 历史配置管理，可查看、删除已保存的拓扑文件
- 支持从 Config 编辑面板导出的模板直接创建新集群

### 📜 日志查看与 AI 分析
- 在集群详情页查看各组件日志
- **AI Analysis**：自动构建分析提示词，下载日志尾部，跳转到 [Knot AI](https://knot.woa.com) 进行智能分析

### 🔧 KV2Graph（图数据查询）
右侧工具面板的 **KV2Graph** Tab：
- 支持通过集群名称或自定义 PD 地址连接
- Key 精确查询 / Prefix 前缀扫描
- Hex 转换工具
- PD 地址历史记录（localStorage 持久化）

### ⚖️ Region Balancer（Region 均衡）
右侧工具面板的 **Region Balancer** Tab：
- 分析各 TiKV Store 的 Region 分布情况
- 创建均衡任务（peer 迁移 + leader 转移）
- 任务队列管理：pending / running / completed / cancelled / failed 状态
- SSE 实时进度推送 + 3s 轮询双保险
- 服务端执行，浏览器关闭后任务继续运行

### 🎛️ PD Ctl
右侧工具面板的 **PD Ctl** Tab：
- 执行 `tiup ctl pd` 命令（白名单命令过滤）
- 支持集群名称或自定义 PD 地址

### 💻 WebTerminal
- 内嵌 Web 终端，支持 Top / Bottom / Left / Right / Float 五种布局模式

### 📊 服务端日志
- 查看 TiUP Visualizer 自身的运行日志

### 🔄 自动更新
- 检查最新版本并一键在线升级

---

## 🤖 AI Analysis

AI 分析功能基于 **Knot** 平台实现，支持一键复制集群日志分析提示词并跳转到 Knot AI 对话界面进行智能分析。

- **首页 AI Analysis 按钮**：直接打开 Knot AI 对话页面
- **详情页 AI Analysis 按钮**：自动复制包含集群信息的分析提示词到剪贴板，并下载日志尾部数据，然后打开 Knot AI 对话页面，粘贴提示词即可开始分析

### Knot 搭建教程

如需自行搭建或配置 Knot AI 工作区，请参考内部教程：

👉 **[Knot 搭建指南](https://iwiki.woa.com/p/4018690792)**

---

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

### Production Build & Deploy

```bash
make build && ./build/deploy-nginx.sh
```

---

## 🗂️ Project Structure

```
tiup-visualizer/
├── Makefile                        # Build, dev, deploy commands
├── backend-go/                     # Go backend (single binary)
│   ├── main.go
│   ├── routes.go                   # HTTP routes & handlers
│   ├── tiup_service.go             # TiUP CLI wrapper
│   ├── cluster_config_service.go   # Config get/save/reload/export
│   ├── cluster_create_service.go   # Cluster deploy + history
│   ├── balancer_service.go         # Region balancer task queue
│   ├── pdctl_service.go            # PD Ctl command executor
│   ├── tikv_service.go             # TiKV RawKV client (KV2Graph)
│   ├── auth.go                     # JWT authentication
│   ├── config.go                   # App configuration
│   ├── static.go                   # Embedded frontend via go:embed
│   └── static/                     # Frontend build output (gitignored)
├── frontend/                       # Vue 3 frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── ClusterCard.vue         # Cluster card (Details + Config buttons)
│   │   │   ├── ClusterConfigModal.vue  # Online config editor panel
│   │   │   ├── ClusterDetailModal.vue  # Cluster detail view
│   │   │   ├── CreateClusterModal.vue  # Cluster creation panel
│   │   │   ├── GraphToolsPanel.vue     # KV2Graph / Balancer / PD Ctl tabs
│   │   │   ├── HostCard.vue            # Physical host card
│   │   │   ├── WebTerminal.vue         # Embedded terminal
│   │   │   ├── ConfirmDialog.vue       # Reusable confirm dialog
│   │   │   └── ConnectionLines.vue     # SVG host-cluster connection lines
│   │   ├── views/
│   │   │   └── HomeView.vue            # Main dashboard
│   │   ├── stores/
│   │   │   ├── cluster.js              # Cluster & host state (Pinia)
│   │   │   └── auth.js                 # Auth state (Pinia)
│   │   └── services/
│   │       └── api.js                  # Axios API client
│   └── package.json
└── scripts/                        # Deployment and utility scripts
```

---

## 📡 API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clusters` | List all clusters |
| GET | `/api/v1/clusters/{name}` | Cluster detail |
| GET | `/api/v1/clusters/{name}/config` | Get cluster meta.yaml |
| POST | `/api/v1/clusters/{name}/config` | Save cluster config |
| POST | `/api/v1/clusters/{name}/reload` | Reload cluster config |
| POST | `/api/v1/clusters/{name}/config/export-template` | Export topology as deploy template |
| POST | `/api/v1/clusters/{name}/start` | Start cluster |
| POST | `/api/v1/clusters/{name}/stop` | Stop cluster |
| POST | `/api/v1/clusters/{name}/clean` | Clean cluster data |
| POST | `/api/v1/clusters/{name}/destroy` | Destroy cluster |
| GET | `/api/v1/hosts` | List all physical hosts |
| GET | `/api/v1/hosts/{ip}/clusters` | Clusters on a host |
| GET | `/api/v1/logs/{cluster}/{component}/{file}` | View/download log file |
| POST | `/api/v1/cluster-create/deploy` | Start async cluster deploy |
| GET | `/api/v1/cluster-create/deploy/stream` | SSE deploy output stream |
| GET | `/api/v1/cluster-create/history` | List saved topology configs |
| GET | `/api/v1/cluster-create/config/{name}` | Get saved topology config |
| DELETE | `/api/v1/cluster-create/config/{name}` | Delete saved config |
| POST | `/api/v1/balancer/analyze` | Analyze region distribution |
| POST | `/api/v1/balancer/tasks` | Create balance task |
| GET | `/api/v1/balancer/tasks` | List balance tasks |
| POST | `/api/v1/balancer/tasks/{id}/cancel` | Cancel task |
| GET | `/api/v1/balancer/events` | SSE balance task events |
| POST | `/api/v1/pdctl/exec` | Execute PD Ctl command |
| GET | `/ws/terminal` | WebSocket terminal |

---

## ⚙️ Configuration

### 用户名和密码

认证配置在 `backend-go/config.yaml` 中：

```yaml
auth:
  username: "admin"
  password: "easygraph"
  secret_key: "tiup-visualizer-secret-key-change-me-in-production"
  token_expire_hours: 24
```

**开发模式**：直接编辑 `backend-go/config.yaml`，重启后端生效。

**Nginx 部署模式**：
```bash
sudo vim /var/www/tiup-visualizer/config.yaml
sudo systemctl restart tiup-visualizer
```

### Makefile 命令

```bash
make dev            # 同时启动后端+前端（热更新）
make dev-backend    # 仅后端
make dev-frontend   # 仅前端
make build          # 完整构建（前端+后端+部署包）
make clean          # 清理构建产物
```

---

## 🔧 Deployment

### Nginx 反向代理部署（推荐）

```bash
make build
./build/deploy-nginx.sh

# 自定义路径前缀和端口
./build/deploy-nginx.sh --prefix /tools/tiup --port 8001
```

脚本自动完成：文件部署到 `/var/www/<prefix>/`、Nginx 配置生成与加载、systemd 服务创建与启动。

### 管理服务

```bash
sudo systemctl status tiup-visualizer
sudo systemctl restart tiup-visualizer
sudo journalctl -u tiup-visualizer -f
```

### Requirements

- Go 1.22+
- Node.js 18+
- TiUP installed and available in PATH

---

## 🛠️ Technologies

**Backend**: Go 1.22+ · net/http · go:embed · gopkg.in/yaml.v3 · golang-jwt

**Frontend**: Vue 3 · Pinia · Axios · Vite · xterm.js

---

## License

MIT
