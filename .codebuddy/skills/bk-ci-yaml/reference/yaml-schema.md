# Pipeline YAML Schema 完整速查表

本文档汇总了流水线 YAML 的所有属性,按层级组织。每个属性包含类型、是否必填、默认值、描述和约束规则。

---

## 📋 顶层属性 (Top-Level)

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `name` | String | 否 | YAML文件路径 | Pipeline名称 | 不支持上下文变量 |
| `label` | String/Array | 否 | - | Pipeline标签 | 单个或多个标签 |
| `version` | String | 否 | v3.0 | YAML语法版本 | 如v3.0 |
| `on` | Object | 否 | push/tag/mr/openapi | 触发器配置 | 不支持上下文变量 |
| `variables` | Object | 否 | - | 流水线级别全局变量 | **值必须是object**(包含value属性) |
| `stages` | Array | **是** | - | 阶段定义 | 每个stage内嵌jobs(object) |
| `finally` | Array | 否 | - | 收尾阶段 | 总是执行 |
| `notices` | Array | 否 | - | 流水线通知配置 | 注意是notices不是notifications |
| `concurrency` | Object | 否 | - | 流水线并发控制 | 项目级生效 |
| `custom-build-num` | String | 否 | - | 自定义构建号格式 | 如`${{DATE:"yyyyMMdd"}}.${{BUILD_NO_OF_DAY}}` |
| `cancel-policy` | Enum | 否 | BROAD | 流水线取消策略 | BROAD/RESTRICTED |
| `syntax-dialect` | Enum | 否 | INHERIT | 流水线语法风格 | CONSTRAINT/CLASSIC/INHERIT |

---

## 🔔 触发器属性 (on.*)

