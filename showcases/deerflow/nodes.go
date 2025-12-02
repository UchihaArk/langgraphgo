package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type logKey struct{}

func logf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// Always print to stdout
	fmt.Print(msg)

	// If log channel exists in context, send it there too
	if ch, ok := ctx.Value(logKey{}).(chan string); ok {
		// Non-blocking send to avoid stalling if channel is full or no one listening
		select {
		case ch <- msg:
		default:
		}
	}
}

// PlannerNode generates a research plan based on the query.
func PlannerNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)
	logf(ctx, "--- 规划节点：正在为查询 '%s' 进行规划 ---\n", s.Request.Query)

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf("你是一名研究规划师。请为以下查询创建一个分步研究计划：%s。仅返回编号列表形式的计划。必须使用中文回复。", s.Request.Query)
	completion, err := llm.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Simple parsing of the plan (splitting by newlines)
	lines := strings.Split(completion, "\n")
	var plan []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			plan = append(plan, trimmed)
		}
	}
	s.Plan = plan
	s.Plan = plan

	// Format plan for better readability
	formattedPlan := "生成的计划：\n"
	for _, step := range s.Plan {
		formattedPlan += fmt.Sprintf("%s\n", step)
	}
	logf(ctx, formattedPlan)

	return s, nil
}

// ResearcherNode executes the research plan.
func ResearcherNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)
	logf(ctx, "--- 研究节点：正在执行计划 ---\n")

	// In a real implementation, we would use a search tool here.
	// For this showcase, we will simulate research or use the LLM to "hallucinate"/generate info if no tool is available.
	// Or better, let's use the LLM to simulate finding information for each step.

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	var results []string
	for _, step := range s.Plan {
		logf(ctx, "正在研究步骤：%s\n", step)
		prompt := fmt.Sprintf("你是一名研究员。请为这个研究步骤查找详细信息：%s。提供发现摘要。必须使用中文回复。", step)
		completion, err := llm.Call(ctx, prompt)
		if err != nil {
			return nil, err
		}
		results = append(results, fmt.Sprintf("Step: %s\nFindings: %s", step, completion))
	}

	s.ResearchResults = results
	return s, nil
}

// ReporterNode compiles the final report.
func ReporterNode(ctx context.Context, state interface{}) (interface{}, error) {
	s := state.(*State)
	logf(ctx, "--- 报告节点：正在生成最终报告 ---\n")

	llm, err := getLLM()
	if err != nil {
		return nil, err
	}

	researchData := strings.Join(s.ResearchResults, "\n\n")
	prompt := fmt.Sprintf("你是一名资深报告撰写员。请根据以下研究结果撰写一份全面的最终报告。使用 Markdown 格式，包含清晰的标题、要点，并在适当的地方使用代码块。数学公式请使用 ```math 代码块包裹，或者使用 $$...$$ (块级) 和 $...$ (行内) 包裹。如果内容适合用流程图、时序图等图表展示，请使用 ```mermaid 代码块生成 Mermaid 图表。必须使用中文撰写报告：\n\n%s\n\n原始查询是：%s", researchData, s.Request.Query)

	completion, err := llm.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Convert Markdown to HTML
	// Clean up markdown code blocks if present
	completion = strings.TrimPrefix(completion, "```markdown")
	completion = strings.TrimPrefix(completion, "```")
	completion = strings.TrimSuffix(completion, "```")

	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(completion))

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	s.FinalReport = string(markdown.Render(doc, renderer))
	logf(ctx, "最终报告已生成。\n")
	return s, nil
}

func getLLM() (llms.Model, error) {
	// Use DeepSeek as per user preference
	// Ensure OPENAI_API_KEY and OPENAI_API_BASE are set in the environment
	return openai.New()
}
