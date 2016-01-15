// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chrissexton/alepale/bot"
	"github.com/chrissexton/kakapo/lisp"
)

type LispPlugin struct {
	Bot *bot.Bot
}

type Program struct {
	Author   string
	Contents string
	Created  time.Time
}

// NewLispPlugin creates a new LispPlugin with the Plugin interface
func NewLispPlugin(bot *bot.Bot) *LispPlugin {
	return &LispPlugin{
		Bot: bot,
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *LispPlugin) Message(message bot.Message) bool {
	ch := message.Channel
	if strings.HasPrefix(message.Body, "lisp:") {
		prog := Program{
			Author:   message.User.Name,
			Contents: strings.Replace(message.Body, "lisp:", "", 1),
		}
		log.Println("Evaluating:", prog)

		defer func() {
			if r := recover(); r != nil {
				p.Bot.SendMessage(ch, fmt.Sprintf("%s", r))
			}
		}()

		results := lisp.EvalStr(prog.Contents)

		p.Bot.SendMessage(ch, fmt.Sprintf("%v", results))
		return true
	}
	return false
}

// LoadData imports any configuration data into the plugin. This is not
// strictly necessary other than the fact that the Plugin interface demands it
// exist. This may be deprecated at a later date.
func (p *LispPlugin) LoadData() {
	// This bot has no data to load
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *LispPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Type some lisp and see what happens...")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *LispPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *LispPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *LispPlugin) RegisterWeb() *string {
	return nil
}
