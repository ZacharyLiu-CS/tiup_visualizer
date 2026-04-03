---
name: bk-ci-yaml
description: Use when creating, modifying, debugging, or querying pipeline YAML configurations. Covers triggers, stages, jobs, steps, variables, conditions, concurrency, and notifications. Provides decision tree, validation checklist, and common pitfall warnings.
---

# bk-ci-yaml Skill

## Overview

本Skill用于指导AI正确地创建、修改、调试和查询流水线YAML配置文件。提供完整的语法规范、决策树引导、校验机制和常见陷阱提示。

## When to Use

使用本Skill当用户请求:
- **创建**: "写/创建/生成一个流水线"、"帮我做个YAML"
- **修改**: "修改/添加/加个触发器/通知/..."、"改一下Job"
- **调试**: "为什么报错"、"YAML有问题"、"执行失败"
- **查询**: "怎么配置"、"语法是什么"、"支持哪些属性"

## Decision Tree (决策树)

根据用户意图自动选择工作流:

```
用户请求
├─ "写/创建/生成流水线" → [workflow-create]
├─ "修改/添加/改..."    → [workflow-modify]
├─ "报错/失败/调试..."   → [workflow-debug]
└─ "怎么/语法/支持..."   → [workflow-query]
```

---

## Workflow 1: Create (创建流水线)

### 触发条件
用户想创建新的YAML文件或从零开始编写流水线

### 执行步骤

#### 1. MANDATORY READ
**必须**完整读取以下文件,确保语法正确:
- `reference/yaml-schema.md` (完整Schema速查表)

#### 2. 了解需求
询问用户:
- 流水线用途(构建/测试/部署)
- 触发条件(push/mr/tag/schedules)
- 运行环境(docker/k8s/macos/windows)
- 是否需要多环境部署

#### 3. 选择模板
根据需求选择合适的模板:
- **简单构建测试**: `templates/basic-ci.yaml`
- **构建+部署**: `templates/build-deploy.yaml`
- **矩阵构建**: `templates/matrix-build.yaml`
- **MR校验**: `templates/mr-validation.yaml`

#### 4. 生成YAML
基于模板生成,并调整:
- 修改name、触发器、stages
- 配置pool(构建环境)
- 添加所需的steps
- 配置variables和secrets
- 添加notices(通知)

#### 5. 执行校验清单
生成后必须检查:
- [ ] 顶层结构完整(name, on, stages)
- [ ] **variables值必须是object**(包含value属性,不能直接写字符串)
- [ ] **stages内嵌jobs**(jobs是stage的子属性,是object不是数组)
- [ ] **stages[].label**使用有效值: Build/Test/Deploy/Approve(或不设置)
- [ ] Job ID唯一(同一stage内)
- [ ] dependsOn无循环依赖
- [ ] 变量引用语法正确(`${{ variables.xxx }}` 不是 `${{ xxx }}`)
- [ ] 条件表达式中引用变量用`variables.XXX == 'xxx'`
- [ ] 触发器属性不含上下文变量
- [ ] 通知使用`notices`不是`notifications`
- [ ] workspace.reuse配合dependsOn
- [ ] finally的Job不依赖普通Stage的Job

#### 6. 提示常见陷阱
提醒用户注意:
- 触发器属性不支持变量
- 矩阵变量引用用`${{ matrix.xxx }}`
- 跨Job传递输出需要声明outputs

---

## Workflow 2: Modify (修改流水线)

### 触发条件
用户想修改现有YAML文件,添加或调整配置

### 执行步骤

#### 1. 读取现有YAML
使用`read_file`读取用户的YAML文件

#### 2. 识别修改类型
判断用户想修改什么:
- **触发器**: 读取`reference/triggers.md`
- **Job配置**: 读取`reference/stages-jobs-steps.md`
- **变量**: 读取`reference/variables.md`
- **条件**: 读取`reference/conditions.md`
- **并发**: 读取`reference/concurrency.md`
- **通知**: 读取`reference/notifications.md`

