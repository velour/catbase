package talklikeapirate

import (
	"fmt"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
	"strings"
)

// TalkLikeAPiratePlugin allows admin of the filter
type TalkLikeAPiratePlugin struct {
	b        bot.Bot
	c        *config.Config
	handlers bot.HandlerTable
}

func New(b bot.Bot) *TalkLikeAPiratePlugin {
	p := &TalkLikeAPiratePlugin{
		b: b,
		c: b.Config(),
	}

	p.register()

	return p
}

func (p *TalkLikeAPiratePlugin) register() {
	p.handlers = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`^enable pirate$`),
			HelpText: "Enable message filter",
			Handler:  p.setEnabled(true),
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`^disable pirate$`),
			HelpText: "Disable message filter",
			Handler:  p.setEnabled(false),
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`^pirate-prompt:? (?P<text>.*)$`),
			HelpText: "Set message filter prompt",
			Handler:  p.setPrompt,
		},
		{
			Kind: bot.Help, IsCmd: false,
			Regex:   regexp.MustCompile(`.*`),
			Handler: p.help,
		},
	}
	p.b.RegisterTable(p, p.handlers)
}

func (p *TalkLikeAPiratePlugin) setEnabled(isEnabled bool) bot.ResponseHandler {
	return func(r bot.Request) bool {
		p.c.SetBool("talklikeapirate.enabled", isEnabled)
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I just set the message filter status to: %v", isEnabled))
		return true
	}
}

func (p *TalkLikeAPiratePlugin) setPrompt(r bot.Request) bool {
	prompt := r.Values["text"]
	p.c.Set("talklikeapirate.systemprompt", prompt)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("I set the message filter prompt to: %s", prompt))
	return true
}

func (p *TalkLikeAPiratePlugin) help(r bot.Request) bool {
	out := "Talk like a pirate commands:\n"
	for _, h := range p.handlers {
		if h.HelpText == "" {
			continue
		}
		out += fmt.Sprintf("```%s```\t%s", h.Regex.String(), h.HelpText)
	}
	out = strings.TrimSpace(out)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, out)
	return true
}
