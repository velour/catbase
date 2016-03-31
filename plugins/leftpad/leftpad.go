// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Leftpad contains the plugin that allows the bot to pad messages
package leftpad

import (
	"strconv"
	"strings"

	"github.com/jamescun/leftpad"
	"github.com/velour/catbase/bot"
)

type LeftpadPlugin struct {
	bot bot.Bot
}

// New creates a new LeftpadPlugin with the Plugin interface
func New(bot bot.Bot) *LeftpadPlugin {
	p := LeftpadPlugin{
		bot: bot,
	}
	return &p
}

type leftpadResp struct {
	Str string
}

func (p *LeftpadPlugin) Message(message bot.Message) bool {
	if !message.Command {
		return false
	}

	parts := strings.Split(message.Body, " ")
	if len(parts) > 3 && parts[0] == "leftpad" {
		padchar := parts[1]
		length, err := strconv.Atoi(parts[2])
		if err != nil {
			p.bot.SendMessage(message.Channel, "Invalid padding number")
			return true
		}
		text := strings.Join(parts[3:], " ")

		res := leftpad.LeftPad(text, length, padchar)

		p.bot.SendMessage(message.Channel, res)
		return true
	}

	return false
}

func (p *LeftpadPlugin) Event(e string, message bot.Message) bool {
	return false
}

func (p *LeftpadPlugin) BotMessage(message bot.Message) bool {
	return false
}

func (p *LeftpadPlugin) Help(e string, m []string) {
}

func (p *LeftpadPlugin) RegisterWeb() *string {
	// nothing to register
	return nil
}
