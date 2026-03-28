package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"golang.org/x/time/rate"
)

var (
	ErrRateLimit       = errors.New("rate limit exceeded")
	ErrAPICall         = errors.New("llm api call failed")
	ErrNoContent       = errors.New("llm returned no content")
	ErrUnmarshal       = errors.New("failed to unmarshal llm response")
	ErrReadPromptFile  = errors.New("failed to read prompt file")
	ErrEmptyPromptFile = errors.New("prompt file is empty")
)

type Client struct {
	client       anthropic.Client
	limiter      *rate.Limiter
	model        anthropic.Model
	systemPrompt string
	schema       map[string]any
}

func New(apiKey string, opts ...Option) (*Client, error) {
	const fn = "llm::New"

	c := &Client{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.ModelClaudeHaiku4_5,
	}

	for _, o := range opts {
		o(c)
	}

	return c, nil
}

func (c *Client) LoadPrompt(path string) error {
	const fn = "llm::Client::LoadPrompt"

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrReadPromptFile, err)
	}

	prompt := string(data)
	if prompt == "" {
		return fmt.Errorf("%s:%w", fn, ErrEmptyPromptFile)
	}

	c.systemPrompt = prompt
	return nil
}

func Complete[T any](ctx context.Context, c *Client, userMessage string) (*T, error) {
	const fn = "llm::Complete"

	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("%s:%w:%w", fn, ErrRateLimit, err)
		}
	}

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{{
			Text: c.systemPrompt,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: c.schema,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrAPICall, err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("%s:%w", fn, ErrNoContent)
	}

	var text string
	for _, block := range resp.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			text = variant.Text
		}
	}

	var result T
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUnmarshal, err)
	}

	return &result, nil
}
