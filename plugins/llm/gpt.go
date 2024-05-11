package llm

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
	"slices"
)

const gpt3URL = "https://api.openai.com/v1/engines/%s/completions"
const gpt3ModURL = "https://api.openai.com/v1/moderations"

type LLMPlugin struct {
	b bot.Bot
	c *config.Config
	h bot.HandlerTable

	chatCount   int
	chatHistory []chatEntry
}

type chatEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func New(b bot.Bot) *LLMPlugin {
	p := &LLMPlugin{
		b: b,
		c: b.Config(),
	}
	p.register()
	return p
}

func (p *LLMPlugin) register() {
	p.h = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^gpt-prompt: (?P<text>.*)`),
			HelpText: "set the ChatGPT prompt",
			Handler:  p.setPromptMessage,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^llm (?P<text>.*)`),
			HelpText: "chat completion",
			Handler:  p.chatMessageForce,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^gpt (?P<text>.*)`),
			HelpText: "chat completion",
			Handler:  p.chatMessageForce,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^got (?P<text>.*)`),
			HelpText: "chat completion",
			Handler:  p.chatMessageForce,
		},
	}
	p.b.RegisterTable(p, p.h)
}

func (p *LLMPlugin) setPromptMessage(r bot.Request) bool {
	prompt := r.Values["text"]
	if err := p.setPrompt(prompt); err != nil {
		resp := fmt.Sprintf("Error: %s", err)
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, resp)
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf(`Okay. I set the prompt to: "%s"`, prompt))
	return true
}

func (p *LLMPlugin) chatMessage(r bot.Request) bool {
	if slices.Contains(p.c.GetArray("gpt.silence", []string{}), r.Msg.Channel) {
		log.Debug().Msgf("%s silenced", r.Msg.Channel)
		return true
	}
	return p.chatMessageForce(r)
}

func (p *LLMPlugin) chatMessageForce(r bot.Request) bool {
	p.chatHistory = append(p.chatHistory, chatEntry{
		Role:    "user",
		Content: r.Values["text"],
	})
	maxHist := p.c.GetInt("gpt.maxhist", 10)
	if len(p.chatHistory) > maxHist {
		p.chatHistory = p.chatHistory[len(p.chatHistory)-maxHist:]
	}
	chatResp, err := p.llama()
	if err == nil {
		p.chatHistory = append(p.chatHistory, chatResp)
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, chatResp.Content)
		return true
	} else if !errors.Is(err, InstanceNotFoundError) {
		log.Error().Err(err).Msgf("error contacting llama")
	} else {
		log.Info().Msgf("Llama is currently down")
	}
	resp, err := p.chatGPT(r.Values["text"])
	if err != nil {
		resp = fmt.Sprintf("Error: %s", err)
	}
	p.chatHistory = append(p.chatHistory, chatEntry{
		Role:    "assistant",
		Content: resp,
	})
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, resp)
	return true
}
