# 条件执行参考

## if 属性使用

### Job 级别
```yaml
jobs:
  - stage: deploy
    job: deploy-prod
    if: github.ref == 'refs/heads/master'
    steps:
      - run: kubectl apply -f deployment.yaml
```

### Step 级别
```yaml
steps:
  - name: 部署到生产
    if: success() && github.ref == 'refs/heads/master'
    run: kubectl apply -f production.yaml
```

## 条件函数

| 函数 | 说明 |
|------|------|
| `success()` | 前面的Step/Job都成功 |
| `failure()` | 任何前面的Step/Job失败 |
| `always()` | 总是执行 |
| `cancelled()` | 流水线被取消 |

## 表达式运算符

### 比较运算符
```yaml
if: ci.branch == 'master'
if: ci.branch != 'develop'
if: ci.build_number > 100
if: ci.build_number <= 1000
```

### 逻辑运算符
```yaml
# AND
if: success() && github.ref == 'refs/heads/master'

# OR
if: ci.branch == 'master' || ci.branch == 'develop'

# NOT (需要引号)
if: "!startsWith(github.ref, 'refs/tags/')"
```

### 括号分组
```yaml
if: (ci.branch == 'master' || ci.branch == 'develop') && success()
```

## 字符串函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `startsWith(str, prefix)` | 字符串以prefix开头 | `startsWith(github.ref, 'refs/heads/')` |
| `endsWith(str, suffix)` | 字符串以suffix结尾 | `endsWith(ci.branch, '-dev')` |
| `contains(str, substring)` | 字符串包含substring | `contains(ci.commit_message, '[deploy]')` |

## 实用示例

### 根据分支部署
```yaml
jobs:
  - stage: deploy
    job: deploy-prod
    if: github.ref == 'refs/heads/master'
    steps:
      - run: ./deploy-production.sh
  
  - stage: deploy
    job: deploy-test
    if: github.ref == 'refs/heads/develop'
    steps:
      - run: ./deploy-test.sh
```

### 失败时清理
```yaml
steps:
  - run: npm test
  
  - name: 失败清理
    if: failure()
    run: docker-compose down
```

### 总是发送通知
```yaml
steps:
  - run: npm run build
  
  - name: 发送通知
    if: always()
    run: ./send-notification.sh
```

### 提交信息条件
```yaml
jobs:
  - stage: deploy
    job: manual-deploy
    if: contains(ci.commit_message, '[deploy]')
    steps:
      - run: ./deploy.sh
  
  - stage: test
    job: skip-tests
    if: "!contains(ci.commit_message, '[skip tests]')"
    steps:
      - run: npm test
```

### 组合条件
```yaml
# 生产分支且构建成功
if: success() && github.ref == 'refs/heads/master'

# master或develop分支
if: github.ref == 'refs/heads/master' || github.ref == 'refs/heads/develop'

# 非标签触发
if: "!startsWith(github.ref, 'refs/tags/')"

# 复杂组合
if: success() && github.ref == 'refs/heads/master' && !contains(ci.commit_message, '[skip deploy]')
```

## 注意事项

1. **默认行为**: 不设置if时,默认为`if: success()`
2. **引号使用**: 包含`!`的表达式必须用引号包裹
3. **运算符优先级**: 使用括号明确优先级
4. **大小写敏感**: 变量名和字符串比较区分大小写
