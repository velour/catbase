// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package admin

import (
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

// This is a admin plugin to serve as an example and quick copy/paste for new plugins.

type AdminPlugin struct {
	Bot bot.Bot
	db  *sqlx.DB
	cfg *config.Config
}

// NewAdminPlugin creates a new AdminPlugin with the Plugin interface
func New(bot bot.Bot) *AdminPlugin {
	p := &AdminPlugin{
		Bot: bot,
		db:  bot.DB(),
		cfg: bot.Config(),
	}
	return p
}

var forbiddenKeys = map[string]bool{
	"twitch.authorization": true,
	"twitch.clientid":      true,
	"untappd.token":        true,
	"slack.token":          true,
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *AdminPlugin) Message(message msg.Message) bool {
	body := message.Body

	if len(body) > 0 && body[0] == '$' {
		return p.handleVariables(message)
	}

	if !message.Command {
		return false
	}

	parts := strings.Split(body, " ")
	if parts[0] == "set" && len(parts) > 2 && forbiddenKeys[parts[1]] {
		p.Bot.SendMessage(message.Channel, "You cannot access that key")
		return true
	} else if parts[0] == "set" && len(parts) > 2 {
		p.cfg.Set(parts[1], strings.Join(parts[2:], " "))
		p.Bot.SendMessage(message.Channel, fmt.Sprintf("Set %s", parts[1]))
		return true
	}
	if parts[0] == "get" && len(parts) == 2 && forbiddenKeys[parts[1]] {
		p.Bot.SendMessage(message.Channel, "You cannot access that key")
		return true
	} else if parts[0] == "get" && len(parts) == 2 {
		v := p.cfg.Get(parts[1], "<unknown>")
		p.Bot.SendMessage(message.Channel, fmt.Sprintf("%s: %s", parts[1], v))
		return true
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
