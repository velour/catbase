package mayi

import (
	"math/rand"
	"regexp"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type MayIPlugin struct {
	b bot.Bot
	c *config.Config
}

var regex = regexp.MustCompile(`(?i)^(may|can) (?P<who>\S+) (?P<what>.+)`)

func New(b bot.Bot) *MayIPlugin {
	m := &MayIPlugin{
		b: b,
		c: b.Config(),
	}

	b.RegisterRegexCmd(m, bot.Message, regex, m.message)

	return m
}

func (p *MayIPlugin) message(r bot.Request) bool {
	msg := p.c.Get("mayi.no", "no")
	if rand.Intn(2) == 0 {
		msg = p.c.Get("mayi.yes", "yes")
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}
