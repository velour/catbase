package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var goatse []string = []string{
	"* g o a t s e x * g o a t s e x * g o a t s e x *",
	"g                                               g",
	"o /     \\             \\            /    \\       o",
	"a|       |             \\          |      |      a",
	"t|       `.             |         |       :     t",
	"s`        |             |        \\|       |     s",
	"e \\       | /       /  \\\\\\   --__ \\\\       :    e",
	"x  \\      \\/   _--~~          ~--__| \\     |    x",
	"*   \\      \\_-~                    ~-_\\    |    *",
	"g    \\_     \\        _.--------.______\\|   |    g",
	"o      \\     \\______// _ ___ _ \\_\\__>  \\   |    o",
	"a       \\   .  C ___)  ______ \\_\\____>  |  /    a",
	"t       /\\ |   C ____)/      \\ \\_____>  |_/     t",
	"s      / /\\|   C_____)       |  \\___>   /  \\    s",
	"e     |   \\   _C_____)\\______/  // _/ /     \\   e",
	"x     |    \\  |__   \\\\_________// \\__/       |  x",
	"*    | \\    \\____)   `----   --'             |  *",
	"g    |  \\_          ___\\       /_          _/ | g",
	"o   |              /    |     |  \\            | o",
	"a   |             |    /       \\  \\           | a",
	"t   |          / /    |{nick}|  \\           |t",
	"s   |         / /      \\__/\\___/    |          |s",
	"e  |           /        |    |       |         |e",
	"x  |          |         |    |       |         |x",
	"* g o a t s e x * g o a t s e x * g o a t s e x *",
}

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
	lowermessage := strings.ToLower(body)

	if channel != p.Bot.Config.MainChannel {
		return false
	}

	if strings.HasPrefix(lowermessage, "say") {
		msg := strings.TrimSpace(body[3:])
		p.Bot.SendMessage(channel, msg)
	}

	if strings.HasPrefix(lowermessage, "goatse") {
		nick := message.User.Name
		if parts := strings.Split(message.Body, " "); len(parts) > 1 {
			nick = parts[1]
		}

		for _, line := range goatse {
			nick = fmt.Sprintf("%9.9s", nick)
			line = strings.Replace(line, "{nick}", nick, 1)
			p.Bot.SendMessage(channel, line)
		}
		return true
	}

	if len(message.User.Name) != 9 {
		msg := fmt.Sprintf("Hey %s, we really like to have 9 character nicks because we're crazy OCD and stuff.",
			message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	}

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
	if kind == "JOIN" && strings.ToLower(message.User.Name) != strings.ToLower(p.Bot.Config.Nick) {
		sayings := p.Bot.Config.WelcomeMsgs
		msg := fmt.Sprintf(sayings[rand.Intn(len(sayings))], message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	}
	return false
}
