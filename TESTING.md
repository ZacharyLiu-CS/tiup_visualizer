# Testing Guide

## 快速测试 (无需真实 TiUP)

如果你想在没有真实 TiUP 环境的情况下测试应用,可以使用 mock 脚本。

### 方法 1: 临时添加 mock tiup 到 PATH

```bash
# 在 tiup-visualizer 目录下
export PATH="$(pwd)/scripts:$PATH"
ln -sf mock-tiup.sh scripts/tiup

# 启动应用
./start.sh
```

### 方法 2: 修改后端代码使用 mock

编辑 `backend/app/services/tiup_service.py`:

```python
def execute_command(command: str) -> str:
    """Execute tiup command and return output"""
    try:
        # 添加这行使用 mock
        if "tiup" in command:
            command = command.replace("tiup", "./scripts/mock-tiup.sh")
        
        result = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            timeout=30
        )
        return result.stdout
    except subprocess.TimeoutExpired:
        raise Exception("Command execution timeout")
    except Exception as e:
        raise Exception(f"Command execution failed: {str(e)}")
```

## 单元测试

运行后端单元测试:

```bash
cd backend
source venv/bin/activate
pytest tests/ -v
```

预期输出:
```
tests/test_tiup_service.py::test_parse_cluster_list PASSED
tests/test_tiup_service.py::test_parse_cluster_display PASSED

====== 2 passed in 0.05s ======
```

## API 测试

### 使用 curl

```bash
# 健康检查
curl http://localhost:8000/health

# 获取所有集群
curl http://localhost:8000/api/v1/clusters | jq

# 获取集群详情
curl http://localhost:8000/api/v1/clusters/eg3-cicd-proxy | jq

# 获取所有主机
curl http://localhost:8000/api/v1/hosts | jq
```

### 使用浏览器

访问 API 文档: http://localhost:8000/docs

这是 FastAPI 自动生成的交互式 API 文档,可以直接测试所有端点。

## 前端测试

### 手动测试清单

- [ ] 页面加载成功
- [ ] 显示所有物理主机卡片
- [ ] 显示所有集群卡片
- [ ] 点击主机,高亮相关集群
- [ ] 绘制连接线(从主机到集群)
- [ ] 点击集群,高亮相关主机
- [ ] 绘制连接线(从集群到主机)
- [ ] 打开集群详情弹窗
- [ ] 详情弹窗显示正确信息
- [ ] 组件状态颜色正确(Up=绿, Down=红)
- [ ] 组件角色徽章颜色正确
- [ ] Dashboard 链接可点击
- [ ] Grafana 链接可点击
- [ ] 关闭详情弹窗
- [ ] Clear Selection 按钮工作
- [ ] 响应式布局(调整窗口大小)

### 浏览器控制台测试

打开浏览器开发者工具 (F12),在控制台执行:

```javascript
// 测试 API 调用
fetch('/api/v1/clusters')
  .then(r => r.json())
  .then(console.log)

// 测试状态管理
import { useClusterStore } from './stores/cluster.js'
const store = useClusterStore()
console.log(store.clusters)
console.log(store.hosts)
```

## 压力测试

测试 API 性能:

```bash
# 安装 Apache Bench
sudo apt-get install apache2-utils  # Ubuntu/Debian
# 或
brew install httpd  # macOS

# 运行测试 (1000 requests, 10 concurrent)
ab -n 1000 -c 10 http://localhost:8000/api/v1/clusters
```

## 回归测试

在修改代码后,确保以下场景仍然工作:

### 场景 1: 基本功能
1. 启动应用
2. 访问页面
3. 确认所有卡片显示
4. 点击一个主机
5. 确认连接线和高亮

### 场景 2: 详情查看
1. 点击一个集群
2. 确认详情弹窗打开
3. 检查所有字段显示正确
4. 关闭弹窗
5. 确认选择已清除

### 场景 3: 多次交互
1. 点击主机 A
2. 点击主机 B (切换)
3. 点击集群 X
4. 点击 Clear Selection
5. 点击同一集群 X (再次打开)

## Mock 数据说明

`scripts/mock-tiup.sh` 提供的测试数据:

- **3 个集群**:
  - eg3-cicd-proxy
  - eg3_cicd_graphrag
  - eg3_cicd_ldbc_rw

- **3 个主机**:
  - 11.154.160.246
  - 11.154.160.28
  - 11.154.160.37

- **18 个组件**:
  - 12x TiKV (每主机 4 个)
  - 3x PD (每主机 1 个)
  - 2x Prometheus
  - 1x Grafana

状态分布:
- 大部分: Up (正常)
- 少数: Down (停止)
- 1 个: N/A (未知)

## 调试技巧

### 后端调试

1. 启用详细日志:
```python
# backend/app/main.py
import logging
logging.basicConfig(level=logging.DEBUG)
```

2. 查看实时请求:
```bash
tail -f /tmp/tiup-visualizer-backend.log
```

### 前端调试

1. 启用 Vue DevTools (浏览器扩展)

2. 查看网络请求:
   - 打开 DevTools → Network 标签
   - 刷新页面
   - 检查 API 请求和响应

3. 查看状态管理:
   - Vue DevTools → Pinia 标签
   - 查看 cluster store 状态

### 常见问题排查

| 问题 | 检查 | 解决方案 |
|------|------|----------|
| 无法连接后端 | `curl http://localhost:8000/health` | 确保后端已启动 |
| 空白页面 | 浏览器控制台错误信息 | 检查前端构建是否成功 |
| 无数据显示 | `/api/v1/clusters` 返回 | 检查 tiup 命令执行权限 |
| 连接线不显示 | 选择状态 | 确保已选中主机或集群 |
| 样式错乱 | 静态资源加载 | 清除浏览器缓存 |

## 自动化测试 (未来)

计划添加:
- [ ] E2E 测试 (Playwright/Cypress)
- [ ] 前端单元测试 (Vitest)
- [ ] API 集成测试
- [ ] 视觉回归测试
- [ ] 性能基准测试

## 测试报告

记录测试结果:

```markdown
## 测试日期: YYYY-MM-DD
## 测试人员: XXX

### 环境
- Python: 3.11.x
- Node.js: 20.x
- Browser: Chrome 120.x

### 测试结果
- ✅ 后端 API 测试: 通过
- ✅ 前端功能测试: 通过
- ⚠️ 性能测试: 部分通过 (详见说明)
- ❌ 兼容性测试: 失败 (Safari 问题)

### 问题列表
1. Safari 浏览器连接线不显示 - 需要修复
2. 移动端响应式布局需要优化

### 建议
1. 添加更多单元测试覆盖
2. 优化大量集群时的渲染性能
```
