package goskills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallnest/goskills"
	"github.com/smallnest/goskills/tool"
	"github.com/tmc/langchaingo/tools"
)

// SkillTool implements tools.Tool for goskills.
type SkillTool struct {
	name        string
	description string
	scriptMap   map[string]string
	skillPath   string
}

var _ tools.Tool = &SkillTool{}

func (t *SkillTool) Name() string {
	return t.name
}

func (t *SkillTool) Description() string {
	return t.description
}

func (t *SkillTool) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":        t.name,
		"description": t.description,
		"skillPath":   t.skillPath,
		"scriptMap":   t.scriptMap,
	})
}

func (t *SkillTool) Call(ctx context.Context, input string) (string, error) {
	// input is the JSON string of arguments
	// We need to parse it based on the tool name, similar to goskills runner.go

	switch t.name {
	case "run_shell_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_code arguments: %w", err)
		}
		shellTool := tool.ShellTool{}
		return shellTool.Run(params.Args, params.Code)

	case "run_shell_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_script arguments: %w", err)
		}
		return tool.RunShellScript(params.ScriptPath, params.Args)

	case "run_python_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_code arguments: %w", err)
		}
		pythonTool := tool.PythonTool{}
		return pythonTool.Run(params.Args, params.Code)

	case "run_python_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_script arguments: %w", err)
		}
		return tool.RunPythonScript(params.ScriptPath, params.Args)

	case "read_file":
		var params struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal read_file arguments: %w", err)
		}
		path := params.FilePath
		if !filepath.IsAbs(path) && t.skillPath != "" {
			resolvedPath := filepath.Join(t.skillPath, path)
			if _, err := os.Stat(resolvedPath); err == nil {
				path = resolvedPath
			}
		}
		return tool.ReadFile(path)

	case "write_file":
		var params struct {
			FilePath string `json:"filePath"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal write_file arguments: %w", err)
		}
		err := tool.WriteFile(params.FilePath, params.Content)
		if err == nil {
			return fmt.Sprintf("Successfully wrote to file: %s", params.FilePath), nil
		}
		return "", err

	case "wikipedia_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal wikipedia_search arguments: %w", err)
		}
		return tool.WikipediaSearch(params.Query)

	case "tavily_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal tavily_search arguments: %w", err)
		}
		return tool.TavilySearch(params.Query)

	case "web_fetch":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal web_fetch arguments: %w", err)
		}
		return tool.WebFetch(params.URL)

	default:
		if scriptPath, ok := t.scriptMap[t.name]; ok {
			var params struct {
				Args []string `json:"args"`
			}
			if input != "" {
				if err := json.Unmarshal([]byte(input), &params); err != nil {
					return "", fmt.Errorf("failed to unmarshal script arguments: %w", err)
				}
			}
			if strings.HasSuffix(scriptPath, ".py") {
				return tool.RunPythonScript(scriptPath, params.Args)
			} else {
				return tool.RunShellScript(scriptPath, params.Args)
			}
		}
		return "", fmt.Errorf("unknown tool: %s", t.name)
	}
}

// SkillsToTools converts a goskills.SkillPackage to a slice of tools.Tool.
func SkillsToTools(skill *goskills.SkillPackage) ([]tools.Tool, error) {
	availableTools, scriptMap := goskills.GenerateToolDefinitions(skill)
	var result []tools.Tool

	for _, t := range availableTools {
		if t.Function.Name == "" {
			continue
		}

		// Create a description that includes the arguments schema if possible,
		// but langchaingo tools usually just have a text description.
		// We can append the JSON schema of parameters to the description to help the LLM.
		desc := t.Function.Description
		// Note: Parameters schema is available via t.Function.Parameters if needed,
		// but langchaingo's tools.Tool interface doesn't have a Schema method.
		// The schema would need to be handled separately if function calling support is required.
		_ = t.Function.Parameters // Acknowledge parameters exist but aren't used here

		result = append(result, &SkillTool{
			name:        t.Function.Name,
			description: desc,
			scriptMap:   scriptMap,
			skillPath:   skill.Path,
		})
	}

	return result, nil
}

// MCPToTools converts MCP tools to langchaingo tools.
// Note: goskills also supports MCP. We can add a helper for that too if needed,
// but the user specifically asked for "Skills封装".
