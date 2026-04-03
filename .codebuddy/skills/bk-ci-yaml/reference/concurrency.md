# 并发控制参考

## 流水线级别并发控制

### 基本语法
```yaml
concurrency:
  group: <并发组名>
  cancel-in-progress: <是否取消进行中>
  queue-length: <队列长度>
  queue-timeout-minutes: <队列超时>
```

### 参数说明

| 参数 | 必填 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| group | ✓ | string | - | 并发组名,支持变量 |
| cancel-in-progress | - | boolean | true | 是否取消进行中任务 |
| queue-length | - | number | - | 队列最大长度(1-20) |
| queue-timeout-minutes | - | number | - | 队列超时(1-1440分钟) |

### 实用示例

**分支级别并发**:
```yaml
concurrency:
  group: ${{ ci.branch }}
  cancel-in-progress: true

on:
  push:
    branches: ['**']
```

**PR自动取消旧构建**:
```yaml
concurrency:
  group: pr-${{ ci.pull_request_number }}
  cancel-in-progress: true

on:
  pull_request:
```

**生产流水线排队**:
```yaml
concurrency:
  group: production-release
  cancel-in-progress: false
  queue-length: 3
  queue-timeout-minutes: 60

on:
  push:
    branches: [master]
```

## Job级别并发控制

### 基本语法
```yaml
jobs:
  - stage: deploy
    job: deploy-prod
    concurrency:
      group: <并发组名>
      cancel-in-progress: <是否取消>
    steps:
      - run: kubectl apply -f deployment.yaml
```

### 实用示例

**生产部署串行执行**:
```yaml
jobs:
  - stage: deploy
    job: deploy-frontend
    concurrency:
      group: production-deployment
      cancel-in-progress: false
    steps:
      - run: kubectl apply -f frontend.yaml
  
  - stage: deploy
    job: deploy-backend
    concurrency:
      group: production-deployment
      cancel-in-progress: false
    steps:
      - run: kubectl apply -f backend.yaml
```

**环境隔离**:
```yaml
jobs:
  # 生产环境 - 串行,不取消
  - stage: deploy
    job: deploy-prod
    if: github.ref == 'refs/heads/master'
    concurrency:
      group: production
      cancel-in-progress: false
    steps:
      - run: ./deploy-prod.sh
  
  # 测试环境 - 可取消
  - stage: deploy
    job: deploy-test
    if: github.ref == 'refs/heads/develop'
    concurrency:
      group: test-env
      cancel-in-progress: true
    steps:
      - run: ./deploy-test.sh
```

## 互斥组 (mutex)

### 基本语法
```yaml
jobs:
  - stage: deploy
    job: deploy
    mutex:
      name: <互斥锁名称>
      timeout: <超时时间>
    steps:
      - run: ./deploy.sh
```

### concurrency vs mutex

**concurrency**:
- 可选择取消或排队
- 支持动态组名
- 支持队列管理
- 更灵活

**mutex**:
- 总是排队,不取消
- 支持超时设置
- 简单的互斥锁
- 更简洁

### 实用示例

```yaml
jobs:
  - stage: deploy
    job: deploy-frontend
    mutex:
      name: production-deploy
      timeout: 60
    steps:
      - run: kubectl apply -f frontend.yaml
  
  - stage: deploy
    job: deploy-backend
    mutex:
      name: production-deploy
      timeout: 60
    steps:
      - run: kubectl apply -f backend.yaml
```

## 最佳实践

### 组名设计
- 开发分支: `dev-${{ ci.branch }}`
- PR构建: `pr-${{ ci.pull_request_number }}`
- 生产环境: 固定组名如`production`

### 取消策略
- 开发/测试环境: `cancel-in-progress: true` (快速迭代)
- 生产环境: `cancel-in-progress: false` (确保完成)

### 层级关系
Job级别并发控制叠加在流水线级别之上,可以同时使用两个级别的控制。
