package tool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

type ShellTool struct {
}

func (t *ShellTool) Run(args map[string]any, code string) (string, error) {
	tmpl, err := template.New("shell").Parse(code)
	if err != nil {
		return "", fmt.Errorf("failed to parse shell template: %w", err)
	}

	var script bytes.Buffer
	err = tmpl.Execute(&script, args)
	if err != nil {
		return "", fmt.Errorf("failed to execute shell template: %w", err)
	}

	tmpfile, err := os.CreateTemp("", "shell-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(script.Bytes()); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return RunShellScript(tmpfile.Name(), nil)
}

// RunShellScript executes a shell script and returns its combined stdout and stderr.
func RunShellScript(scriptPath string, args []string) (string, error) {
	// nolint:gosec // G204: This is the intended functionality of the shell tool
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run shell script '%s': %w\nStdout: %s\nStderr: %s", scriptPath, err, stdout.String(), stderr.String())
	}

	return stdout.String() + stderr.String(), nil
}
