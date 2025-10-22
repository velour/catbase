// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Leftpad contains the plugin that allows the bot to pad messages
package leftpad

import (
	"fmt"
	"regexp"
	"strconv"

	"code.chrissexton.org/cws/leftpad"
	"github.com/velour/catbase/bot"
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
	b.RegisterRegexCmd(p, bot.Message, leftpadRegex, p.leftpadCmd)
	return p
}

type leftpadResp struct {
	Str string
}

var leftpadRegex = regexp.MustCompile(`(?i)^leftpad (?P<padstr>\S+) (?P<padding>\d+) (?P<text>.+)$`)

func (p *LeftpadPlugin) leftpadCmd(r bot.Request) bool {
	padchar := r.Values["padstr"]
	length, err := strconv.Atoi(r.Values["padding"])
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Invalid padding number")
		return true
	}
	maxLen, who := p.config.GetInt("LeftPad.MaxLen", 50), p.config.Get("LeftPad.Who", "Putin")
	if length > maxLen && maxLen > 0 {
		msg := fmt.Sprintf("%s would kill me if I did that.", who)
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
		return true
	}
	text := r.Values["text"]

	res := leftpad.LeftPad(text, length, padchar)

	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, res)
	return true
}
