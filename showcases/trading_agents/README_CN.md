# Trading Agents - 交易代理系统

基于 LangGraph Go 和 LangChain Go 构建的多代理 LLM 驱动的金融交易框架。本项目是 [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents) 的完整 Go 语言实现。

## 📊 概述

Trading Agents 模拟了一个专业的交易公司，由专业的 AI 代理协同工作来分析市场并做出明智的交易决策。该系统结合了基本面分析、情绪分析、技术分析和风险管理，提供全面的交易建议。

## 🎯 项目成果

### 1. **核心代理系统**（7 个专业代理）
- **基本面分析师**：分析公司财务和估值
- **情绪分析师**：评估社交媒体和公众情绪
- **技术分析师**：执行技术指标分析
- **新闻分析师**：监控新闻和宏观经济因素
- **看涨研究员**：提供乐观的视角
- **看跌研究员**：识别风险和担忧
- **风险管理员**：评估和管理交易风险
- **交易员**：综合所有报告做出最终决策

### 2. **三个完整的接口**
- **后端 API 服务器**：具有健康检查和分析端点的 RESTful API
- **CLI 工具**：用于终端分析的命令行界面
- **Web 仪表板**：现代化、响应式的 Web 界面

### 3. **市场数据集成**
- Alpha Vantage API 集成
- 实时报价和公司信息
- 技术指标计算
- 情绪数据收集

## 🏗️ 架构

### 代理团队

1. **分析师团队**
   - **基本面分析师**：评估公司财务和绩效指标
   - **情绪分析师**：分析社交媒体和公众情绪
   - **新闻分析师**：监控全球新闻和宏观经济指标
   - **技术分析师**：使用技术指标进行价格趋势分析

2. **研究团队**
   - **看涨研究员**：倡导买入机会
   - **看跌研究员**：识别潜在风险和卖出信号

3. **交易员**
   - 综合所有分析师和研究员报告
   - 做出最终交易决策

4. **风险管理团队**
   - 监控投资组合风险敞口
   - 实施风险缓解策略
   - 确保符合风险承受能力

### 代理管道流程

```
1. 数据收集
   ├─> 市场报价
   ├─> 公司基本面
   ├─> 技术指标
   └─> 情绪数据

2. 分析师团队（概念上并行）
   ├─> 基本面分析师
   ├─> 情绪分析师
   └─> 技术分析师

3. 研究团队
   ├─> 看涨研究员
   └─> 看跌研究员

4. 风险管理
   └─> 风险管理员

5. 最终决策
   └─> 交易员（综合所有报告）
```

## ✨ 功能特性

- ✅ 多代理协作分析
- ✅ 实时市场数据集成
- ✅ 后端 API 服务器
- ✅ 命令行界面（CLI）
- ✅ 基于 Web 的仪表板
- ✅ 全面的日志记录和跟踪
- ✅ 可配置的风险承受能力
- ✅ 多时间框架支持
- ✅ 详细模式显示代理报告

## 📁 项目结构

```
trading_agents/
├── README.md              # 本文件
├── types.go              # 核心类型定义
├── graph.go              # 主交易图工作流
├── agents/               # 代理实现
│   ├── fundamentals_analyst.go
│   ├── sentiment_analyst.go
│   ├── technical_analyst.go
│   ├── trader.go
│   ├── risk_manager.go
│   └── researchers.go
├── tools/                # 市场数据工具
│   └── market_data.go
├── backend/              # API 服务器
│   └── main.go
├── cli/                  # CLI 工具
│   └── main.go
├── web/                  # Web 界面
│   ├── index.html
│   ├── style.css
│   └── app.js
└── examples/             # 使用示例
    └── simple_analysis.go
```

## 📈 统计数据

- **Go 代码总量**：约 2,000 行
- **文件数量**：17 个
- **实现的代理**：7 个专业代理
- **接口**：3 个（API、CLI、Web）
- **二进制文件大小**：
  - 后端：9.2 MB
  - CLI：8.6 MB

## 🚀 快速开始

### 前置要求

- Go 1.21+
- OpenAI API 密钥（用于 LLM）
- Alpha Vantage API 密钥（用于市场数据，可选）

### 安装

```bash
# 设置环境变量
export OPENAI_API_KEY="your-openai-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"  # 可选

# 从 langgraphgo 根目录
cd showcases/trading_agents

# 构建后端服务器
go build -o ../../bin/trading-agents ./backend

# 构建 CLI 工具
go build -o ../../bin/trading-cli ./cli
```

