package deadend

import (
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
)

const defaultMessage = "I don't know how to respond to that. If you'd like to ask GPT, use the `gpt` command."

type DeadEndPlugin struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *DeadEndPlugin {
	p := &DeadEndPlugin{
		b: b,
		c: b.Config(),
	}
	b.RegisterRegexCmd(p, bot.Message, regexp.MustCompile(`.*`), p.message)
	return p
}

func (p *DeadEndPlugin) message(r bot.Request) bool {
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel,
		p.c.Get("deadend.message", defaultMessage))
	return true
}
