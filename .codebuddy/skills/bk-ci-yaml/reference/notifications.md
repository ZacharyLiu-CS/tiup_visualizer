# 通知配置参考

## 基本语法

```yaml
notices:
  - type: <通知类型>
    content: <消息内容>
    title: <消息标题>
    receivers: <接收人列表>
    ccs: <抄送人列表>
    chat-id: <企业微信会话ID>
    if: <条件>
```

## 参数说明

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| type | ✓ | string | 通知类型: email/wework-message/wework-chat |
| content | - | string | 消息内容,可使用变量,wework支持markdown |
| title | - | string | 消息标题,type=email时可指定 |
| receivers | - | array | 接收人,email/wework-message时可指定 |
| ccs | - | array | 抄送人,type=email时可指定 |
| chat-id | - | array | 企业微信会话ID,type=wework-chat时必填 |
| if | - | string | 条件: FAILURE/SUCCESS/CANCELED/ALWAYS(默认) |

## 实用示例

### 使用默认模板
```yaml
notices:
  - type: email
  - type: wework-message
```

### 自定义通知内容
```yaml
notices:
  # 失败时发送邮件
  - type: email
    title: 构建失败 - ${{ ci.branch }}
    content: |
      流水线执行失败
      分支: ${{ ci.branch }}
      提交: ${{ ci.commit_sha }}
      触发人: ${{ ci.author }}
    receivers:
      - devops-team@company.com
      - ${{ ci.author }}
    ccs:
      - manager@company.com
    if: FAILURE
  
  # 成功时发送企业微信
  - type: wework-message
    title: 构建成功
    content: |
      **流水线执行成功**
      > 分支: ${{ ci.branch }}
      > 构建号: #${{ ci.build_number }}
    receivers:
      - user1
      - user2
    if: SUCCESS
  
  # 发送到企业微信群
  - type: wework-chat
    content: |
      ### 流水线通知
      **状态**: 执行完成
      **分支**: ${{ ci.branch }}
      **提交**: ${{ ci.commit_sha }}
      **触发人**: ${{ ci.author }}
    chat-id:
      - chat_id_123456
    if: ALWAYS
```

### 多渠道通知
```yaml
name: 完整通知示例

on:
  push:
    branches: [master, develop]

jobs:
  - stage: build
    job: build-app
    steps:
      - run: npm run build
      - run: npm test

notices:
  # 失败邮件
  - type: email
    title: ❌ 构建失败
    content: |
      流水线执行失败,请及时处理
      分支: ${{ ci.branch }}
      构建号: ${{ ci.build_number }}
      提交信息: ${{ ci.commit_message }}
    receivers:
      - ${{ ci.author }}
    if: FAILURE
  
  # 成功企业微信
  - type: wework-message
    title: ✅ 构建成功
    content: 流水线执行成功
    receivers:
      - ${{ ci.author }}
    if: SUCCESS
  
  # 总是通知到群
  - type: wework-chat
    content: |
      流水线: **${{ ci.pipeline_id }}**
      状态: ${{ ci.status }}
      分支: ${{ ci.branch }}
    chat-id:
      - your-chat-id
    if: ALWAYS
```

## 注意事项

1. **条件判断**: if参数不支持自定义变量,只支持FAILURE/SUCCESS/CANCELED/ALWAYS
2. **变量支持**: content和title可以使用`ci`、`variables`等上下文变量
3. **Markdown**: wework-message和wework-chat支持部分Markdown语法
4. **获取chat-id**: 将服务号"Stream消息通知"加到群里,@Stream消息通知 会话ID即可获得
