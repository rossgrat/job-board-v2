package llm

import (
	"github.com/anthropics/anthropic-sdk-go"
	"golang.org/x/time/rate"
)

type Option func(*Client)

func WithModel(model anthropic.Model) Option {
	return func(c *Client) {
		c.model = model
	}
}

func WithRateLimiter(limiter *rate.Limiter) Option {
	return func(c *Client) {
		c.limiter = limiter
	}
}

func WithSchema(schema map[string]any) Option {
	return func(c *Client) {
		c.schema = schema
	}
}
