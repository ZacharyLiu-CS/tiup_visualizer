# TiUP Visualizer - 项目总结

## ✅ 已完成功能

### 后端 (FastAPI)
- ✅ 高性能 FastAPI 框架搭建
- ✅ TiUP 命令集成
  - `tiup cluster list` - 获取集群列表
  - `tiup cluster display <name>` - 获取集群详情
- ✅ 命令输出解析器
- ✅ RESTful API 设计
- ✅ CORS 跨域支持
- ✅ 静态文件服务 (生产环境)
- ✅ 健康检查端点
- ✅ 自动 API 文档 (Swagger)

### 前端 (Vue 3)
- ✅ 现代化 Vue 3 + Vite 架构
- ✅ Pinia 状态管理
- ✅ 响应式布局设计
- ✅ 物理主机可视化组件
- ✅ TiKV 集群可视化组件
- ✅ 交互式连接线绘制
- ✅ 集群详情模态窗口
- ✅ 组件状态颜色编码
- ✅ 高亮和选择功能

### 部署方式
- ✅ 开发模式一键启动 (`./start.sh`)
- ✅ 生产模式一键部署 (`./start-prod.sh`)
- ✅ Docker 支持 (Dockerfile + docker-compose)
- ✅ Systemd 服务配置
- ✅ 构建脚本 (`scripts/build.sh`)

### 文档
- ✅ 完整的 README
- ✅ 快速开始指南 (QUICKSTART.md)
- ✅ 项目结构文档 (STRUCTURE.md)
- ✅ 使用指南 (USAGE.md)
- ✅ 单元测试示例

## 📋 核心特性

### 1. 上下布局设计
```
┌─────────────────────────┐
│  物理主机 (上)          │  ← 服务器图标、IP、统计信息
├─────────────────────────┤
│  TiKV 集群 (下)         │  ← 集群图标、名称、版本
└─────────────────────────┘
```

### 2. 双向交互
- **点击主机** → 显示该主机上的所有集群
- **点击集群** → 显示集群部署的所有主机 + 详情窗口

### 3. 可视化连接
- SVG 绘制虚线连接
- 蓝色: 主机→集群
- 紫色: 集群→主机

### 4. 详细信息展示
- 集群元数据
- 组件列表 (ID、角色、主机、端口、状态、目录)
- Dashboard 和 Grafana 链接
- 状态和角色颜色编码

## 🚀 启动方式

### 最简单的方式
```bash
cd tiup-visualizer
./start.sh
```
访问: http://localhost:5173

### 生产部署
```bash
cd tiup-visualizer
./start-prod.sh
```
访问: http://localhost:8000

### Docker 部署
```bash
docker-compose up -d
```
访问: http://localhost:8000

## 📁 项目结构

```
tiup-visualizer/
├── start.sh              # 一键启动 (开发)
├── start-prod.sh         # 一键启动 (生产)
├── backend/              # FastAPI 后端
│   └── app/
│       ├── api/          # REST API
│       ├── models/       # 数据模型
│       └── services/     # TiUP 服务
├── frontend/             # Vue 3 前端
│   └── src/
│       ├── components/   # UI 组件
│       ├── views/        # 页面
│       ├── stores/       # 状态管理
│       └── services/     # API 客户端
└── scripts/              # 工具脚本
```

## 🔧 技术栈

| 层级 | 技术 | 版本 |
|------|------|------|
| 后端框架 | FastAPI | 0.109.0 |
| ASGI 服务器 | Uvicorn | 0.27.0 |
| 数据验证 | Pydantic | 2.5.3 |
| 前端框架 | Vue | 3.4.15 |
| 构建工具 | Vite | 5.0.11 |
| 状态管理 | Pinia | 2.1.7 |
| HTTP 客户端 | Axios | 1.6.5 |
| 容器化 | Docker | - |

## 🎯 数据流

```
TiUP CLI
    ↓
tiup cluster list / display
    ↓
Backend (tiup_service.py)
    ↓ 解析输出
REST API (/api/v1/*)
    ↓ JSON
Frontend (api.js)
    ↓
Pinia Store (cluster.js)
    ↓
Vue Components
    ↓
UI Display
```

## 📊 API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/clusters` | GET | 获取所有集群 |
| `/api/v1/clusters/{name}` | GET | 获取集群详情 |
| `/api/v1/hosts` | GET | 获取所有主机 |
| `/api/v1/hosts/{ip}/clusters` | GET | 获取主机上的集群 |
| `/health` | GET | 健康检查 |
| `/docs` | GET | API 文档 (Swagger) |

## 🎨 UI 组件

| 组件 | 文件 | 功能 |
|------|------|------|
| HostCard | `HostCard.vue` | 物理主机卡片 |
| ClusterCard | `ClusterCard.vue` | 集群卡片 |
| ClusterDetailModal | `ClusterDetailModal.vue` | 详情弹窗 |
| ConnectionLines | `ConnectionLines.vue` | SVG 连接线 |
| HomeView | `HomeView.vue` | 主页面布局 |

## 🔒 安全注意事项

1. **命令执行**: 后端执行 TiUP 命令,需要确保:
   - 运行用户有执行 tiup 的权限
   - 设置合适的超时时间 (30秒)
   - 不接受用户输入作为命令参数

2. **CORS**: 默认允许 localhost:5173 和 localhost:3000
   - 生产环境需修改 `backend/app/core/config.py`

3. **权限**: 读取 TiUP 配置文件需要适当权限
   - 通常需要与部署 TiUP 集群的用户相同

## 📈 性能特点

- **异步处理**: FastAPI 支持异步请求
- **虚拟 DOM**: Vue 3 高效渲染
- **按需加载**: Vite 模块化构建
- **缓存优化**: 浏览器缓存静态资源

## 🐛 调试

### 查看日志
```bash
# 后端日志
tail -f /tmp/tiup-visualizer-backend.log

# 前端日志
tail -f /tmp/tiup-visualizer-frontend.log
```

### 测试后端 API
```bash
# 健康检查
curl http://localhost:8000/health

# 获取集群列表
curl http://localhost:8000/api/v1/clusters

# 查看 API 文档
open http://localhost:8000/docs
```

## 🔮 未来改进方向

### 功能增强
- [ ] 实时状态更新 (WebSocket)
- [ ] 集群操作功能 (启动/停止)
- [ ] 组件日志查看
- [ ] 性能指标图表
- [ ] 告警通知
- [ ] 多用户支持和权限管理

### UI/UX 改进
- [ ] 深色模式
- [ ] 自定义布局 (拖拽)
- [ ] 键盘快捷键
- [ ] 组件搜索和过滤
- [ ] 拓扑图视图

### 技术优化
- [ ] 数据缓存机制
- [ ] 增量更新
- [ ] 单元测试覆盖
- [ ] E2E 测试
- [ ] 性能监控
- [ ] CI/CD 流水线

## 📝 许可证

MIT License

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议!

## 📧 联系方式

项目地址: `/data/home/zacharyzliu/tikv-related-projects/tikv_operator/tiup-visualizer`

---

**项目状态**: ✅ 已完成核心功能,可以投入使用

**最后更新**: 2026-02-10
