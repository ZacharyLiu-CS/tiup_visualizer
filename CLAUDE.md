# CLAUDE.md — TiUP Visualizer 开发指南

本文件为 Claude Code 提供项目上下文、开发规范与常见问题的修复模式，每次进入本项目时请先阅读此文件。

---

## 项目概述

**TiUP Visualizer** 是一个基于 Web 的 TiUP 集群管理可视化工具，提供集群部署、监控、TiKV 数据查询、Region 负载均衡等功能。

- **后端**：Go 1.22+，单二进制，无 CGO 依赖
- **前端**：Vue 3 + Vite，打包后嵌入 Go 二进制（`go:embed`）
- **部署**：支持 Docker / Nginx + Systemd / 直接运行二进制

---

## 目录结构

```
tiup-visualizer/
├── Makefile                          # 所有构建/开发命令入口
├── backend-go/                       # Go 后端（所有服务）
│   ├── main.go                       # 入口：加载配置、初始化日志、启动 Server
│   ├── routes.go                     # HTTP Handler 注册 + 大部分 handler 实现
│   ├── cluster_create_service.go     # 创建集群：文件上传、部署、历史配置管理
│   ├── tiup_service.go               # TiUP CLI 封装：TTL 缓存、解析输出
│   ├── tikv_service.go               # TiKV 客户端池（TTL 清理）
│   ├── balancer_service.go           # Region 均衡：异步任务 + SSE 推送
│   ├── pdctl_service.go              # pd-ctl 命令白名单执行
│   ├── auth.go                       # JWT HS256 认证
│   ├── config.go                     # YAML 配置加载（支持环境变量覆盖）
│   ├── update_service.go             # 自动更新（下载新版本并重启）
│   ├── terminal_service.go           # WebSocket 终端
│   ├── models.go                     # 共用数据结构
│   ├── config.yaml.example           # 配置模板
│   └── create_cluster_history/       # 历史配置文件存储目录（运行时自动创建）
├── frontend/                         # Vue 3 前端
│   ├── src/
│   │   ├── views/                    # 页面级组件（HomeView, LoginView）
│   │   ├── components/               # 功能组件（CreateClusterModal 等）
│   │   ├── services/api.js           # Axios API 封装
│   │   └── stores/                   # Pinia 状态（auth）
│   └── dist/                         # 构建产物（gitignored，打包时复制到 backend-go/static/）
└── scripts/
    └── deploy-nginx.sh               # 生产部署脚本（Nginx + Systemd）
```

---

## 开发命令

```bash
# 同时启动前后端（前端 :5173，后端 :8000，支持热重载）
make dev

# 仅启动后端
make dev-backend

# 仅启动前端
make dev-frontend

# 生产构建（输出到 build/）
make build

# 清理构建产物
make clean
```

### 生产部署

```bash
make build
cd build/
./deploy-nginx.sh --prefix /tiup-visualizer --port 8000
```

`deploy-nginx.sh` 会自动：
- 创建部署目录 `/var/www/tiup-visualizer/`（含 `logs/` 和 `create_cluster_history/`）
- 配置 Nginx（兼容 Debian/RHEL）
- 创建 Systemd 服务并启动

---

## 架构要点

### 请求链路

```
HTTP 请求
  → s.mux（net/http ServeMux）
  → requireAuth 中间件（JWT 验证，写入 X-Username header）
  → Handler 函数（routes.go 或各 service 文件）
  → Service 方法（业务逻辑）
  → writeJSON / writeError（统一 JSON 响应）
```

### 日志规范（必须遵守）

使用 `log/slog`，**每个 handler 必须覆盖以下三个节点**：

| 节点 | 级别 | 示例 |
|------|------|------|
| 收到请求 | `slog.Info` | `slog.Info("ClusterCreate: deploy request", "cluster", name, "user", r.Header.Get("X-Username"))` |
| 操作失败 | `slog.Error` 或 `slog.Warn` | `slog.Error("ClusterCreate: deploy failed", "cluster", name, "error", err)` |
| 操作成功 | `slog.Info` | `slog.Info("ClusterCreate: deploy success", "cluster", name)` |

**破坏性操作**（destroy、clean、delete）用 `slog.Warn` 代替 `slog.Info`。

**禁止记录**密码、Token 等敏感信息。

### API 响应格式

```go
// 成功
writeJSON(w, http.StatusOK, map[string]any{"key": value})

// 失败
writeError(w, http.StatusInternalServerError, "错误描述")
// → {"detail": "错误描述"}
```

---

## 新增功能开发流程

### 1. 后端新增 Endpoint

**步骤：**

1. **在 `models.go` 定义结构体**（Request/Response）

2. **在对应 service 文件实现业务逻辑**
   - 命名规范：`func (s *XxxService) DoSomething(...) (result, error)`
   - 路径安全检查：`strings.Contains(name, "..") || strings.Contains(name, "/")` → 返回 400

3. **在 `routes.go` 或 service 文件中实现 Handler**
   ```go
   func (s *Server) handleXxxYyy(w http.ResponseWriter, r *http.Request) {
       param := r.PathValue("param")    // URL 路径参数
       query := r.URL.Query().Get("q")  // 查询参数
       user := r.Header.Get("X-Username") // 当前登录用户（来自 requireAuth）

       // 1. 收到请求日志
       slog.Info("Xxx: yyy request", "param", param, "user", user)

       // 2. 执行业务
       result, err := s.xxxService.DoSomething(param)
       if err != nil {
           slog.Error("Xxx: yyy failed", "param", param, "error", err)
           writeError(w, http.StatusInternalServerError, err.Error())
           return
       }

       // 3. 成功日志 + 响应
       slog.Info("Xxx: yyy success", "param", param)
       writeJSON(w, http.StatusOK, map[string]any{"result": result})
   }
   ```

