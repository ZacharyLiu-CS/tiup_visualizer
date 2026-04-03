# 变量系统参考

## 1. 流水线级别变量 (variables)

**语法**:
```yaml
variables:
  APP_NAME: myapp
  VERSION: 1.0.0
```

**特点**:
- 全局作用域,所有Stage/Job/Step都可访问
- 不可修改
- 引用语法: `${{ variables.变量名 }}`

**示例**:
```yaml
variables:
  APP_NAME: myapp
  IMAGE_TAG: ${{ variables.APP_NAME }}:${{ ci.build_number }}

jobs:
  - stage: build
    steps:
      - run: docker build -t ${{ variables.IMAGE_TAG }} .
```

## 2. Job/Step 环境变量 (env)

**Job级别**:
```yaml
jobs:
  - stage: build
    job: build-app
    env:
      NODE_ENV: production
      API_URL: https://api.example.com
    steps:
      - run: npm run build
```

**Step级别**:
```yaml
steps:
  - name: 构建
    env:
      BUILD_TYPE: release
    run: npm run build:$BUILD_TYPE
```

**优先级**: Step级别 > Job级别 > 流水线级别

## 3. 凭据使用 (secrets)

**语法**: `${{ secrets.凭据名 }}`

**示例**:
```yaml
steps:
  - run: |
      echo "${{ secrets.DOCKER_PASSWORD }}" | \
      docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
```

## 4. 插件输出变量

**Step内引用**:
```yaml
steps:
  - id: docker-build
    uses: docker/build@v1
    with:
      image: myapp
  
  - run: echo "${{ steps.docker-build.outputs.image }}"
```

**跨Job传递**:
```yaml
jobs:
  - stage: build
    job: build-image
    outputs:
      image: ${{ steps.docker-build.outputs.image }}
    steps:
      - id: docker-build
        uses: docker/build@v1
  
  - stage: deploy
    job: deploy-app
    dependsOn: build-image
    steps:
      - run: echo "${{ jobs.build-image.outputs.image }}"
```

**自定义输出**:
```yaml
steps:
  - id: get-version
    run: echo "version=1.0.0" >> $GITHUB_OUTPUT
  
  - run: echo "${{ steps.get-version.outputs.version }}"
```

## 5. 引用语法对比

### ${{ }} 语法
用于流水线上下文(YAML解析时替换):
```yaml
${{ variables.变量名 }}
${{ ci.变量名 }}
${{ secrets.凭据名 }}
${{ steps.step-id.outputs.变量名 }}
```

### $VAR 语法
用于Shell环境变量(Shell执行时替换):
```yaml
steps:
  - env:
      NODE_ENV: production
    run: echo "$NODE_ENV"
```

## 6. 内置上下文

### ci上下文
```yaml
${{ ci.branch }}          # 当前分支名
${{ ci.build_number }}    # 构建号
${{ ci.commit_sha }}      # 提交哈希
${{ ci.commit_message }}  # 提交信息
${{ ci.author }}          # 提交作者
${{ ci.pipeline_id }}     # 流水线ID
```

### github上下文
```yaml
${{ github.ref }}         # Git引用
${{ github.repository }}  # 仓库名称
```

### matrix上下文
```yaml
${{ matrix.node_version }}  # 矩阵变量
${{ matrix.os }}
```

## 实用示例

```yaml
variables:
  APP_NAME: myapp
  VERSION: 2.0.0

jobs:
  - stage: build
    job: build-app
    env:
      NODE_ENV: production
    outputs:
      image_tag: ${{ steps.build.outputs.tag }}
    steps:
      - checkout: self
      
      - run: echo "Building ${{ variables.APP_NAME }}"
      
      - name: Docker登录
        run: |
          echo "${{ secrets.DOCKER_PASSWORD }}" | \
          docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
      
      - id: build
        uses: docker/build@v1
        with:
          image: ${{ variables.APP_NAME }}
          tag: ${{ ci.build_number }}
  
  - stage: deploy
    job: deploy-app
    dependsOn: build-app
    steps:
      - run: kubectl set image deployment/app app=${{ jobs.build-app.outputs.image_tag }}
```
