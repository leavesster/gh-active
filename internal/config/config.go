package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	LLM    LLMConfig    `yaml:"llm"`
	Report ReportConfig `yaml:"report"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type LLMConfig struct {
	Default string       `yaml:"default"`
	Claude  ClaudeConfig `yaml:"claude"`
	OpenAI  OpenAIConfig `yaml:"openai"`
}

type ClaudeConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
	Mode    string `yaml:"mode"`
}

type ReportConfig struct {
	Language string `yaml:"language"`
	Output   string `yaml:"output"`
}

func Load() (*Config, error) {
	cfg := &Config{
		LLM: LLMConfig{
			Default: "claude",
			Claude:  ClaudeConfig{Model: "claude-sonnet-4-5-20250929"},
			OpenAI:  OpenAIConfig{Model: "gpt-4o", Mode: "responses"},
		},
		Report: ReportConfig{Language: "zh-CN"},
	}

	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, ".gh-active.yaml")
		if data, err := os.ReadFile(path); err == nil {
			_ = yaml.Unmarshal(data, cfg)
		}
	}

	// env vars override config file
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		cfg.GitHub.Token = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.LLM.Claude.APIKey = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.LLM.OpenAI.APIKey = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.LLM.OpenAI.Model = v
	}
	if v := os.Getenv("OPENAI_API_MODE"); v != "" {
		cfg.LLM.OpenAI.Mode = v
	}

	// fallback: use gh CLI auth token if no GitHub token configured
	if cfg.GitHub.Token == "" {
		if token := ghAuthToken(); token != "" {
			cfg.GitHub.Token = token
		}
	}

	return cfg, nil
}

// ghAuthToken tries to get the GitHub token from the gh CLI.
func ghAuthToken() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func DefaultYAML() string {
	return `github:
  token: ""           # or set GITHUB_TOKEN env var

llm:
  default: claude
  claude:
    api_key: ""       # or set ANTHROPIC_API_KEY env var
    model: claude-sonnet-4-5-20250929
    base_url: ""      # custom API endpoint (e.g. proxy)
  openai:
    api_key: ""       # or set OPENAI_API_KEY env var
    model: gpt-4o     # or set OPENAI_MODEL env var
    base_url: ""      # custom API endpoint (e.g. proxy)
    mode: responses   # responses or chat; can also be OPENAI_API_MODE

report:
  language: zh-CN
  output: ""          # default stdout; can also be overridden by --output
`
}
