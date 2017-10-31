package rpgORdie

import (
	"strings"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type RPGPlugin struct {
	Bot bot.Bot
	listenFor map[string]bool
}

func New(b bot.Bot) *RPGPlugin {
	return &RPGPlugin{
		Bot: b,
		listenFor: map[string]bool{},
	}
}

func (p *RPGPlugin) Message(message msg.Message) bool {
	if strings.ToLower(message.Body) == "start rpg" {
		ts := p.Bot.SendMessage(message.Channel, "I'll edit this.")

		p.listenFor[ts] = true

		time.Sleep(2 * time.Second)

		edited := ""
		for i := 0; i <= 5; i++ {
			p.Bot.Edit(message.Channel, edited, ts)
			edited += ":fire:"
			time.Sleep(500 * time.Millisecond)
		}
		p.Bot.Edit(message.Channel, "HECK YES", ts)

		p.Bot.ReplyToMessageIdentifier(message.Channel, "How's this reply?", ts)
		return true
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

func (p *RPGPlugin) ReplyMessage(message msg.Message, identifier string) bool {
	if strings.ToLower(message.User.Name) != strings.ToLower(p.Bot.Config().Nick) {
		if _, ok := p.listenFor[identifier]; ok {
			p.Bot.ReplyToMessageIdentifier(message.Channel, "Pong", identifier)
			return true
		}
	}
	return false
}
