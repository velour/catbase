// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"fmt"
	"github.com/chrissexton/alepale/bot"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"strings"
	"time"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	Bot                      *bot.Bot
	factC, remC, beerC, varC *mgo.Collection
}

// NewAdminPlugin creates a new AdminPlugin with the Plugin interface
func NewAdminPlugin(bot *bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		Bot: bot,
	}
	p.LoadData()
	return p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *AdminPlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything

	if !message.User.Admin {
		return false
	}

	body := message.Body

	if body[0] == '$' {
		return p.handleVariables(message)
	}

	return false
}

func (p *AdminPlugin) handleVariables(message bot.Message) bool {
	parts := strings.SplitN(message.Body, "=", 2)
	if len(parts) != 2 {
		return false
	}

	variable := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	q := p.varC.Find(bson.M{"variable": variable, "value": value})
	if n, _ := q.Count(); n != 0 {
		p.Bot.SendMessage(message.Channel, "I've already got that one.")
		return true
	}

	p.varC.Insert(bot.Variable{
		Variable: variable,
		Value:    value,
	})

	msg := fmt.Sprintf("Added '%s' to %s.\n", value, variable)
	p.Bot.SendMessage(message.Channel, msg)
	return true
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *AdminPlugin) LoadData() {
	// This bot has no data to load
	rand.Seed(time.Now().Unix())
	p.factC = p.Bot.Db.C("factoid")
	p.remC = p.Bot.Db.C("remember")
	p.beerC = p.Bot.Db.C("beers")
	p.varC = p.Bot.Db.C("variables")
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *AdminPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "This does super secret things that you're not allowed to know about.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *AdminPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *AdminPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *AdminPlugin) RegisterWeb() *string {
	return nil
}
