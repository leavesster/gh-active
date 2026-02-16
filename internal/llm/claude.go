package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/yleaf/gh-active/pkg/model"
)

type Claude struct {
	client anthropic.Client
	model  string
}

func NewClaude(apiKey, modelName string) *Claude {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	c := &Claude{
		client: anthropic.NewClient(opts...),
		model:  modelName,
	}
	if c.model == "" {
		c.model = string(anthropic.ModelClaude3_7SonnetLatest)
	}
	return c
}

func (c *Claude) Summarize(activities []model.Activity, opts SummarizeOpts) (string, error) {
	prompt := BuildPrompt(activities, opts)

	msg, err := c.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		MaxTokens: 2048,
		Model:     anthropic.Model(c.model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API: %w", err)
	}

	for _, block := range msg.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("claude: no text in response")
}
