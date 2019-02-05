// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package talker

import (
	"fmt"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
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
	Bot     bot.Bot
	sayings []string
}

func New(b bot.Bot) *TalkerPlugin {
	tp := &TalkerPlugin{
		Bot: b,
	}
	b.Register(tp, bot.Message, tp.message)
	b.Register(tp, bot.Help, tp.help)
	return tp
}

func (p *TalkerPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	channel := message.Channel
	body := message.Body
	lowermessage := strings.ToLower(body)

	// TODO: This ought to be space split afterwards to remove any punctuation
	if message.Command && strings.HasPrefix(lowermessage, "say") {
		msg := strings.TrimSpace(body[3:])
		p.Bot.Send(bot.Message, channel, msg)
		return true
	}

	if message.Command && strings.HasPrefix(lowermessage, "goatse") {
		nick := message.User.Name
		if parts := strings.Fields(message.Body); len(parts) > 1 {
			nick = parts[1]
		}

		output := ""
		for _, line := range goatse {
			nick = fmt.Sprintf("%9.9s", nick)
			line = strings.Replace(line, "{nick}", nick, 1)
			output += line + "\n"
		}
		p.Bot.Send(bot.Message, channel, output)
		return true
	}

	return false
}

func (p *TalkerPlugin) help(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(bot.Message, message.Channel, "Hi, this is talker. I like to talk about FredFelps!")
	return true
}

// Register any web URLs desired
func (p *TalkerPlugin) RegisterWeb() *string {
	return nil
}
