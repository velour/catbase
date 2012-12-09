package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"fmt"
	"math/rand"
	"strings"
	"time"
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

	if len(message.User.Name) != 9 {
		msg := fmt.Sprintf("Hey %s, we really like to have 9 character nicks because we're crazy OCD and stuff.",
			message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
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
	rand.Seed(time.Now().Unix())
}

func (p *TalkerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Hi, this is talker. I like to talk about FredFelps!")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *TalkerPlugin) Event(kind string, message bot.Message) bool {
	if kind == "JOIN" && message.User.Name != p.Bot.Config.Nick {
		sayings := []string{
			"Real men use screen, %s.",
			"Joins upset the hivemind's OCD, %s.",
			"Joins upset the hivemind's CDO, %s.",
		}
		msg := fmt.Sprintf(sayings[rand.Intn(len(sayings))], message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	}
	return false
}
