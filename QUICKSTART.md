# TiUP Visualizer - Quick Start Guide

## 一键启动 (开发模式)

最简单的启动方式:

```bash
cd tiup-visualizer
make dev
```

**这会同时启动 Go 后端和 Vite 前端开发服务器，** 支持前端热更新。

**访问地址:**
- 🌐 前端页面: http://localhost:5173
- 🔧 后端 API: http://localhost:8000

**停止服务:** 按 `Ctrl+C`（同时停止前后端）

也可以分别启动：
```bash
make dev-backend    # 仅启动后端 (:8000)
make dev-frontend   # 仅启动前端 (:5173)，需后端已运行
```

**默认账号:** 用户名 `admin`，密码 `easygraph`

> 修改密码：编辑 `backend-go/config.yaml` 中的 `auth.username` 和 `auth.password` 字段，重启服务即可。
> Nginx 部署模式下修改部署目录的配置：`sudo vim /var/www/tiup-visualizer/config.yaml && sudo systemctl restart tiup-visualizer`

---

## 系统要求

- **Go 1.22+** - 运行后端
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
**A:** 修改 `backend-go/config.yaml` 中的端口配置，或设置环境变量：
```bash
LISTEN_ADDR=:8080 make dev
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

---

## 日志查看

开发模式下，日志直接输出到终端。

生产部署模式下：
```bash
# 查看 systemd 服务日志
sudo journalctl -u tiup-visualizer -f

# 查看 Nginx 日志
tail -f /var/log/nginx/tiup-visualizer-access.log
```

---

## 更多帮助

详细文档请查看: [README.md](README.md)
