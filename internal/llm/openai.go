package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/leavesster/gh-active/pkg/model"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

const (
	openAIModeResponses = "responses"
	openAIModeChat      = "chat"
)

type OpenAI struct {
	apiKey     string
	model      string
	baseURL    string
	mode       string
	httpClient *http.Client
}

func NewOpenAI(apiKey, modelName, baseURL, mode string) *OpenAI {
	o := &OpenAI{
		apiKey:  apiKey,
		model:   modelName,
		baseURL: strings.TrimRight(baseURL, "/"),
		mode:    strings.ToLower(strings.TrimSpace(mode)),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	if o.model == "" {
		o.model = "gpt-4o"
	}
	if o.baseURL == "" {
		o.baseURL = defaultOpenAIBaseURL
	}
	if o.mode == "" {
		o.mode = openAIModeResponses
	}
	return o
}

func (o *OpenAI) Summarize(activities []model.Activity, opts SummarizeOpts) (string, error) {
	prompt := BuildPrompt(activities, opts)

	switch o.mode {
	case openAIModeResponses:
		return o.summarizeResponses(prompt)
	case openAIModeChat:
		return o.summarizeChat(prompt)
	default:
		return "", fmt.Errorf("unknown openai API mode %q (supported: responses, chat)", o.mode)
	}
}

func (o *OpenAI) summarizeResponses(prompt string) (string, error) {
	body, err := json.Marshal(openAIResponsesRequest{
		Model:           o.model,
		Input:           prompt,
		MaxOutputTokens: 2048,
	})
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	var out openAIResponsesResponse
	if err := o.postJSON("/responses", body, &out, "openai responses API"); err != nil {
		return "", err
	}
	if text := out.Text(); text != "" {
		return text, nil
	}
	return "", fmt.Errorf("openai: no text in response")
}

func (o *OpenAI) summarizeChat(prompt string) (string, error) {
	body, err := json.Marshal(openAIChatRequest{
		Model: o.model,
		Messages: []openAIChatMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	var out openAIChatResponse
	if err := o.postJSON("/chat/completions", body, &out, "openai chat completions API"); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices in response")
	}
	if strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("openai: no text in response")
	}
	return out.Choices[0].Message.Content, nil
}

func (o *OpenAI) postJSON(path string, body []byte, out any, apiName string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, o.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", apiName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s: status %d: %s", apiName, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode openai response: %w", err)
	}
	return nil
}

type openAIResponsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
}

type openAIResponsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func (r openAIResponsesResponse) Text() string {
	if strings.TrimSpace(r.OutputText) != "" {
		return r.OutputText
	}

	var parts []string
	for _, item := range r.Output {
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) == "" {
				continue
			}
			parts = append(parts, content.Text)
		}
	}
	return strings.Join(parts, "\n")
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}
