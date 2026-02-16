package config

import (
	"os"
	"path/filepath"

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
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type ReportConfig struct {
	Language string `yaml:"language"`
}

func Load() (*Config, error) {
	cfg := &Config{
		LLM: LLMConfig{
			Default: "claude",
			Claude:  ClaudeConfig{Model: "claude-sonnet-4-5-20250929"},
			OpenAI:  OpenAIConfig{Model: "gpt-4o"},
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

	return cfg, nil
}

func DefaultYAML() string {
	return `github:
  token: ""           # or set GITHUB_TOKEN env var

llm:
  default: claude
  claude:
    api_key: ""       # or set ANTHROPIC_API_KEY env var
    model: claude-sonnet-4-5-20250929
  openai:
    api_key: ""       # or set OPENAI_API_KEY env var
    model: gpt-4o

report:
  language: zh-CN
`
}