### on.push

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.push.branches` | Array | 否 | ["*"] | 监听的分支 | 支持`*`通配符,`**`匹配所有,`release/*`匹配双层,`release/**`匹配多层 |
| `on.push.branches-ignore` | Array | 否 | - | 排除的分支 | 规则同branches |
| `on.push.path-filter-type` | String | 否 | NamePrefixFilter | 路径过滤类型 | RegexBasedFilter/NamePrefixFilter |
| `on.push.paths` | Array | 否 | - | 监听的路径 | 支持通配符或前缀匹配 |
| `on.push.paths-ignore` | Array | 否 | - | 排除的路径 | 规则同paths |
| `on.push.users` | Array | 否 | - | 指定用户 | 用户名列表 |
| `on.push.users-ignore` | Array | 否 | - | 排除的用户 | 用户名列表 |
| `on.push.action` | Array | 否 | ["new-branch","push-file"] | push动作类型 | new-branch/push-file |

### on.tag

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.tag.tags` | Array | 否 | ["*"] | 监听的tag名称 | 支持`*`通配符 |
| `on.tag.tags-ignore` | Array | 否 | - | 排除的tag | 规则同tags |
| `on.tag.users` | Array | 否 | - | 指定用户 | 支持`*`通配符 |
| `on.tag.users-ignore` | Array | 否 | - | 排除的用户 | 用户名列表 |
| `on.tag.from-branches` | Array | 否 | - | 来源分支 | 支持`*`通配符 |

### on.mr

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.mr.target-branches` | Array | 否 | ["*"] | MR目标分支 | 支持`*`通配符 |
| `on.mr.source-branches` | Array | 否 | - | MR源分支 | 支持`*`通配符 |
| `on.mr.source-branches-ignore` | Array | 否 | - | 排除的源分支 | 规则同source-branches |
| `on.mr.target-branches-ignore` | Array | 否 | - | 排除的目标分支 | 规则同target-branches |
| `on.mr.path-filter-type` | String | 否 | NamePrefixFilter | 路径过滤类型 | RegexBasedFilter/NamePrefixFilter |
| `on.mr.paths` | Array | 否 | - | 监听的路径 | 支持通配符或前缀匹配 |
| `on.mr.paths-ignore` | Array | 否 | - | 排除的路径 | 规则同paths |
| `on.mr.action` | Array | 否 | ["open","reopen","push-update"] | MR动作类型 | open/close/reopen/push-update/merge |
| `on.mr.users` | Array | 否 | - | 指定用户 | 用户名列表 |
| `on.mr.users-ignore` | Array | 否 | - | 排除的用户 | 用户名列表 |
| `on.mr.report-commit-check` | Boolean | 否 | true | 上报commit check到工蜂 | true/false |
| `on.mr.block-mr` | Boolean | 否 | false | 失败时阻止MR合并 | 需report-commit-check=true |

### on.schedules

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.schedules.cron` | String | **是** | - | crontab表达式 | 不支持秒级别 |
| `on.schedules.branches` | Array | 否 | 默认分支 | 执行的分支 | 最多3个,不支持通配符 |
| `on.schedules.always` | Boolean | 否 | false | 是否始终执行 | false=仅代码变更时执行 |
| `on.schedules.repo-id` | String | 否 | - | 代码库HashId | 优先级高于repo-name |
| `on.schedules.repo-name` | String | 否 | - | 代码库别名 | - |

### on.issue

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.issue.action` | Array | 否 | - | Issue动作 | open/close/reopen/update |

### on.review

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.review.states` | Array | 否 | - | 评审状态 | approved/approving/change_denied/change_required |

### on.note

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.note.types` | Array | 否 | - | 评论类型 | commit/merge_request/issue |
| `on.note.comment` | Array | 否 | - | 评论内容过滤 | 支持正则表达式 |

### 跨仓库触发器

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `on.repo-name` | String | **是** | - | 代码库别名 | 需预先关联 |
| `on.type` | String | **是** | - | 代码库类型 | git/tgit/github/p4/svn |

---

## 🔄 并发控制属性 (concurrency.*)

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `concurrency.group` | String | **是** | - | 并发组名 | 支持变量:`ci.pipeline_id`/`ci.branch`/`ci.head_ref`/`ci.base_ref` |
| `concurrency.cancel-in-progress` | Boolean | 否 | true | 是否取消进行中任务 | true/false |
| `concurrency.queue-length` | Integer | 否 | - | 队列最大长度 | 1-20,cancel-in-progress=false时生效 |
| `concurrency.queue-timeout-minutes` | Integer | 否 | - | 队列超时时间(分钟) | 1-1440,cancel-in-progress=false时生效 |

---

## 📋 阶段与作业嵌套结构 (stages.* / jobs.*)

### ⚠️ 重要: stages 和 jobs 的正确嵌套关系

**stages 和 jobs 的关系是: stages 内嵌 jobs (不是顶层并列)**

正确格式:
```yaml
stages:
  - name: stage_name
    label: Build          # 可选: Build/Test/Deploy/Approve
    jobs:
      job_id:             # jobs 是 object (Map),不是数组
        name: 作业名称
        steps: [...]
```

**错误格式** (旧版本,已废弃):
```yaml
stages:
  - name: build
jobs:                     # ❌ 顶层 jobs 数组
  - stage: build
    job: xxx
```

### 阶段属性 (stages.*)

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `stages[].name` | String | **是** | - | 阶段唯一标识符 | 必须唯一 |
| `stages[].label` | Enum | 否 | - | 阶段标签 | Build/Test/Deploy/Approve |
| `stages[].jobs` | Object | **是** | - | 该阶段的作业Map | key为job ID,value为job定义 |

### 作业属性 (stages[].jobs.<job-id>.*)

**注意**: jobs 是 stage 的子属性,是 **object (Map)** 不是数组

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `jobs.<job-id>.name` | String | 否 | - | Job显示名称 | 在Web界面展示 |
| `jobs.<job-id>.if` | String | 否 | - | 条件表达式 | 控制Job是否执行 |
| `jobs.<job-id>.steps` | Array | **是** | - | 步骤列表 | 至少一个step |
| `jobs.<job-id>.runs-on` | Object | 否 | - | 构建机配置(非标准) | self-hosted等 |

### 环境配置

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `jobs.<job-id>.pool` | Object | 否 | - | 构建环境配置 | - |
| `jobs.<job-id>.runs-on` | Object | 否 | - | 私有构建机配置 | 非标准字段 |
| `jobs.<job-id>.runs-on.self-hosted` | Boolean | 否 | - | 是否私有构建机 | - |
| `jobs.<job-id>.runs-on.node-name` | String | 否 | - | 节点名称 | - |
| `jobs.<job-id>.runs-on.agent-selector` | Array | 否 | - | Agent选择器 | 如["macos"] |
| `jobs.<job-id>.env` | Object | 否 | - | Job级环境变量 | key-value格式 |

### 执行控制

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `jobs.<job-id>.dependsOn` | String/Array | 否 | - | Job依赖 | 依赖其他Job |
| `jobs.<job-id>.if` | String | 否 | - | 条件表达式 | 控制Job是否执行,引用变量用`variables.XXX` |
| `jobs.<job-id>.timeout` | Number | 否 | 60 | Job超时时间(分钟) | - |
| `jobs.<job-id>.continueOnError` | Boolean | 否 | false | 失败是否继续 | true/false |

---

## 🔧 步骤属性 (steps.*)

### 通用属性

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `steps[].name` | String | 否 | - | 步骤显示名称 | - |
| `steps[].id` | String | 否 | - | 步骤唯一标识符 | 用于引用输出 |
| `steps[].if` | String | 否 | success() | 条件表达式 | 控制Step是否执行 |
| `steps[].env` | Object | 否 | - | Step级环境变量 | - |
| `steps[].workingDirectory` | String | 否 | /workspace | 工作目录 | - |
| `steps[].continueOnError` | Boolean | 否 | false | 失败是否继续 | true/false |
| `steps[].timeout` | Number | 否 | - | Step超时时间(分钟) | - |
| `steps[].shell` | String | 否 | bash | Shell类型 | bash/sh/python/pwsh |

### run步骤

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `steps[].run` | String | **是** | - | 执行的命令或脚本 | 支持单行或多行 |

### checkout步骤

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `steps[].checkout` | String | **是** | - | 仓库引用 | self=当前仓库 |
| `steps[].clean` | Boolean | 否 | true | 是否清理工作目录 | true/false |
| `steps[].fetchDepth` | Number | 否 | 1 | Git拉取深度 | 0=完整历史 |
| `steps[].lfs` | Boolean | 否 | false | 是否拉取LFS文件 | true/false |
| `steps[].submodules` | Boolean/String | 否 | false | 是否拉取子模块 | false/true/recursive |
| `steps[].path` | String | 否 | - | 检出路径 | - |
| `steps[].ref` | String | 否 | - | 分支/标签/提交 | - |

### uses步骤

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `steps[].uses` | String | **是** | - | 插件路径@版本 | 如docker/build@v1 |
| `steps[].with` | Object | 否 | - | 插件参数 | key-value格式 |

---

## 🏁 收尾阶段属性 (finally.*)

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `finally[]` | Array | 否 | - | 收尾Job列表 | 总是执行,同jobs结构 |

---

## 📢 通知属性 (notices.*)

| 属性路径 | 类型 | 必填 | 默认值 | 描述 | 约束/规则 |
|---------|------|------|--------|------|----------|
| `notices[].type` | String | **是** | - | 通知类型 | email/wework-message/wework-chat |
| `notices[].content` | String | 否 | 系统模板 | 消息内容 | 支持上下文变量,wework支持markdown |
| `notices[].title` | String | 否 | 默认标题 | 消息标题 | type=email时可指定 |
| `notices[].receivers` | Array | 否 | 触发人 | 消息接收人 | email/wework-message时可指定 |
| `notices[].ccs` | Array | 否 | - | 邮件抄送人 | type=email时可指定 |
| `notices[].chat-id` | Array | 否 | - | 企业微信会话ID | type=wework-chat时必填 |
| `notices[].if` | String | 否 | ALWAYS | 条件执行 | FAILURE/SUCCESS/CANCELED/ALWAYS |

---

## 🔢 流水线变量 (variables.*)

### 变量定义格式

流水线变量的值**必须是 object 类型**,不能直接写字符串。格式如下:

```yaml
variables:
  # 运行时参数(可在启动时修改)
  PARAM_NAME:
    value: "default-value"
    allow-modify-at-startup: true
    props:
      type: selector           # 可选: 下拉选择器
      options:
        - id: option1
          label: 选项1
        - id: option2
          label: 选项2
  
  # 常量(不可修改)
  CONSTANT_NAME:
    value: "fixed-value"
    const: true
  
  # 普通变量
  VAR_NAME:
    value: "some-value"
```

**关键规则**:
- ❌ **错误写法**: `VAR_NAME: "value"` (schema 校验失败)
- ✅ **正确写法**: `VAR_NAME: { value: "value" }`
- 所有变量值必须包含 `value` 属性
- 运行时参数需加 `allow-modify-at-startup: true`
- 常量需加 `const: true`
- 下拉选择器需定义 `props.type: selector` 和 `options`

---

## 🔤 内置变量 (Built-in Variables)

### ci上下文

| 变量 | 类型 | 描述 |
|------|------|------|
| `ci.branch` | String | 当前分支名 |
| `ci.build_number` | Number | 构建号 |
| `ci.commit_sha` | String | 提交哈希 |
| `ci.commit_message` | String | 提交信息 |
| `ci.author` | String | 提交作者 |
| `ci.pipeline_id` | String | 流水线ID |
| `ci.head_ref` | String | PR头引用 |
| `ci.base_ref` | String | PR基引用 |
| `ci.pull_request_number` | Number | PR编号 |
| `ci.timestamp` | String | 时间戳 |
| `ci.event` | String | 事件类型 |
| `ci.repo_url` | String | 仓库URL |
| `ci.issue_*` | String | Issue相关信息 |
| `ci.review_*` | String | Review相关信息 |
| `ci.note_*` | String | Note相关信息 |

### 其他上下文

| 变量 | 类型 | 描述 |
|------|------|------|
| `github.ref` | String | Git引用 |
| `github.repository` | String | 仓库名称 |
| `github.event_name` | String | 事件名称 |
| `variables.<变量名>` | Any | 流水线变量 |
| `secrets.<凭据名>` | String | 凭据引用 |
| `steps.<step-id>.outputs.<变量名>` | String | Step输出变量 |
| `jobs.<job-id>.outputs.<变量名>` | String | Job输出变量 |
| `env.<变量名>` | String | 环境变量 |
| `matrix.<变量名>` | String | 矩阵变量 |

### 特殊变量

| 变量 | 描述 |
|------|------|
| `$GITHUB_OUTPUT` | 用于设置Step输出变量 |
| `$PIPELINE_STATUS` | 流水线状态(success/failure/cancelled) |
| `$PIPELINE_RESULT` | 流水线结果(succeeded/failed/cancelled) |
| `$BUILD_NO_OF_DAY` | 当天构建号 |
| `DATE:"format"` | 日期格式化 |

---

## 🎯 条件函数 (Condition Functions)

| 函数 | 返回类型 | 描述 |
|------|---------|------|
| `success()` | Boolean | 前面的Step/Job都成功 |
| `failure()` | Boolean | 任何前面的Step/Job失败 |
| `always()` | Boolean | 总是执行 |
| `cancelled()` | Boolean | 流水线被取消 |
| `startsWith(str, prefix)` | Boolean | 字符串以prefix开头 |
| `endsWith(str, suffix)` | Boolean | 字符串以suffix结尾 |
| `contains(str, substring)` | Boolean | 字符串包含substring |

---

## 📊 统计摘要

- **顶层属性**: 13个
- **触发器属性**: 50+个
- **并发控制属性**: 4个
- **阶段属性**: 2个
- **作业属性**: 40+个
- **步骤属性**: 20+个
- **通知属性**: 7个
- **内置变量**: 30+个
- **条件函数**: 7个

**总计约 170+ 个完整属性路径**
