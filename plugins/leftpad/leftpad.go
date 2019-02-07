// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Leftpad contains the plugin that allows the bot to pad messages
package leftpad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chrissexton/leftpad"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type LeftpadPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// New creates a new LeftpadPlugin with the Plugin interface
func New(b bot.Bot) *LeftpadPlugin {
	p := &LeftpadPlugin{
		bot:    b,
		config: b.Config(),
	}
	b.Register(p, bot.Message, p.message)
	return p
}

type leftpadResp struct {
	Str string
}

func (p *LeftpadPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !message.Command {
		return false
	}

	parts := strings.Fields(message.Body)
	if len(parts) > 3 && parts[0] == "leftpad" {
		padchar := parts[1]
		length, err := strconv.Atoi(parts[2])
		if err != nil {
			p.bot.Send(bot.Message, message.Channel, "Invalid padding number")
			return true
		}
		maxLen, who := p.config.GetInt("LeftPad.MaxLen", 50), p.config.Get("LeftPad.Who", "Putin")
		if length > maxLen && maxLen > 0 {
			msg := fmt.Sprintf("%s would kill me if I did that.", who)
			p.bot.Send(bot.Message, message.Channel, msg)
			return true
		}
		text := strings.Join(parts[3:], " ")

		res := leftpad.LeftPad(text, length, padchar)

		p.bot.Send(bot.Message, message.Channel, res)
		return true
	}

	return false
}
