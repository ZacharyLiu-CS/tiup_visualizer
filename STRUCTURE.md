# TiUP Visualizer - Project Structure

```
tiup-visualizer/
│
├── 📄 start.sh                    # 一键启动脚本 (开发模式)
├── 📄 start-prod.sh               # 一键启动脚本 (生产模式)
├── 📄 README.md                   # 项目文档
├── 📄 QUICKSTART.md               # 快速开始指南
├── 📄 .gitignore                  # Git 忽略文件
├── 📄 Dockerfile                  # Docker 镜像构建
├── 📄 docker-compose.yml          # Docker Compose 配置
│
├── 📁 backend/                    # FastAPI 后端
│   ├── 📄 requirements.txt        # Python 依赖
│   ├── 📄 .env.example            # 环境变量模板
│   │
│   └── 📁 app/                    # 应用主目录
│       ├── 📄 __init__.py
│       ├── 📄 main.py             # FastAPI 应用入口
│       │
│       ├── 📁 api/                # API 路由
│       │   ├── 📄 __init__.py
│       │   └── 📄 routes.py       # REST API 端点定义
│       │
│       ├── 📁 core/               # 核心配置
│       │   ├── 📄 __init__.py
│       │   └── 📄 config.py       # 应用配置
│       │
│       ├── 📁 models/             # 数据模型
│       │   ├── 📄 __init__.py
│       │   └── 📄 cluster.py      # Pydantic 模型定义
│       │
│       └── 📁 services/           # 业务逻辑
│           ├── 📄 __init__.py
│           └── 📄 tiup_service.py # TiUP 命令封装和解析
│
├── 📁 frontend/                   # Vue 3 前端
│   ├── 📄 package.json            # Node.js 依赖
│   ├── 📄 vite.config.js          # Vite 构建配置
│   ├── 📄 index.html              # HTML 入口
│   │
│   ├── 📁 public/                 # 静态资源
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
    ├── 📄 build.sh                # 生产构建脚本
    ├── 📄 dev.sh                  # 开发环境设置
    └── 📄 start-dev.sh            # 开发服务器启动
```

## 核心文件说明

### 启动脚本
- **start.sh** - 开发模式一键启动,前后端分离
- **start-prod.sh** - 生产模式一键启动,前后端合并

### 后端核心
- **app/main.py** - FastAPI 应用,路由配置,静态文件服务
- **app/services/tiup_service.py** - 核心业务逻辑:
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
TiUP Commands → Backend Service → REST API → Frontend Store → Vue Components
     ↓               ↓                ↓            ↓              ↓
  tiup CLI      Parse Output     JSON Response   Pinia       UI Display
```

## API 端点

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/clusters` | 获取所有集群列表 |
| GET | `/api/v1/clusters/{name}` | 获取指定集群详情 |
| GET | `/api/v1/hosts` | 获取所有物理主机 |
| GET | `/api/v1/hosts/{ip}/clusters` | 获取主机上的集群 |
| GET | `/health` | 健康检查 |
| GET | `/docs` | API 交互文档 |