### 运行后端

```bash
./bin/trading-agents --port 8080
```

服务器将在 `http://localhost:8080` 上启动

### 运行 CLI

```bash
# 分析股票
./bin/trading-cli -cmd analyze -symbol AAPL -verbose

# 获取交易建议
./bin/trading-cli -cmd recommend -symbol AAPL -capital 10000
```

### 运行 Web 界面

```bash
# 先启动后端，然后
cd showcases/trading_agents/web
# 在浏览器中打开 index.html
```

或使用简单的 HTTP 服务器：

```bash
cd showcases/trading_agents/web
python3 -m http.server 3000
# 然后在浏览器中打开 http://localhost:3000
```

## 📚 使用指南

### 后端 API

#### 健康检查

```bash
curl http://localhost:8080/health
```

#### 完整分析

```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "AAPL",
    "capital": 10000,
    "risk_tolerance": "moderate",
    "timeframe": "1D"
  }'
```

#### 快速检查

```bash
curl -X POST http://localhost:8080/api/quick-check \
  -H "Content-Type: application/json" \
  -d '{"symbol": "TSLA"}'
```

### CLI 选项

- `-cmd`：命令（analyze、recommend、quick）
- `-symbol`：股票代码（必需）
- `-capital`：可用资金（美元，默认：10000）
- `-risk-level`：风险承受能力（low、moderate、high）
- `-timeframe`：交易时间框架（5m、1H、1D、1W）
- `-verbose`：显示详细的代理报告
- `-json`：以 JSON 格式输出

### CLI 示例

#### 完整分析

```bash
./bin/trading-cli -cmd analyze -symbol AAPL -verbose
```

**详细模式输出示例**：

```
📊 开始分析 AAPL...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔍 正在收集市场数据...
✓ 获取当前报价
✓ 获取公司基本面信息
✓ 计算技术指标
✓ 收集情绪数据

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 基本面分析师报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Apple Inc. (AAPL)
当前价格: $178.50
市值: $2.8T
市盈率: 29.5

公司财务状况强劲，现金流充裕...
估值水平：适中
建议：轻度看涨

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
😊 情绪分析师报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
社交媒体情绪: 积极 (75%)
新闻情绪: 中性偏积极 (62%)

最新趋势：
- iPhone 15 销量强劲
- 服务收入增长稳定
...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📈 技术分析师报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
趋势方向: 上升
支撑位: $175.00
阻力位: $182.00

技术指标：
- RSI(14): 58 (中性)
- MACD: 看涨交叉
- 移动平均线: 价格高于 50 日和 200 日均线
...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🐂 看涨研究员报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
看涨观点：
1. 强劲的产品周期
2. 服务业务增长
3. 股票回购计划
4. 技术突破即将到来
...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🐻 看跌研究员报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
看跌担忧：
1. 估值略高
2. 宏观经济不确定性
3. 竞争压力
...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚖️ 风险管理员报告
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
风险评估: 中等 (45/100)

建议的仓位大小: 8.5% 的投资组合
止损建议: $170.00 (-4.8%)
获利目标: $190.00 (+6.4%)
...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💼 交易员最终决策
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
综合所有分析后的建议...

最终建议: 买入 🟢
信心水平: 72%
建议仓位: 48 股
总价值: $8,568.00

理由：
- 基本面强劲
- 技术面向好
- 情绪积极
- 风险可控

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

#### 交易建议

```bash
./bin/trading-cli -cmd recommend -symbol GOOGL -capital 50000 -risk-level low
```

#### 快速检查

```bash
./bin/trading-cli -cmd quick -symbol TSLA
```

### 示例工作流程

#### 日内交易工作流程

```bash
# 对多只股票进行快速检查
./bin/trading-cli -cmd quick -symbol AAPL
./bin/trading-cli -cmd quick -symbol GOOGL
./bin/trading-cli -cmd quick -symbol TSLA

# 对最佳候选进行完整分析
./bin/trading-cli -cmd analyze -symbol AAPL -timeframe 5m -verbose
```

#### 投资分析工作流程

```bash
# 保守的长期投资
./bin/trading-cli -cmd recommend \
  -symbol MSFT \
  -capital 100000 \
  -risk-level low \
  -timeframe 1W \
  -verbose
```

### API 集成示例

```python
import requests

