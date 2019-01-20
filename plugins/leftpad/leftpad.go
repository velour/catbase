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
func New(bot bot.Bot) *LeftpadPlugin {
	p := LeftpadPlugin{
		bot:    bot,
		config: bot.Config(),
	}
	return &p
}

type leftpadResp struct {
	Str string
}

func (p *LeftpadPlugin) Message(message msg.Message) bool {
	if !message.Command {
		return false
	}

	parts := strings.Fields(message.Body)
	if len(parts) > 3 && parts[0] == "leftpad" {
		padchar := parts[1]
		length, err := strconv.Atoi(parts[2])
		if err != nil {
			p.bot.SendMessage(message.Channel, "Invalid padding number")
			return true
		}
		if length > p.config.GetInt("LeftPad.MaxLen") && p.config.GetInt("LeftPad.MaxLen") > 0 {
			msg := fmt.Sprintf("%s would kill me if I did that.", p.config.Get("LeftPad.Who"))
			p.bot.SendMessage(message.Channel, msg)
			return true
		}
		text := strings.Join(parts[3:], " ")

		res := leftpad.LeftPad(text, length, padchar)

		p.bot.SendMessage(message.Channel, res)
		return true
	}

	return false
}

func (p *LeftpadPlugin) Event(e string, message msg.Message) bool {
	return false
}

func (p *LeftpadPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *LeftpadPlugin) Help(e string, m []string) {
}

func (p *LeftpadPlugin) RegisterWeb() *string {
	// nothing to register
	return nil
}

func (p *LeftpadPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
