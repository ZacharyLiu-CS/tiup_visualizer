# 常见陷阱与注意事项

# 常见陷阱与注意事项

## 🚨 结构与格式相关

### ❌ 陷阱 1: variables 值必须是 object
```yaml
# ❌ 错误 - 直接写字符串
variables:
  APP_NAME: "myapp"
  VERSION: "1.0.0"

# ✅ 正确 - 使用 object 格式
variables:
  APP_NAME:
    value: "myapp"
  VERSION:
    value: "1.0.0"
    const: true
  ENV:
    value: "test"
    allow-modify-at-startup: true
```

### ❌ 陷阱 2: stages 内嵌 jobs (不是顶层并列)
```yaml
# ❌ 错误 - 旧版格式(顶层 jobs 数组)
stages:
  - name: build
jobs:
  - stage: build
    job: compile
    steps: [...]

# ✅ 正确 - stages 内嵌 jobs (object Map)
stages:
  - name: build
    label: Build
    jobs:
      compile:         # job ID 作为 key
        name: 编译
        steps: [...]
```

### ❌ 陷阱 3: stages[].label 只接受特定值
```yaml
# ❌ 错误 - 自定义 label
stages:
  - name: build
    label: "构建阶段"  # ❌ 不被接受

# ✅ 正确 - 使用预定义值或不设置
stages:
  - name: build
    label: Build      # ✅ Build/Test/Deploy/Approve
  - name: custom
    # 不设置 label 也可以
```

## 🚨 变量引用相关

### ❌ 陷阱 4: 条件表达式中引用变量不加 ${{ }}
```yaml
# ❌ 错误 - 在 if 中使用 ${{ }}
jobs:
  deploy:
    if: ${{ variables.ENV }} == 'prod'  # ❌ 解析失败
    steps: [...]

# ✅ 正确 - if 中直接用变量名
jobs:
  deploy:
    if: variables.ENV == 'prod'  # ✅ 正确语法
    steps: [...]
```

### ❌ 陷阱 5: 流水线变量必须带 variables. 前缀
```yaml
# ❌ 错误 - 缺少 variables 前缀
variables:
  APP_NAME:
    value: "myapp"

steps:
  - run: echo "${{ APP_NAME }}"  # ❌ 无法解析

# ✅ 正确
steps:
  - run: echo "${{ variables.APP_NAME }}"  # ✅ 必须带 variables.
```

### ❌ 陷阱 6: 矩阵变量引用错误
```yaml
# ❌ 错误
strategy:
  matrix:
    node: ['14', '16']
steps:
  - run: echo "$node"  # 无法获取

# ✅ 正确
strategy:
  matrix:
    node: ['14', '16']
steps:
  - run: echo "${{ matrix.node }}"
```

## 🚨 触发器相关

### ❌ 陷阱 7: 触发器属性不支持上下文变量
```yaml
# ❌ 错误 - 触发器配置不能使用变量
on:
  push:
    branches: ["${{ variables.BRANCH }}"]

# ✅ 正确 - 必须写死或使用通配符
on:
  push:
    branches: ["master", "release/**"]
```

### ❌ 陷阱 8: 定时任务不支持秒级别
```yaml
# ❌ 错误 - 包含秒字段
on:
  schedules:
    cron: "0 0 1 * * *"

# ✅ 正确 - 5个字段
on:
  schedules:
    cron: "0 1 * * *"
```

### ❌ 陷阱 9: schedules branches不支持通配符
```yaml
# ❌ 错误
on:
  schedules:
    cron: "0 1 * * *"
    branches: ["release/*"]

# ✅ 正确 - 最多3个,明确指定
on:
  schedules:
    cron: "0 1 * * *"
    branches: [master, develop]
```

## 🚨 Job/Step相关

### ❌ 陷阱 10: finally的Job不能依赖普通Stage的Job
```yaml
# ❌ 错误 - finally 不能依赖普通 Job
stages:
  - name: build
    jobs:
      compile:
        steps:
          - run: make build

finally:
  - name: cleanup
    jobs:
      cleanup-job:
        dependsOn: compile  # ❌ finally不能依赖普通Job
        steps:
          - run: make clean

# ✅ 正确 - finally的Job之间可以依赖
finally:
  - name: cleanup
    jobs:
      cleanup-docker:
        steps:
          - run: docker-compose down
      
      cleanup-files:
        dependsOn: cleanup-docker  # ✅ OK
        steps:
          - run: rm -rf /tmp/*
```