# 分析股票
response = requests.post('http://localhost:8080/api/analyze', json={
    'symbol': 'AAPL',
    'capital': 50000,
    'risk_tolerance': 'moderate'
})

result = response.json()
print(f"建议: {result['recommendation']}")
print(f"信心: {result['confidence']}%")
print(f"仓位大小: {result['position_size']} 股")
```

## 📊 理解输出

### 建议类型

- **BUY** 🟢：识别出强烈的买入机会
- **SELL** 🔴：卖出建议或做空机会
- **HOLD** 🟡：维持当前头寸或保持观望

### 信心评分

- **80-100%**：非常高的信心，所有代理的信号都很强
- **60-80%**：良好的信心，大多数代理同意
- **40-60%**：中等信心，信号混合
- **低于 40%**：低信心，信号冲突

### 风险评分

- **0-30**：低风险，保守交易
- **30-70**：中等风险，平衡方法
- **70-100**：高风险，激进交易

### 代理报告

每个分析包括来自以下方面的报告：

1. **基本面分析师**：公司财务和估值
2. **情绪分析师**：社交媒体和公众情绪
3. **技术分析师**：图表模式和指标
4. **看涨研究员**：积极的视角和机会
5. **看跌研究员**：风险和警示信号
6. **风险管理员**：风险评估和缓解策略

**交易员**综合所有报告做出最终建议。

## 🛠️ 技术实现

### 状态管理
- 基于映射的状态流经管道
- 每个代理丰富状态
- 最终状态包含所有报告和决策

### LLM 集成
- OpenAI GPT-4 用于代理推理
- 温度控制的响应
- 结构化输出解析

### LangGraph Go 集成
- 通过图节点进行状态管理
- 代理管道的顺序执行
- 清晰的关注点分离

## 🔧 故障排除

### "需要 API 密钥"错误

确保已设置 OpenAI API 密钥：
```bash
export OPENAI_API_KEY="your-key-here"
```

### "分析失败"错误

- 检查您的互联网连接
- 验证股票代码是否有效
- 确保后端服务器正在运行（对于 Web 界面）

### 后端服务器问题

检查服务器是否正在运行：
```bash
curl http://localhost:8080/health
```

查看服务器日志进行调试：
```bash
./bin/trading-agents --verbose
```

## 💡 获得最佳结果的提示

1. **使用有效的代码**：确保使用正确的股票代码（例如，AAPL 代表 Apple，而不是 APPLE）

2. **设置实际资金**：使用实际资金金额以获得准确的仓位大小

3. **匹配风险承受能力**：选择与您实际风险承受能力相匹配的风险水平

4. **审查所有报告**：不要只看建议 - 阅读详细分析

5. **考虑背景**：分析是时点性的。市场条件变化迅速。

6. **结合研究**：将此作为研究过程中的众多工具之一

## 🎓 教育价值

本项目演示了：
- 多代理系统架构
- LangGraph Go 工作流编排
- LangChain Go LLM 集成
- RESTful API 设计
- 现代 Web 界面开发
- 生产就绪的 Go 应用程序结构

## ⚠️ 重要免责声明

- **仅用于教育目的**：不是财务建议
- **研究工具**：用于学习和实验
- **无责任**：使用风险自负
- **专业建议**：做出投资决策前请咨询合格的财务顾问

**此框架仅用于研究和教育目的。它不作为财务、投资或交易建议。在做出投资决策之前，请务必咨询合格的金融专业人士。**

## 🔮 未来增强

潜在改进：
- 实时 WebSocket 流
- 历史回测
- 投资组合管理
- 多时间框架分析
- 机器学习集成
- 数据库持久化
- 用户认证
- 模拟交易模式

## 🙏 致谢

- 灵感来自 [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents)
- 使用 [LangGraph Go](https://github.com/smallnest/langgraphgo) 构建
- 由 [LangChain Go](https://github.com/tmc/langchaingo) 提供支持
- 来自 Alpha Vantage 的市场数据

## 📝 参考资料

- [TradingAgents 论文 (arXiv:2412.20138)](https://arxiv.org/abs/2412.20138)
- [TradingAgents GitHub](https://github.com/TauricResearch/TradingAgents)
- [LangGraph Go 文档](https://github.com/smallnest/langgraphgo)
- [LangChain Go 文档](https://github.com/tmc/langchaingo)

## 📄 许可证

MIT 许可证 - 有关详细信息，请参阅 LICENSE 文件

---

**项目状态**：✅ 完成并可使用

**最后更新**：2024 年 12 月
