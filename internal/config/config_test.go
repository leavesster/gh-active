package config

import "testing"

func TestLoadOpenAIKeyAlias(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_KEY", "alias-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.OpenAI.APIKey != "alias-key" {
		t.Fatalf("OpenAI API key = %q, want alias-key", cfg.LLM.OpenAI.APIKey)
	}
}

func TestLoadOpenAIAPIKeyPrecedence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", "")
	t.Setenv("OPENAI_API_KEY", "api-key")
	t.Setenv("OPENAI_KEY", "alias-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LLM.OpenAI.APIKey != "api-key" {
		t.Fatalf("OpenAI API key = %q, want api-key", cfg.LLM.OpenAI.APIKey)
	}
}
