# Stage、Job、Step 参考

## Stage (阶段)

### 基本语法
```yaml
stages:
  - name: build
    displayName: 构建阶段
  - name: test
    displayName: 测试阶段
  - name: deploy
    displayName: 部署阶段
```

### Finally Stage (收尾阶段)
总是在最后执行,无论成功或失败:

```yaml
finally:
  - stage: cleanup
    job: cleanup-job
    steps:
      - run: docker-compose down
  
  - stage: notify
    job: send-notification
    if: always()
    steps:
      - run: ./send-notification.sh
```

## Job (作业)

### 核心属性

| 属性 | 必填 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| stage | ✓ | string | - | 所属Stage |
| job | ✓ | string | - | Job唯一标识 |
| displayName | - | string | - | 显示名称 |
| steps | ✓ | array | - | 步骤列表 |
| pool | - | object | - | 构建环境 |
| env | - | object | - | 环境变量 |
| dependsOn | - | string/array | - | 依赖的Job |
| if | - | string | success() | 条件表达式 |
| timeout | - | number | 60 | 超时(分钟) |
| workspace | - | object | - | 工作空间配置 |
| concurrency | - | object | - | 并发控制 |
| mutex | - | object | - | 互斥组 |
| strategy | - | object | - | 矩阵策略 |
| outputs | - | object | - | 输出变量 |

### pool (构建环境)
```yaml
jobs:
  - stage: build
    job: compile
    pool:
      name: docker
      container: node:16
      resources:
        requests:
          cpu: 2
          memory: 4Gi
        limits:
          cpu: 4
          memory: 8Gi
```

### dependsOn (依赖)
```yaml
jobs:
  - stage: build
    job: compile
    steps:
      - run: make build
  
  # 单个依赖
  - stage: build
    job: package
    dependsOn: compile
    steps:
      - run: make package
  
  # 多个依赖
  - stage: test
    job: integration
    dependsOn:
      - compile-frontend
      - compile-backend
    steps:
      - run: npm test
```

### workspace (工作空间复用)
```yaml
jobs:
  - stage: build
    job: compile
    workspace:
      clean: false
    steps:
      - checkout: self
      - run: npm run build
  
  - stage: build
    job: package
    dependsOn: compile
    workspace:
      reuse: compile  # 复用compile的工作空间
    steps:
      - run: tar -czf app.tar.gz dist/
```

### strategy (矩阵策略)

**单维度**:
```yaml
jobs:
  - stage: test
    job: test-node
    strategy:
      matrix:
        node_version: ['14', '16', '18']
    pool:
      container: node:${{ matrix.node_version }}
    steps:
      - run: npm test
```

**多维度**:
```yaml
jobs:
  - stage: test
    job: cross-test
    strategy:
      matrix:
        os: ['ubuntu', 'macos']
        node_version: ['16', '18']
      max-parallel: 4
      fail-fast: false
      exclude:
        - os: macos
          node_version: '14'
    steps:
      - run: npm test
```

### outputs (输出变量)
```yaml
jobs:
  - stage: build
    job: compile
    outputs:
      version: ${{ steps.get-version.outputs.version }}
    steps:
      - id: get-version
        run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  - stage: deploy
    job: deploy-prod
    dependsOn: compile
    steps:
      - run: echo "版本: ${{ jobs.compile.outputs.version }}"
```

## Step (步骤)

### 通用属性

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| name | string | - | 显示名称 |
| id | string | - | 唯一标识 |
| if | string | - | 条件表达式 |
| env | object | - | 环境变量 |
| workingDirectory | string | - | 工作目录 |
| continueOnError | boolean | false | 失败是否继续 |
| timeout | number | - | 超时(分钟) |
| shell | string | bash | Shell类型 |

### 1. run (执行脚本)

**单行命令**:
```yaml
steps:
  - run: echo "Hello World"
  - run: npm install
```

**多行脚本**:
```yaml
steps:
  - run: |
      npm ci
      npm run build
      npm test
```

**设置输出变量**:
```yaml
steps:
  - id: get-version
    run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  - run: echo "${{ steps.get-version.outputs.version }}"
```

### 2. checkout (拉取代码)

**基本用法**:
```yaml
steps:
  - checkout: self
```

**完整属性**:
```yaml
steps:
  - checkout: self
    clean: true          # 清理工作目录
    fetchDepth: 1        # Git拉取深度
    lfs: false           # 拉取LFS文件
    submodules: false    # 拉取子模块
    path: ./source       # 检出路径
    ref: develop         # 分支/标签/提交
```

**常用场景**:
```yaml
# 拉取完整历史
steps:
  - checkout: self
    fetchDepth: 0

# 拉取子模块
steps:
  - checkout: self
    submodules: recursive

# 检出特定分支
steps:
  - checkout: self
    ref: release-v2.0
```

### 3. uses (使用插件)

**基本语法**:
```yaml
steps:
  - uses: <插件路径>@<版本>
    with:
      <参数名>: <参数值>
```

**常用插件**:
```yaml
# Docker构建
steps:
  - uses: docker/build@v1
    with:
      image: myapp
      tag: ${{ ci.build_number }}

# 上传构建产物
steps:
  - uses: upload-artifact@v1
    with:
      name: build-artifacts
      path: dist/

# 下载构建产物
steps:
  - uses: download-artifact@v1
    with:
      name: build-artifacts
      path: ./artifacts
```

**引用插件输出**:
```yaml
steps:
  - id: build-image
    uses: docker/build@v1
    with:
      image: myapp
  
  - run: echo "镜像: ${{ steps.build-image.outputs.image }}"
```

## 执行规则

### Stage执行顺序
```
Stage1 → Stage2 → Stage3 → ... → finally
```

### Job执行规则
- 同一Stage内的Job默认**并行执行**
- 使用`dependsOn`指定**串行依赖**
- 使用`mutex`实现**互斥执行**

### Step执行规则
- 同一Job内的Step**顺序执行**
- Step失败会导致后续Step跳过(除非`continueOnError: true`)