#### 3. 应用修改
使用`replace_in_file`精确修改:
- 保持原有缩进和格式
- 只修改相关部分
- 不改动无关配置

#### 4. 执行校验清单
修改后检查:
- [ ] 语法正确性
- [ ] 新增配置与现有配置兼容
- [ ] 依赖关系正确
- [ ] 变量引用正确

#### 5. 提示相关陷阱
根据修改内容提醒常见错误

---

## Workflow 3: Debug (调试流水线)

### 触发条件
用户报告YAML有错误、执行失败或不符合预期

### 执行步骤

#### 1. 读取YAML和错误信息
- 读取用户的YAML文件
- 获取错误日志或失败描述

#### 2. 加载诊断文档
读取:
- `reference/common-pitfalls.md` (常见陷阱)
- 相关的语法参考文档

#### 3. 诊断问题
检查常见错误:
- 触发器配置错误
- 变量引用语法错误
- Job依赖循环
- 条件表达式错误
- workspace.reuse未配置dependsOn
- notices拼写错误
- finally依赖普通Job

#### 4. 提供解决方案
- 指出具体错误位置
- 提供正确的配置示例
- 说明为什么错误

---

## Workflow 4: Query (查询语法)

### 触发条件
用户询问如何配置某个功能或查询语法

### 执行步骤

#### 1. 识别查询主题
判断用户想了解:
- 触发器 → `reference/triggers.md`
- 变量 → `reference/variables.md`
- 条件 → `reference/conditions.md`
- Job/Step → `reference/stages-jobs-steps.md`
- 并发 → `reference/concurrency.md`
- 通知 → `reference/notifications.md`
- 所有属性 → `reference/yaml-schema.md`

#### 2. 加载对应文档
只读取相关的参考文档,避免加载所有内容

#### 3. 提供答案
- 给出语法说明
- 提供实用示例
- 说明注意事项

---

## Validation Checklist (校验清单)

生成或修改YAML后,**必须**执行以下检查:

### 结构完整性
- [ ] 包含必需的顶层属性: `stages`
- [ ] **stages 内嵌 jobs**(jobs 是 stage 的子属性,是 object 不是数组)
- [ ] 每个 stage 包含: `name`, `jobs`
- [ ] 每个 job 包含: `steps`(至少一个)
- [ ] 每个Step有有效的操作: `run`/`checkout`/`uses`

### 变量定义
- [ ] **variables 值必须是 object**: 包含 `value` 属性,不能直接写字符串
- [ ] 运行时参数加 `allow-modify-at-startup: true`
- [ ] 常量加 `const: true`
- [ ] 下拉选择器定义 `props.type: selector` 和 `options`

### 命名唯一性
- [ ] Stage名称唯一
- [ ] 同一Stage内Job ID唯一
- [ ] Step的id(如果有)唯一

### 依赖关系
- [ ] `dependsOn`引用的Job存在
- [ ] 无循环依赖
- [ ] `workspace.reuse`配合`dependsOn`使用
- [ ] finally的Job不依赖普通Stage的Job

### 变量引用
- [ ] 流水线变量用`${{ variables.xxx }}`(不是`${{ xxx }}`)
- [ ] 条件表达式中用`variables.XXX == 'xxx'`(不带`${{ }}`)
- [ ] 环境变量用`$VAR`或`${VAR}`
- [ ] 矩阵变量用`${{ matrix.xxx }}`
- [ ] Step输出用`${{ steps.xxx.outputs.yyy }}`
- [ ] Job输出用`${{ jobs.xxx.outputs.yyy }}`

### 条件表达式
- [ ] 包含`!`的表达式用引号包裹
- [ ] 条件函数拼写正确: `success()`, `failure()`, `always()`, `cancelled()`
- [ ] 字符串比较用`==`或`!=`

### 触发器
- [ ] 触发器属性不使用上下文变量
- [ ] schedules的cron不包含秒字段
- [ ] schedules的branches不超过3个且不用通配符

### 通知
- [ ] 使用`notices`不是`notifications`
- [ ] `notices[].if`只用FAILURE/SUCCESS/CANCELED/ALWAYS
- [ ] type为wework-chat时提供chat-id

