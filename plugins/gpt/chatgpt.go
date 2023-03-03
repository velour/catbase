package gpt

import (
	"context"
	"fmt"
)
import "github.com/andrewstuart/openai"

var session *openai.ChatSession
var client *openai.Client

func (p *GPTPlugin) getClient() (*openai.Client, error) {
	token := p.c.Get("gpt.token", "")
	if token == "" {
		return nil, fmt.Errorf("no GPT token given")
	}
	if client == nil {
		return openai.NewClient(token)
	}
	return client, nil
}

func (p *GPTPlugin) chatGPT(request string) (string, error) {
	if session == nil {
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
	client, err := p.getClient()
	if err != nil {
		return err
	}
	sess := client.NewChatSession(prompt)
	session = &sess
	return nil
}
