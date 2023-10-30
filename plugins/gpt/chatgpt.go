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
		if err := p.setPrompt(p.getDefaultPrompt()); err != nil {
			return "", err
		}
	}
	if p.chatCount > p.c.GetInt("gpt.maxchats", 10) {
		p.setPrompt(p.c.Get("gpt3.lastprompt", p.getDefaultPrompt()))
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
	err = p.c.Set("gpt3.lastprompt", prompt)
	if err != nil {
		return err
	}
	return nil
}