### 并发控制
- [ ] queue-length和queue-timeout只在cancel-in-progress=false时配置

---

## Common Pitfalls (常见陷阱速查)

⚠️ 生成YAML后提醒用户注意这些常见错误:

1. **variables值必须是object** - 不能直接写字符串,必须`{ value: "xxx" }`
2. **stages内嵌jobs** - jobs是stage的子属性(object Map),不是顶层数组
3. **stages[].label只接受特定值** - Build/Test/Deploy/Approve,或不设置
4. **条件中引用变量不加`${{ }}`** - 写`variables.XXX == 'xxx'`而非`${{ variables.XXX }} == 'xxx'`
5. **触发器属性不支持变量** - 必须写死或用通配符
6. **通知是`notices`不是`notifications`**
7. **workspace.reuse必须配合dependsOn**
8. **finally的Job不能依赖普通Stage的Job**
9. **条件表达式包含`!`需要引号**
10. **矩阵变量引用用`${{ matrix.xxx }}`**
11. **跨Job传递输出需要声明outputs**
12. **schedules的cron不支持秒级别**
13. **queue配置只在cancel-in-progress=false时生效**

完整陷阱列表见: `reference/common-pitfalls.md`

---

## Quick Reference (快速参考)

### 基础结构
```yaml
version: v3.0
name: 流水线名称

variables:
  VAR_NAME:
    value: "默认值"
    allow-modify-at-startup: true  # 运行时参数

stages:
  - name: stage_name
    label: Build  # Build/Test/Deploy/Approve
    jobs:
      job_id:     # job ID 作为 key
        name: 作业名称
        if: variables.PARAM == 'value'  # 条件(不带${{ }})
        steps: [...]
```

### 常用变量
```yaml
${{ ci.branch }}          # 分支名
${{ ci.build_number }}    # 构建号
${{ ci.commit_sha }}      # 提交SHA
${{ github.ref }}         # Git引用
${{ variables.xxx }}      # 流水线变量
${{ secrets.xxx }}        # 凭据
${{ matrix.xxx }}         # 矩阵变量
```

### 条件函数
```yaml
success()    # 前面都成功
failure()    # 任何失败
always()     # 总是执行
cancelled()  # 流水线被取消
```

### 关键规则
1. **variables 值必须是 object**: `{ value: "xxx" }`
2. **stages 内嵌 jobs**: jobs 是 stage 子属性
3. **条件中引用变量**: `variables.XXX == 'xxx'` (不带 `${{ }}`)
4. **流水线变量**: `${{ variables.XXX }}` (必须带 variables. 前缀)

### 执行顺序
```
Stage1 → Stage2 → Stage3 → ... → finally
```
- 同一Stage内的Job并行执行
- 使用dependsOn串行
- 使用mutex互斥

---

## Files Structure

```
bk-ci-yaml/
├── SKILL.md                    # 本文件
├── skills_summary.md           # 中文摘要
├── LICENSE.txt                 # 许可证
│
├── reference/                  # 语法参考文档
│   ├── yaml-schema.md          # 完整Schema速查表
│   ├── triggers.md             # 触发器语法
│   ├── stages-jobs-steps.md    # Stage+Job+Step语法
│   ├── variables.md            # 变量系统
│   ├── conditions.md           # 条件执行
│   ├── concurrency.md          # 并发控制
│   ├── notifications.md        # 通知配置
│   └── common-pitfalls.md      # 常见陷阱
│
└── templates/                  # 常用模板
    ├── basic-ci.yaml           # 基础CI模板
    ├── build-deploy.yaml       # 构建+部署模板
    ├── matrix-build.yaml       # 矩阵构建模板
    └── mr-validation.yaml      # MR校验模板
```

---

## Remember

1. **创建流水线前必须读取yaml-schema.md**
2. **生成后必须执行校验清单**
3. **修改时只改相关部分,保持格式**
4. **调试时优先检查common-pitfalls.md**
5. **查询时按需加载,避免全部加载**
