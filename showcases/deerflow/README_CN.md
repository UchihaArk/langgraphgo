# DeerFlow - 深度研究智能体 (Go 复刻版)

这是 [ByteDance DeerFlow](https://github.com/bytedance/deer-flow) 深度研究智能体的 Go 语言实现版本，基于 [langgraphgo](https://github.com/smallnest/langgraphgo) 和 [langchaingo](https://github.com/tmc/langchaingo) 构建。

DeerFlow 是一个多智能体系统，旨在对给定主题进行深度研究。它能够规划研究策略，执行搜索步骤（模拟或真实），并综合生成一份详尽的报告。

## 功能特性

- **多智能体架构**：使用状态图编排 `Planner`（规划者）、`Researcher`（研究员）和 `Reporter`（报告员）智能体。
- **Web 界面**：提供现代化的暗色主题 Web UI，利用服务器发送事件 (SSE) 实现实时状态更新。
- **CLI 支持**：支持直接从命令行运行以进行快速查询。
- **可扩展性**：基于 `langgraphgo` 构建，易于添加新节点、工具或复杂的控制流。

## 前置要求

- Go 1.23 或更高版本
- OpenAI 兼容的 API Key (例如 OpenAI, DeepSeek)

## 配置

设置以下环境变量：

```bash
export OPENAI_API_KEY="your-api-key"

# 可选：如果使用 DeepSeek 或其他兼容提供商
export OPENAI_API_BASE="https://api.deepseek.com/v1" 
```

## 使用方法

### Web 界面 (推荐)

编译并运行应用程序：

```bash
go build -o deerflow ./showcases/deerflow
./deerflow
```

默认情况下，服务器将在 `http://localhost:8085` 启动。

### Nginx 配置

项目包含了一个示例 `nginx.conf` 文件，用于配置反向代理。这对于生产部署或需要使用标准端口（如 80）的情况很有用。

主要配置点：
- 代理到 `http://localhost:8085`
- 关闭缓冲以支持 SSE (Server-Sent Events) 流式输出

你可以使用以下命令启动 Nginx（假设你已安装 Nginx）：

```bash
nginx -c $(pwd)/showcases/deerflow/nginx.conf
```

或者将配置内容复制到你的 Nginx 配置文件中。

### 命令行接口 (CLI)

直接从终端运行查询：

```bash
./deerflow "固态电池的最新进展是什么？"
```

## 项目结构

- **`main.go`**：程序入口。处理 CLI 参数并启动 HTTP 服务器。
- **`graph.go`**：定义 `State` 结构体和图拓扑（节点和边）。
- **`nodes.go`**：包含 `Planner`、`Researcher` 和 `Reporter` 的实现逻辑。
- **`web/`**：包含前端资源 (HTML, CSS, JS)。

## 架构

智能体遵循顺序工作流：

1.  **Planner**：将用户查询分解为逐步的研究计划。
2.  **Researcher**：遍历计划，为每一步收集信息。
3.  **Reporter**：将收集到的所有信息综合成最终的格式化报告。

## 许可证

MIT
