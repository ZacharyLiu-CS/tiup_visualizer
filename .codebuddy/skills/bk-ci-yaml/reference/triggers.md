# 触发器配置参考

## 总体说明

- **顶级关键字**: `on`
- **默认配置**: 监听所有push、tag、mr事件和openapi
- **重要**: 触发器属性值不支持通过上下文变量设置

## Push 触发器

```yaml
on:
  push:
    branches: ["master", "release/*"]
    branches-ignore: []
    path-filter-type: NamePrefixFilter
    paths: []
    paths-ignore: []
    users: []
    users-ignore: []
    action: [new-branch, push-file]
```

**通配符规则**:
- `*`: 单层不含斜杠
- `**`: 所有分支
- `release/*`: 双层路径
- `release/**`: 多层路径

## Tag 触发器

```yaml
on:
  tag:
    tags: ["v2.*"]
    tags-ignore: ["v1.*"]
    from-branches: ["master"]
    users: []
    users-ignore: []
```

## MR 触发器

```yaml
on:
  mr:
    target-branches: ["master"]
    source-branches: []
    action: [open, reopen, push-update]
    report-commit-check: true
    block-mr: false
```

**action取值**:
- `open`: 新建MR
- `close`: 关闭MR
- `reopen`: 重新打开
- `push-update`: 源分支push
- `merge`: 已合并

## Schedules 定时触发器

```yaml
on:
  schedules:
    cron: "0 1 * * *"
    branches: [master]  # 最多3个,不支持通配符
    always: false
```

## 跨代码库触发器

```yaml
on:
  repo-name: mingshewhe/webhook_test3
  type: git  # git/tgit/github/p4/svn
  push:
    branches: ["master"]
```

## Issue/Review/Note 触发器

```yaml
on:
  issue:
    action: [open, close, reopen, update]
  
  review:
    states: [approved, change_required]
  
  note:
    types: [commit, merge_request, issue]
    comment: ["hejie*"]
```

## 实用示例

### 生产环境发布
```yaml
on:
  - push: [master]
  - tag:
      tags: ["v*"]
      from-branches: [master]
```

### 特性开发
```yaml
on:
  mr:
    source-branches: [feat_**]
    target-branches: [develop]
    block-mr: true
```

### 夜间构建
```yaml
on:
  schedules:
    cron: "0 2 * * *"
    branches: [master]
    always: false
```
