package gpt

import (
	"context"
	"fmt"
)
import "github.com/andrewstuart/openai"

var session openai.ChatSession
var client *openai.Client

func (p *GPTPlugin) getClient() (*openai.Client, error) {
	token := p.c.Get("gpt.token", "")
	if token == "" {
		return nil, fmt.Errorf("no GPT token given")
	}
	return openai.NewClient(token)
}

func (p *GPTPlugin) chatGPT(request string) (string, error) {
	if client == nil {
		if err := p.setDefaultPrompt(); err != nil {
			return "", err
		}
	}
	return session.Complete(context.Background(), request)
}

func (p *GPTPlugin) setDefaultPrompt() error {
	return p.setPrompt(p.c.Get("gpt.prompt", ""))
}

func (p *GPTPlugin) setPrompt(prompt string) error {
	var err error
	client, err = p.getClient()
	if err != nil {
		return err
	}
	session = client.NewChatSession(prompt)
	return nil
}
