package llm

import (
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
)

const gpt3URL = "https://api.openai.com/v1/engines/%s/completions"
const gpt3ModURL = "https://api.openai.com/v1/moderations"

type LLMPlugin struct {
	b bot.Bot
	c *config.Config
	h bot.HandlerTable

	chatCount   int
	chatHistory []chatEntry

	geminiClient *genai.Client
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
			HelpText: "chat completion using first-available AI",
			Handler:  p.geminiChatMessage,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^gpt4 (?P<text>.*)`),
			HelpText: "chat completion using OpenAI",
			Handler:  p.gptMessage,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?is)^llm-puke$`),
			HelpText: "clear chat history",
			Handler:  p.puke,
		},
	}
	p.b.RegisterTable(p, p.h)
}

func (p *LLMPlugin) setPromptMessage(r bot.Request) bool {
	p.c.Set("gemini.systemprompt", r.Values["text"])
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf(`Okay. I set the prompt to: "%s"`, r.Values["text"]))
	return true
}

func (p *LLMPlugin) geminiChatMessage(r bot.Request) bool {
	if p.geminiClient == nil && p.geminiConnect() != nil {
		log.Error().Msgf("Could not connect to Gemini")
		return p.gptMessage(r)
	}
	chatResp, err := p.gemini(r.Values["text"])
	if err != nil {
		log.Error().Err(err).Send()
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Problem with Gemini: %s", err))
		return true
	}
	p.chatHistory = append(p.chatHistory, chatEntry{"User", r.Values["text"]})
	p.chatHistory = append(p.chatHistory, chatResp)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, chatResp.Content)
	log.Info().Msgf("Successfully used Gemini")
	return true
}

func (p *LLMPlugin) ollamaChatMessage(r bot.Request) bool {
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
		log.Error().Msgf("llama is currently down")
	}
	return p.gptMessage(r)
}

func (p *LLMPlugin) gptMessage(r bot.Request) bool {
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

func (p *LLMPlugin) puke(r bot.Request) bool {
	resp := fmt.Sprintf("I just forgot %d lines of chat history.", len(p.chatHistory))
	p.chatHistory = []chatEntry{}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, resp)
	return true
}
