// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package talker

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
)

var goatse []string = []string{
	"```* g o a t s e x * g o a t s e x * g o a t s e x *",
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
	"* g o a t s e x * g o a t s e x * g o a t s e x *```",
}

type TalkerPlugin struct {
	Bot          bot.Bot
	enforceNicks bool
	sayings      []string
}

func New(bot bot.Bot) *TalkerPlugin {
	rand.Seed(time.Now().Unix())
	return &TalkerPlugin{
		Bot:          bot,
		enforceNicks: bot.Config().EnforceNicks,
		sayings:      bot.Config().WelcomeMsgs,
	}
}

func (p *TalkerPlugin) Message(message bot.Message) bool {
	channel := message.Channel
	body := message.Body
	lowermessage := strings.ToLower(body)

	// TODO: This ought to be space split afterwards to remove any punctuation
	if message.Command && strings.HasPrefix(lowermessage, "say") {
		msg := strings.TrimSpace(body[3:])
		p.Bot.SendMessage(channel, msg)
		return true
	}

	if message.Command && strings.HasPrefix(lowermessage, "goatse") {
		nick := message.User.Name
		if parts := strings.Split(message.Body, " "); len(parts) > 1 {
			nick = parts[1]
		}

		output := ""
		for _, line := range goatse {
			nick = fmt.Sprintf("%9.9s", nick)
			line = strings.Replace(line, "{nick}", nick, 1)
			output += line + "\n"
		}
		p.Bot.SendMessage(channel, output)
		return true
	}

	if p.enforceNicks && len(message.User.Name) != 9 {
		msg := fmt.Sprintf("Hey %s, we really like to have 9 character nicks because we're crazy OCD and stuff.",
			message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	}

	return false
}

func (p *TalkerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Hi, this is talker. I like to talk about FredFelps!")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *TalkerPlugin) Event(kind string, message bot.Message) bool {
	if kind == "JOIN" && strings.ToLower(message.User.Name) != strings.ToLower(p.Bot.Config().Nick) {
		if len(p.sayings) == 0 {
			return false
		}
		msg := fmt.Sprintf(p.sayings[rand.Intn(len(p.sayings))], message.User.Name)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	}
	return false
}

// Handler for bot's own messages
func (p *TalkerPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *TalkerPlugin) RegisterWeb() *string {
	return nil
}