4. **在 `routes.go` 的 `registerRoutes()` 中注册路由**
   ```go
   s.mux.HandleFunc("GET "+prefix+"/xxx/yyy", s.requireAuth(s.handleXxxYyy))
   ```

5. **在 `frontend/src/services/api.js` 中添加 API 调用**

6. **在对应 Vue 组件中调用 API 并处理响应**

### 2. 前端新增操作按钮

参考 `CreateClusterModal.vue` 的模式：

```javascript
async handleAction() {
  try {
    const res = await xxxAPI.doSomething(params)
    this.result = res.data
  } catch (e) {
    const msg = e.response?.data?.detail || e.message || '操作失败'
    this.error = msg
  }
}
```

---

## 创建集群功能详解

### 核心文件

- `backend-go/cluster_create_service.go`：所有业务逻辑 + HTTP Handler
- `frontend/src/components/CreateClusterModal.vue`：前端 UI

### API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/cluster-create/deploy` | 部署集群（JSON 粘贴或文件上传） |
| `GET` | `/api/v1/cluster-create/history` | 列出历史配置 |
| `GET` | `/api/v1/cluster-create/config/{name}` | 查看/下载配置（`?action=download`） |
| `DELETE` | `/api/v1/cluster-create/config/{name}` | 删除配置 |

### 部署流程

1. 前端提交表单（JSON 模式 or multipart 文件上传）
2. `handleClusterCreateDeploy` 解析请求，记录日志
3. `DeployCluster` / `UploadAndDeploy` 将配置保存到 `create_cluster_history/{name}.yaml`
4. 执行 `tiup cluster deploy {name} {version} {config} --user {user} -y`（30 分钟超时）
5. 返回部署输出

### 目录依赖

`create_cluster_history/` 目录由两处自动创建：
- **运行时**：`NewClusterCreateService(execDir)` 调用 `os.MkdirAll`（首次启动时自动创建）
- **部署时**：`deploy-nginx.sh` 在创建部署目录时显式 `mkdir -p ... create_cluster_history/`

> **关键**：若在 `/var/www/tiup-visualizer/` 下手动部署（不通过 `deploy-nginx.sh`），需确保服务进程对该目录有写权限，或首次启动时二进制有权限自动创建该目录。

---

## TiUP 命令封装规范

| 操作 | 超时 | 备注 |
|------|------|------|
| `tiup cluster list` | 30s | TTL 30s 缓存 |
| `tiup cluster display {name}` | 30s | TTL 30s 缓存，per-cluster |
| `tiup cluster start/stop` | 5min | — |
| `tiup cluster clean` | 5min | — |
| `tiup cluster destroy` | 10min | — |
| `tiup cluster deploy` | 30min | — |

缓存失效：所有写操作（start/stop/clean/destroy/deploy）完成后应调用 `s.tiup.InvalidateCache()`。

---

## 安全规范

1. **路径遍历防护**：所有接收用户文件名/集群名的接口必须检查 `..`、`/`、`\`
   ```go
   if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
       writeError(w, http.StatusBadRequest, "Invalid name")
       return
   }
   ```

2. **命令注入防护**：PDCtl 使用白名单（见 `routes.go handlePDCtlExec`）

3. **文件上传限制**：multipart 最大 32MB

4. **生产必改**：`config.yaml` 中的 `secret_key` 和 `password` 默认值不得用于生产

---

## 常见问题 & 修复模式

### 问题：新机器部署后，上传配置文件报 `no such file or directory`

**原因**：`create_cluster_history/` 目录不存在，且进程无权创建（例如 `/var/www/` 下）。

**修复**：
- `deploy-nginx.sh` 已包含 `mkdir -p "$DEPLOY_DIR/create_cluster_history"`，重新执行部署脚本即可
- 或手动创建：`sudo mkdir -p /var/www/tiup-visualizer/create_cluster_history && sudo chown $USER /var/www/tiup-visualizer/create_cluster_history`

### 问题：操作按钮点击后日志无输出

**原因**：Handler 缺少 `slog.Info` 调用。

**修复**：参考"日志规范"部分，为每个 handler 添加请求收到、失败、成功三个日志点。

### 问题：前端构建产物没有更新

**原因**：`go:embed` 在编译时嵌入，需重新 `make build`。

**修复**：`make clean && make build`

### 问题：集群操作缓存未刷新

**原因**：写操作后未调用 `s.tiup.InvalidateCache()`。

**修复**：在对应 handler 成功路径末尾添加缓存失效调用。

---

## 配置文件说明

```yaml
# backend-go/config.yaml
auth:
  username: "admin"
  password: "easygraph"            # 生产环境必须修改
  secret_key: "change-me"          # 生产环境必须修改
  token_expire_hours: 24

logging:
  log_dir: "./logs"                # 相对于可执行文件目录
  log_level: "INFO"               # DEBUG / INFO / WARN / ERROR
  max_file_size_mb: 10
  backup_count: 5

server:
  listen_addr: "0.0.0.0:8000"
  api_prefix: "/api/v1"
```

环境变量覆盖（优先级高于配置文件）：
- `TIUP_VISUALIZER_CONFIG`：指定配置文件路径
- `LISTEN_ADDR`：监听地址
- `ROOT_PATH`：API 前缀（用于 Nginx 子路径部署）

---

## 代码质量规范

- 使用 `any` 替代 `interface{}`（Go 1.18+）
- `map[string]any{}` 而非 `map[string]interface{}{}`
- 错误包装：`fmt.Errorf("操作失败: %w", err)`（保留错误链）
- 长字符串截断输出：`truncate(output, 500)`（防止日志溢出）
