# BK-CI YAML Skill 使用指南

## 简介

BK-CI YAML Skill 是一个专门用于流水线YAML配置的参考技能,帮助AI正确地创建、修改、调试和查询流水线配置文件。

## 适用场景

当你需要以下操作时,应该使用此Skill:

1. **创建新流水线**: 从零开始编写YAML配置
2. **修改现有流水线**: 添加触发器、Job、Step、通知等
3. **调试流水线**: 排查YAML语法错误或执行失败问题
4. **查询语法**: 了解如何配置特定功能

## 核心能力

### 1. 智能决策树
根据用户意图自动选择最合适的工作流:
- 创建流水线 → 加载模板快速生成
- 修改流水线 → 精确定位修改目标
- 调试流水线 → 诊断常见错误
- 查询语法 → 按需加载参考文档

### 2. 完整语法规范
提供170+个属性的完整参考:
- 触发器(push/tag/mr/schedules等)
- Stage/Job/Step完整语法
- 变量系统(variables/env/secrets)
- 条件执行(if/success()/failure())
- 并发控制(concurrency/mutex)
- 通知配置(notices)

### 3. 常用模板库
4个开箱即用的YAML模板:
- `basic-ci.yaml`: 基础构建测试
- `build-deploy.yaml`: 构建+多环境部署
- `matrix-build.yaml`: 矩阵策略多版本测试
- `mr-validation.yaml`: MR代码审查和质量检查

### 4. 校验机制
内置完整的校验清单:
- 结构完整性检查
- 命名唯一性验证
- 依赖关系校验
- 变量引用语法检查
- 条件表达式验证

### 5. 陷阱提示
汇总15+个常见错误:
- 触发器属性不支持变量
- 通知配置是`notices`不是`notifications`
- workspace.reuse必须配合dependsOn
- finally的Job不能依赖普通Stage的Job
- 等等...

## 使用方法

### 创建新流水线
```
用户: "帮我写一个构建和部署的流水线"

AI工作流:
1. 读取 yaml-schema.md (MANDATORY)
2. 了解需求(触发条件、环境、部署目标)
3. 选择模板 build-deploy.yaml
4. 基于模板生成并调整
5. 执行校验清单
6. 提示常见陷阱
```

### 修改现有流水线
```
用户: "帮我加个MR触发器"

AI工作流:
1. 读取现有YAML文件
2. 加载 triggers.md 参考文档
3. 使用 replace_in_file 精确修改
4. 校验语法正确性
5. 提示相关注意事项
```

### 调试流水线
```
用户: "我的YAML报错了"

AI工作流:
1. 读取YAML和错误信息
2. 加载 common-pitfalls.md
3. 诊断问题(对照常见陷阱)
4. 提供解决方案
```

### 查询语法
```
用户: "如何配置矩阵构建?"

AI工作流:
1. 识别查询主题(矩阵策略)
2. 加载 stages-jobs-steps.md
3. 提供语法说明和示例
```

## 文件结构

```
bk-ci-yaml/
├── SKILL.md                    # 核心入口(决策树+工作流)
├── skills_summary.md           # 本文件
├── LICENSE.txt                 # 许可证
│
├── reference/                  # 精简参考文档
│   ├── yaml-schema.md          # 完整Schema速查表(170+属性)
│   ├── triggers.md             # 触发器详细语法
│   ├── stages-jobs-steps.md    # Stage+Job+Step语法
│   ├── variables.md            # 变量系统详解
│   ├── conditions.md           # 条件执行与表达式
│   ├── concurrency.md          # 并发控制(流水线级和Job级)
│   ├── notifications.md        # 通知配置
│   └── common-pitfalls.md      # 常见陷阱汇总
│
└── templates/                  # 常用YAML模板
    ├── basic-ci.yaml           # 基础CI(构建+测试)
    ├── build-deploy.yaml       # 构建+多环境部署
    ├── matrix-build.yaml       # 矩阵策略构建
    └── mr-validation.yaml      # MR校验流水线
```

## 优势

相比直接读取原始文档(35个文件,7000+行):

| 对比项 | 原始文档 | bk-ci-yaml Skill |
|--------|---------|------------------|
| 文件数量 | 35个 | 8个精简参考文档 |
| 加载量 | 全部加载(7000+行) | 按需加载(每次200-500行) |
| 查找效率 | 需要搜索多个文件 | 决策树直接定位 |
| 模板支持 | 无 | 4个开箱即用模板 |
| 校验机制 | 无 | 内置完整校验清单 |
| 陷阱提示 | 散落在文档中 | 集中汇总15+个 |

## 最佳实践

1. **创建前必读**: 生成任何YAML前必须读取`yaml-schema.md`
2. **基于模板**: 优先使用模板,减少从零编写的复杂度
3. **精确修改**: 修改时只改相关部分,保持原有格式
4. **校验清单**: 生成或修改后必须执行完整的校验清单
5. **陷阱提示**: 完成后主动提示用户注意常见错误

## 典型使用示例

### 示例1: 创建基础CI流水线
```yaml
# 基于 basic-ci.yaml 模板生成
name: My CI Pipeline

on:
  push:
    branches: [master, develop]

stages:
  - name: build
  - name: test

jobs:
  - stage: build
    job: build-app
    pool:
      name: docker
      container: node:16
    steps:
      - checkout: self
      - run: npm ci
      - run: npm run build
```

### 示例2: 添加MR触发器
```yaml
# 在现有YAML中添加
on:
  mr:
    target-branches: [master]
    block-mr: true
```

### 示例3: 诊断常见错误
```
❌ 错误: notifications 未定义
✅ 正确: 应该使用 notices 而不是 notifications

❌ 错误: workspace.reuse 未生效
✅ 正确: 必须配合 dependsOn 使用
```

## 总结

bk-ci-yaml Skill通过决策树、精简参考、模板库和校验机制,将35个文档的知识整合为高效的工作流,帮助AI快速、准确地完成流水线开发任务。
