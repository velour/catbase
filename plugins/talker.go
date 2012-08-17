package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"strings"
)

type TalkerPlugin struct {
	Bot *bot.Bot
}

func NewTalkerPlugin(bot *bot.Bot) *TalkerPlugin {
	return &TalkerPlugin{
		Bot: bot,
	}
}

func (p *TalkerPlugin) Message(message bot.Message) bool {
	channel := message.Channel
	body := message.Body

	if channel != p.Bot.Config.MainChannel {
		return false
	}

	lowermessage := strings.ToLower(body)

	if strings.Contains(lowermessage, "felps") || strings.Contains(lowermessage, "fredfelps") {
		outmsg := p.Bot.Filter(message, "GOD HATES $NICK")
		p.Bot.SendMessage(channel, outmsg)
		return true
	}

	return false
}

func (p *TalkerPlugin) LoadData() {
	// no data to load yet?
}

func (p *TalkerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Hi, this is talker. I like to talk about FredFelps!")
}
