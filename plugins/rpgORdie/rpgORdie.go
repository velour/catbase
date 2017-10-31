package rpgORdie

import (
	"strings"
	// "log"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/slack"
)

type RPGPlugin struct {
	Bot     bot.Bot
	Slack   *slack.Slack //nasty
}

func New(b bot.Bot) *RPGPlugin {
	return &RPGPlugin{
		Bot:     b,
		Slack:   b.Conn().(*slack.Slack), //oh boy, this is just filthy
	}
}

func (p *RPGPlugin) Message(message msg.Message) bool {
	if strings.ToLower(message.Body) == "start rpg" {
		p.Bot.SendMessage(message.Channel, "I'll edit this.")
		ts := p.Slack.GetLastMessageId()

		time.Sleep(2 * time.Second)

		edited := ""
		for i := 0; i <= 5; i++ {
			p.Bot.Edit(message.Channel, edited, ts)
			edited += ":fire:"
			time.Sleep(2 * time.Second)
		}
		p.Bot.Edit(message.Channel, "HECK YES", ts)
	}
	return false
}

func (p *RPGPlugin) LoadData() {

}

func (p *RPGPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Go find a walkthrough or something.")
}

func (p *RPGPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *RPGPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *RPGPlugin) RegisterWeb() *string {
	return nil
}