### ❌ 陷阱 11: Step输出变量跨Job传递需要声明outputs
```yaml
# ❌ 错误 - 直接跨Job引用
stages:
  - name: build
    jobs:
      build-app:
        steps:
          - id: get-version
            run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  - name: deploy
    jobs:
      deploy-app:
        steps:
          - run: echo "${{ steps.get-version.outputs.version }}"  # ❌ 无法访问

# ✅ 正确 - 通过Job outputs传递
stages:
  - name: build
    jobs:
      build-app:
        outputs:
          version: ${{ steps.get-version.outputs.version }}
        steps:
          - id: get-version
            run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  - name: deploy
    jobs:
      deploy-app:
        steps:
          - run: echo "${{ jobs.build-app.outputs.version }}"  # ✅ OK
```

### ❌ 陷阱 12: workspace.reuse 必须配合 dependsOn
```yaml
# ❌ 错误
stages:
  - name: build
    jobs:
      compile:
        steps:
          - checkout: self
      
      package:
        workspace:
          reuse: compile  # ❌ 没有dependsOn会失败
        steps:
          - run: tar -czf app.tar.gz dist/

# ✅ 正确
stages:
  - name: build
    jobs:
      compile:
        steps:
          - checkout: self
      
      package:
        dependsOn: compile  # ✅ 必须依赖
        workspace:
          reuse: compile
        steps:
          - run: tar -czf app.tar.gz dist/
```

## 🚨 通知相关

### ❌ 陷阱 13: 通知配置是 notices 不是 notifications
```yaml
# ❌ 错误
notifications:
  - type: email

# ✅ 正确
notices:
  - type: email
```

### ❌ 陷阱 14: notices的if不支持自定义表达式
```yaml
# ❌ 错误
notices:
  - type: email
    if: github.ref == 'refs/heads/master'  # ❌ 不支持

# ✅ 正确 - 只支持固定值
notices:
  - type: email
    if: FAILURE  # FAILURE/SUCCESS/CANCELED/ALWAYS
```

## 🚨 并发控制相关

### ❌ 陷阱 15: queue-length和queue-timeout只在cancel-in-progress=false时生效
```yaml
# ❌ 错误 - queue相关配置不生效
concurrency:
  group: production
  cancel-in-progress: true  # ❌ 直接取消,队列配置无效
  queue-length: 3
  queue-timeout-minutes: 30

# ✅ 正确
concurrency:
  group: production
  cancel-in-progress: false  # ✅ 必须为false
  queue-length: 3
  queue-timeout-minutes: 30
```

## 🚨 语法相关

### ❌ 陷阱 16: 条件表达式包含!需要引号
```yaml
# ❌ 错误
if: !contains(ci.commit_message, '[skip]')

# ✅ 正确
if: "!contains(ci.commit_message, '[skip]')"
```

### ❌ 陷阱 17: checkout步骤fetchDepth=0才能获取完整历史
```yaml
# ❌ 错误 - 只拉取最新提交
steps:
  - checkout: self
  - run: git log --oneline -10  # 可能只有1条

# ✅ 正确 - 拉取完整历史
steps:
  - checkout: self
    fetchDepth: 0
  - run: git log --oneline -10
```

## 🚨 性能相关

### ❌ 陷阱 18: 不必要的clean会降低性能
```yaml
# ❌ 不推荐 - 每次都清理,浪费时间
stages:
  - name: build
    jobs:
      compile:
        workspace:
          clean: true
        steps:
          - checkout: self
          - run: npm ci  # 每次重新下载依赖

# ✅ 推荐 - 复用工作空间加速构建
stages:
  - name: build
    jobs:
      compile:
        workspace:
          clean: false
        steps:
          - checkout: self
            clean: false
          - run: npm ci
```

## 📋 检查清单

生成YAML后自检:
- [ ] **variables 值是 object**(包含 value 属性)
- [ ] **stages 内嵌 jobs**(jobs 是 stage 子属性,object 类型)
- [ ] **stages[].label** 使用有效值(Build/Test/Deploy/Approve)或不设置
- [ ] **条件中引用变量**: `variables.XXX == 'xxx'`(不带 `${{ }}`)
- [ ] **流水线变量**: `${{ variables.XXX }}`(必须带 variables. 前缀)
- [ ] Stage名称/Job ID唯一性
- [ ] dependsOn无循环依赖
- [ ] 触发器属性不含上下文变量
- [ ] 通知配置使用`notices`而非`notifications`
- [ ] workspace.reuse配合dependsOn使用
- [ ] finally的Job不依赖普通Stage的Job
