package sms

import (
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type SMSPlugin struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *SMSPlugin {
	sp := &SMSPlugin{
		b: b,
		c: b.Config(),
	}
	b.Register(sp, bot.Message, sp.message)
	b.Register(sp, bot.Help, sp.help)
	return sp
}

func (p *SMSPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	return false
}

func (p *SMSPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	p.b.Send(c, bot.Message, ch, "There is no help for you.")
	return true
}
