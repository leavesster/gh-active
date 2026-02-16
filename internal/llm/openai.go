package llm

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"github.com/yleaf/gh-active/pkg/model"
)

type OpenAI struct {
	client *openai.Client
	model  string
}

func NewOpenAI(apiKey, modelName string) *OpenAI {
	o := &OpenAI{
		client: openai.NewClient(apiKey),
		model:  modelName,
	}
	if o.model == "" {
		o.model = openai.GPT4o
	}
	return o
}

func (o *OpenAI) Summarize(activities []model.Activity, opts SummarizeOpts) (string, error) {
	prompt := BuildPrompt(activities, opts)

	resp, err := o.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: o.model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("openai API: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}
