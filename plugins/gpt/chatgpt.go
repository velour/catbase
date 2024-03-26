package gpt

import (
	"context"
	"fmt"

	"github.com/andrewstuart/openai"
)

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
		if err := p.setPrompt(p.getDefaultPrompt()); err != nil {
			return "", err
		}
	}
	if p.chatCount > p.c.GetInt("gpt.maxchats", 10) {
		p.setPrompt(p.c.Get("gpt.lastprompt", p.getDefaultPrompt()))
		p.chatCount = 0
	}
	p.chatCount++
	return session.Complete(context.Background(), request)
}

func (p *GPTPlugin) getDefaultPrompt() string {
	return p.c.Get("gpt.prompt", "")
}

func (p *GPTPlugin) setPrompt(prompt string) error {
	var err error
	client, err = p.getClient()
	if err != nil {
		return err
	}
	session = client.NewChatSession(prompt)
	session.Model = p.c.Get("gpt.model", "gpt-3.5-turbo")
	err = p.c.Set("gpt.lastprompt", prompt)
	if err != nil {
		return err
	}
	return nil
}
