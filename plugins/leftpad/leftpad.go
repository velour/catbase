// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Leftpad contains the plugin that allows the bot to pad messages
package leftpad

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/velour/catbase/bot"
)

type LeftpadPlugin struct {
	bot *bot.Bot
}

// New creates a new LeftpadPlugin with the Plugin interface
func New(bot *bot.Bot) *LeftpadPlugin {
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
		length := parts[2]
		text := parts[3:][0]
		url := fmt.Sprintf("https://api.left-pad.io/?str=%s&len=%s&ch=%s",
			text,
			length,
			padchar,
		)
		log.Printf("Requesting leftpad url: %s", url)
		resp, err := http.Get(url)
		if err != nil {
			p.bot.SendMessage(message.Channel, err.Error())
			return true
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			p.bot.SendMessage(message.Channel, "I can't leftpad right now :(")
			log.Printf("Error decoding leftpad: %s", err)
			return true
		}
		r := leftpadResp{}
		err = json.Unmarshal(body, &r)
		if err != nil {
			p.bot.SendMessage(message.Channel, "I can't leftpad right now :(")
			log.Printf("Error decoding leftpad: %s", err)
			return true
		}
		p.bot.SendMessage(message.Channel, r.Str)
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

func (p *LeftpadPlugin) LoadData() {
}

func (p *LeftpadPlugin) Help(e string, m []string) {
}

func (p *LeftpadPlugin) RegisterWeb() *string {
	// nothing to register
	return nil
}
