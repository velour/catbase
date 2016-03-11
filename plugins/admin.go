// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	Bot *bot.Bot
	DB  *sql.DB
}

// NewAdminPlugin creates a new AdminPlugin with the Plugin interface
func NewAdminPlugin(bot *bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		Bot: bot,
		DB:  bot.DB,
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

	if len(body) == 0 {
		return false
	}

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
	value := parts[1]

	var count int64
	var varId int64
	err := p.DB.QueryRow(`select count(*), varId from variables vs inner join "values" v on vs.id = v.varId where vs.name = ? and v.value = ?`, variable, value).Scan(&count)
	switch {
	case err == sql.ErrNoRows:
		_, err := p.DB.Exec(`insert into "values" (varId, value) values (?, ?)`, varId, value)
		if err != nil {
			log.Println(err)
		}
		msg := fmt.Sprintf("Added '%s' to %s.\n", value, variable)
		p.Bot.SendMessage(message.Channel, msg)
		return true
	case err != nil:
		p.Bot.SendMessage(message.Channel, "I'm broke and need attention in my variable creation code.")
		log.Println("Admin error: ", err)
		return true
	}
	p.Bot.SendMessage(message.Channel, "I've already got that one.")
	return true
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *AdminPlugin) LoadData() {
	// This bot has no data to load
	rand.Seed(time.Now().Unix())
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
