// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package admin

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	Bot bot.Bot
	db  *sqlx.DB
}

// NewAdminPlugin creates a new AdminPlugin with the Plugin interface
func New(bot bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		Bot: bot,
		db:  bot.DB(),
	}
	p.LoadData()
	return p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *AdminPlugin) Message(message msg.Message) bool {
	body := message.Body

	if len(body) > 0 && body[0] == '$' {
		return p.handleVariables(message)
	}

	return false
}

func (p *AdminPlugin) handleVariables(message msg.Message) bool {
	if parts := strings.SplitN(message.Body, "!=", 2); len(parts) == 2 {
		variable := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		_, err := p.db.Exec(`delete from variables where name=? and value=?`, variable, value)
		if err != nil {
			p.Bot.SendMessage(message.Channel, "I'm broke and need attention in my variable creation code.")
			log.Println("[admin]: ", err)
		} else {
			p.Bot.SendMessage(message.Channel, "Removed.")
		}

		return true
	}

	parts := strings.SplitN(message.Body, "=", 2)
	if len(parts) != 2 {
		return false
	}

	variable := strings.ToLower(strings.TrimSpace(parts[0]))
	value := strings.TrimSpace(parts[1])

	var count int64
	row := p.db.QueryRow(`select count(*) from variables where value = ?`, variable, value)
	err := row.Scan(&count)
	if err != nil {
		p.Bot.SendMessage(message.Channel, "I'm broke and need attention in my variable creation code.")
		log.Println("[admin]: ", err)
		return true
	}

	if count > 0 {
		p.Bot.SendMessage(message.Channel, "I've already got that one.")
	} else {
		_, err := p.db.Exec(`INSERT INTO variables (name, value) VALUES (?, ?)`, variable, value)
		if err != nil {
			p.Bot.SendMessage(message.Channel, "I'm broke and need attention in my variable creation code.")
			log.Println("[admin]: ", err)
			return true
		}
		p.Bot.SendMessage(message.Channel, "Added.")
	}
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
func (p *AdminPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *AdminPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *AdminPlugin) RegisterWeb() *string {
	return nil
}

func (p *AdminPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
