package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"oss-aps-cli/internal/core"
)

type Tool struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	InstallCmd   []string `json:"install_cmd"`
	VerifyCmd    []string `json:"verify_cmd"`
	PlatformTags []string `json:"platform_tags"`
	Version      string   `json:"version,omitempty"`
}

var ToolRegistry = map[string]Tool{
	"claude": {
		Name:         "Claude Code",
		Description:  "Anthropic's AI coding assistant",
		InstallCmd:   []string{"npm", "install", "-g", "@anthropic-ai/claude-code@latest"},
		VerifyCmd:    []string{"claude", "--version"},
		PlatformTags: []string{"all"},
	},
	"gemini": {
		Name:         "Google Gemini CLI",
		Description:  "Google's AI coding assistant",
		InstallCmd:   []string{"npm", "install", "-g", "@google/gemini-cli"},
		VerifyCmd:    []string{"gemini", "--version"},
		PlatformTags: []string{"all"},
	},
	"codex": {
		Name:         "OpenAI Codex",
		Description:  "OpenAI's AI coding assistant",
		InstallCmd:   []string{"npm", "install", "-g", "@openai/codex-cli"},
		VerifyCmd:    []string{"codex", "--version"},
		PlatformTags: []string{"all"},
	},
	"python3": {
		Name:         "Python 3",
		Description:  "Python 3 interpreter",
		InstallCmd:   []string{"", "", ""},
		VerifyCmd:    []string{"python3", "--version"},
		PlatformTags: []string{"all"},
	},
	"node": {
		Name:         "Node.js",
		Description:  "Node.js JavaScript runtime",
		InstallCmd:   []string{"", "", ""},
		VerifyCmd:    []string{"node", "--version"},
		PlatformTags: []string{"all"},
	},
	"git": {
		Name:         "Git",
		Description:  "Git version control",
		InstallCmd:   []string{"", "", ""},
		VerifyCmd:    []string{"git", "--version"},
		PlatformTags: []string{"all"},
	},
}

func GetTool(name string) (Tool, error) {
	tool, ok := ToolRegistry[name]
	if !ok {
		return Tool{}, fmt.Errorf("tool '%s' not found in registry", name)
	}
	return tool, nil
}

func ListTools() []Tool {
	var tools []Tool
	for _, tool := range ToolRegistry {
		if isToolAvailable(tool) {
			tools = append(tools, tool)
		}
	}
	return tools
}

func isToolAvailable(tool Tool) bool {
	for _, tag := range tool.PlatformTags {
		if tag == "all" || tag == runtime.GOOS {
			return true
		}
	}
	return false
}

func IsToolInstalled(tool Tool) bool {
	if len(tool.VerifyCmd) == 0 || tool.VerifyCmd[0] == "" {
		return false
	}

	if _, err := exec.LookPath(tool.VerifyCmd[0]); err != nil {
		return false
	}

	cmd := exec.Command(tool.VerifyCmd[0], tool.VerifyCmd[1:]...)
	err := cmd.Run()
	return err == nil
}

func EnsureTool(name string, version string) error {
	tool, err := GetTool(name)
	if err != nil {
		return err
	}

	if version != "" {
		tool.Version = version
		tool.InstallCmd = updateInstallVersion(tool.InstallCmd, version)
	}

	if IsToolInstalled(tool) {
		if version != "" {
			installedVersion, _ := getToolVersion(tool)
			if installedVersion == version {
				return nil
			}
		} else {
			return nil
		}
	}

	fmt.Printf("Installing tool '%s' (%s)\n", tool.Name, version)
	return InstallTool(tool)
}

func updateInstallVersion(cmd []string, version string) []string {
	if len(cmd) == 0 {
		return cmd
	}

	for i, arg := range cmd {
		if strings.Contains(arg, "@latest") {
			cmd[i] = strings.Replace(arg, "@latest", "@"+version, -1)
		}
	}

	return cmd
}

func getToolVersion(tool Tool) (string, error) {
	if len(tool.VerifyCmd) == 0 {
		return "", fmt.Errorf("no verify command for tool")
	}

	cmd := exec.Command(tool.VerifyCmd[0], tool.VerifyCmd[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(output))

	parts := strings.Split(version, " ")
	if len(parts) > 1 {
		version = parts[len(parts)-1]
	}

	version = strings.TrimPrefix(version, "v")
	version = strings.TrimSuffix(version, "\n")

	return version, nil
}

func InstallTool(tool Tool) error {
	if len(tool.InstallCmd) == 0 || tool.InstallCmd[0] == "" {
		return fmt.Errorf("tool '%s' cannot be automatically installed", tool.Name)
	}

	cmd := exec.Command(tool.InstallCmd[0], tool.InstallCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tool '%s': %w", tool.Name, err)
	}

	fmt.Printf("Tool '%s' installed successfully\n", tool.Name)

	if !IsToolInstalled(tool) {
		return fmt.Errorf("tool '%s' installation verification failed", tool.Name)
	}

	return nil
}

func DiscoverProfileScripts(profileID string) ([]Tool, error) {
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return nil, err
	}

	toolsDir := filepath.Join(profileDir, "tools")
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		return []Tool{}, nil
	}

	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}

	var tools []Tool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)

		switch ext {
		case ".sh":
			tools = append(tools, Tool{
				Name:         strings.TrimSuffix(name, ext),
				Description:  fmt.Sprintf("Profile script: %s", name),
				InstallCmd:   []string{},
				VerifyCmd:    []string{"test", "-x", filepath.Join(toolsDir, name)},
				PlatformTags: []string{"all"},
			})
		case ".py":
			tools = append(tools, Tool{
				Name:         strings.TrimSuffix(name, ext),
				Description:  fmt.Sprintf("Profile script: %s", name),
				InstallCmd:   []string{},
				VerifyCmd:    []string{"test", "-f", filepath.Join(toolsDir, name)},
				PlatformTags: []string{"all"},
			})
		case ".js":
			tools = append(tools, Tool{
				Name:         strings.TrimSuffix(name, ext),
				Description:  fmt.Sprintf("Profile script: %s", name),
				InstallCmd:   []string{},
				VerifyCmd:    []string{"test", "-f", filepath.Join(toolsDir, name)},
				PlatformTags: []string{"all"},
			})
		}
	}

	return tools, nil
}

func ExecuteProfileTool(profileID string, toolName string, args []string) error {
	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return err
	}

	toolsDir := filepath.Join(profileDir, "tools")
	toolPath := filepath.Join(toolsDir, toolName)

	if _, err := os.Stat(toolPath); err != nil {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	ext := filepath.Ext(toolPath)

	var cmd *exec.Cmd
	switch ext {
	case ".sh":
		cmd = exec.Command("/bin/sh", toolPath)
	case ".py":
		cmd = exec.Command("python3", toolPath)
	case ".js":
		cmd = exec.Command("node", toolPath)
	default:
		cmd = exec.Command(toolPath)
	}

	cmd.Args = append(cmd.Args, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	return nil
}
