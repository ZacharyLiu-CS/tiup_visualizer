# TiUP Visualizer - Quick Start Guide

## 一键启动 (开发模式)

最简单的启动方式:

```bash
cd tiup-visualizer
./scripts/start-dev.sh
```

**这个脚本会自动完成:**
- ✅ 检查环境 (Python 3, Node.js, TiUP)
- ✅ 创建 Python 虚拟环境
- ✅ 安装所有依赖包
- ✅ 启动后端服务 (端口 8000)
- ✅ 启动前端服务 (端口 5173)

**访问地址:**
- 🌐 前端页面: http://localhost:5173
- 🔧 后端 API: http://localhost:8000
- 📚 API 文档: http://localhost:8000/docs

**停止服务:** 按 `Ctrl+C`

---

## 系统要求

- **Python 3.8+** - 运行后端
- **Node.js 18+** - 构建前端  
- **TiUP** - 必须已安装并在 PATH 中

### 安装 TiUP

如果还没有安装 TiUP:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://tiup-mirrors.pingcap.com/install.sh | sh
```

---

## 使用说明

### 1. 查看物理主机

页面顶部显示所有物理主机,每个卡片显示:
- 🖥️ 服务器图标
- 📍 IP 地址
- 📊 集群数量
- 🔧 组件数量

### 2. 查看 TiKV 集群

页面底部显示所有 TiKV 集群,每个卡片显示:
- 🔷 集群名称
- 📦 版本号
- 👤 部署用户

### 3. 交互操作

**点击主机:**
- 高亮显示该主机上的所有集群
- 绘制连接线

**点击集群:**
- 高亮显示集群部署的所有主机
- 打开详情弹窗,显示:
  - 集群基本信息
  - Dashboard 和 Grafana 链接
  - 所有组件详细信息 (IP、端口、状态、目录)

**清除选择:**
- 点击 "Clear Selection" 按钮
- 或再次点击已选中的项

---

## Docker 部署

如果更喜欢使用 Docker:

```bash
cd tiup-visualizer
docker-compose up -d
```

访问: http://localhost:8000

---

## 常见问题

### Q: 提示 "TiUP is not installed"?
**A:** 需要先安装 TiUP,或确保 tiup 命令在 PATH 中

### Q: 端口被占用?
**A:** 修改 `backend/app/core/config.py` 中的端口配置,或在启动命令中指定:
```bash
# 后端使用其他端口
cd backend
source venv/bin/activate
python -m uvicorn app.main:app --port 8080
```

### Q: 看不到集群数据?
**A:** 确保:
1. TiUP 已正确安装
2. 当前用户有权限执行 `tiup cluster list` 命令
3. 已经部署了 TiKV 集群

### Q: 前端无法连接后端?
**A:** 检查:
1. 后端是否正常启动 (访问 http://localhost:8000/health)
2. 防火墙设置
3. CORS 配置 (在 `backend/app/core/config.py`)

---

## 日志查看

开发模式下,日志保存在:
- 后端: `/tmp/tiup-visualizer-backend.log`
- 前端: `/tmp/tiup-visualizer-frontend.log`

查看实时日志:
```bash
tail -f /tmp/tiup-visualizer-backend.log
tail -f /tmp/tiup-visualizer-frontend.log
```

---

## 更多帮助

详细文档请查看: [README.md](README.md)
