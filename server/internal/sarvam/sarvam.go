// Package sarvam wraps the two Sarvam calls the bot needs:
//   - Prism (a decision model) to pick which existing task a message belongs to
//   - a small chat model to write a vague title + description for a new task
package sarvam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Option struct {
	ID   string
	Text string
}

type Naming struct {
	Title       string
	Description string
}

type Client struct {
	Key         string
	BaseURL     string // https://api.sarvam.ai
	NamingModel string // smallest chat model; no reasoning
	HTTP        *http.Client
}

func New(key, baseURL, namingModel string) *Client {
	return &Client{Key: key, BaseURL: strings.TrimRight(baseURL, "/"), NamingModel: namingModel, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	body, _ := json.Marshal(in)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-subscription-key", c.Key)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sarvam %s: %d %s", path, resp.StatusCode, truncate(string(raw), 200))
	}
	return json.Unmarshal(raw, out)
}

// Choose asks Prism which option fits `state` best and returns a probability per option ID
// (they sum to 1).
func (c *Client) Choose(ctx context.Context, state, instructions string, options []Option) (map[string]float64, error) {
	criteria := make(map[string]string, len(options))
	for _, o := range options {
		criteria[o.ID] = o.Text
	}
	req := map[string]any{
		"model": "sarvam-prism",
		"state": state,
		"questions": map[string]any{
			"pick": map[string]any{"type": "choice", "instructions": instructions, "criteria": criteria},
		},
	}
	var resp struct {
		Answers map[string]struct {
			Probabilities map[string]float64 `json:"probabilities"`
		} `json:"answers"`
	}
	if err := c.post(ctx, "/v2/systemone", req, &resp); err != nil {
		return nil, err
	}
	p := resp.Answers["pick"].Probabilities
	if len(p) == 0 {
		return nil, fmt.Errorf("sarvam prism: empty answer")
	}
	return p, nil
}

// Name writes a short, vague title and description for a message that starts a new task.
func (c *Client) Name(ctx context.Context, text string) (Naming, error) {
	req := map[string]any{
		"model": c.NamingModel,
		"messages": []map[string]string{
			{"role": "system", "content": `You name maintenance/request tickets from chat messages (English, Hindi or Hinglish). ` +
				`Reply with ONLY JSON: {"title": "...", "description": "..."}. ` +
				`title: at most 6 words, vague and general (e.g. "Water leak", "Locker issue"), no names or room numbers. ` +
				`description: one short sentence, in English, summarising the problem in general terms.`},
			{"role": "user", "content": text},
		},
		"temperature": 0.2,
		"max_tokens":  200,
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.post(ctx, "/v1/chat/completions", req, &resp); err != nil {
		return Naming{}, err
	}
	if len(resp.Choices) == 0 {
		return Naming{}, fmt.Errorf("sarvam chat: no choices")
	}
	return ParseNaming(resp.Choices[0].Message.Content)
}

// ParseNaming pulls {"title","description"} out of a model reply, tolerating code fences and chatter.
func ParseNaming(content string) (Naming, error) {
	s := content
	if i, j := strings.Index(s, "{"), strings.LastIndex(s, "}"); i >= 0 && j > i {
		s = s[i : j+1]
	}
	var n struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(s), &n); err != nil || strings.TrimSpace(n.Title) == "" {
		return Naming{}, fmt.Errorf("sarvam chat: unusable reply %q", truncate(content, 100))
	}
	return Naming{Title: strings.TrimSpace(n.Title), Description: strings.TrimSpace(n.Description)}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
