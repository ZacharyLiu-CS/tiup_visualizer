/**
 * AI Analysis Prompt Template for Yuanbao
 * 
 * Available placeholders:
 *   {{clusterName}}   - Cluster name
 *   {{clusterVersion}} - Cluster version (e.g. v7.5.0)
 *   {{clusterType}}   - Cluster type
 *   {{componentRole}}  - Component role (tikv, pd, tidb, etc.)
 *   {{componentId}}    - Component ID (e.g. 192.168.1.1:20160)
 *   {{componentHost}}  - Component host IP
 *   {{componentStatus}} - Component status (Up, Down, etc.)
 *   {{logFilename}}    - Log filename (e.g. tikv.log)
 *   {{deployDir}}      - Component deploy directory
 *   {{dataDir}}        - Component data directory
 *   {{ports}}          - Component ports
 */

export const AI_PROMPT_TEMPLATE = `你是一位 TiDB/TiKV 数据库集群运维专家。请帮我分析以下日志文件，识别其中的异常、错误、警告信息，并给出诊断建议。

## 集群信息
- 集群名称: {{clusterName}}
- 集群版本: {{clusterVersion}}
- 集群类型: {{clusterType}}

## 组件信息
- 组件角色: {{componentRole}}
- 组件 ID: {{componentId}}
- 主机地址: {{componentHost}}
- 当前状态: {{componentStatus}}
- 端口: {{ports}}
- 部署目录: {{deployDir}}
- 数据目录: {{dataDir}}

## 日志文件
- 文件名: {{logFilename}}

请对上传的日志文件进行以下分析：

1. **错误与异常识别**: 找出所有 ERROR、FATAL、PANIC 级别的日志，分析其根因
2. **警告信息**: 列出关键 WARN 信息，评估其严重程度
3. **性能问题**: 检查是否存在慢查询、延迟抖动、Region 调度异常等性能相关问题
4. **资源状况**: 分析是否有内存、磁盘、网络等资源瓶颈迹象
5. **建议措施**: 基于分析结果，给出具体的排障步骤和优化建议

请用中文回复，以结构化的格式输出分析报告。`

/**
 * Fill the prompt template with actual component/cluster data.
 * @param {Object} params - Template parameters
 * @returns {string} Filled prompt string
 */
export function buildAIPrompt(params) {
  let prompt = AI_PROMPT_TEMPLATE
  const placeholders = {
    '{{clusterName}}': params.clusterName || '-',
    '{{clusterVersion}}': params.clusterVersion || '-',
    '{{clusterType}}': params.clusterType || '-',
    '{{componentRole}}': params.componentRole || '-',
    '{{componentId}}': params.componentId || '-',
    '{{componentHost}}': params.componentHost || '-',
    '{{componentStatus}}': params.componentStatus || '-',
    '{{logFilename}}': params.logFilename || '-',
    '{{deployDir}}': params.deployDir || '-',
    '{{dataDir}}': params.dataDir || '-',
    '{{ports}}': params.ports || '-',
  }
  for (const [key, value] of Object.entries(placeholders)) {
    prompt = prompt.replaceAll(key, value)
  }
  return prompt
}
