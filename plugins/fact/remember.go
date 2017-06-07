// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package fact

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new
// plugins.

type RememberPlugin struct {
	Bot bot.Bot
	Log map[string][]msg.Message
	db  *sqlx.DB
}

// NewRememberPlugin creates a new RememberPlugin with the Plugin interface
func NewRemember(b bot.Bot) *RememberPlugin {
	p := RememberPlugin{
		Bot: b,
		Log: make(map[string][]msg.Message),
		db:  b.DB(),
	}
	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *RememberPlugin) Message(message msg.Message) bool {

	if strings.ToLower(message.Body) == "quote" && message.Command {
		q := p.randQuote()
		p.Bot.SendMessage(message.Channel, q)

		// is it evil not to remember that the user said quote?
		return true
	}

	user := message.User
	parts := strings.Fields(message.Body)
	if message.Command && len(parts) >= 3 &&
		strings.ToLower(parts[0]) == "remember" {

		// we have a remember!
		// look through the logs and find parts[1] as a user, if not,
		// fuck this hoser
		// some people use @nick instead of just nick
		nick := strings.TrimPrefix(parts[1], "@")
		snip := strings.Join(parts[2:], " ")
		for i := len(p.Log[message.Channel]) - 1; i >= 0; i-- {
			entry := p.Log[message.Channel][i]
			log.Printf("Comparing %s:%s with %s:%s",
				entry.User.Name, entry.Body, nick, snip)
			if strings.ToLower(entry.User.Name) == strings.ToLower(nick) &&
				strings.Contains(
					strings.ToLower(entry.Body),
					strings.ToLower(snip),
				) {
				log.Printf("Found!")

				var msg string
				if entry.Action {
					msg = fmt.Sprintf("*%s* %s", entry.User.Name, entry.Body)
				} else {
					msg = fmt.Sprintf("<%s> %s", entry.User.Name, entry.Body)
				}

				trigger := fmt.Sprintf("%s quotes", entry.User.Name)

				fact := factoid{
					Fact:     strings.ToLower(trigger),
					Verb:     "reply",
					Tidbit:   msg,
					Owner:    user.Name,
					created:  time.Now(),
					accessed: time.Now(),
					Count:    0,
				}
				if err := fact.save(p.db); err != nil {
					log.Println("ERROR!!!!:", err)
					p.Bot.SendMessage(message.Channel, "Tell somebody I'm broke.")
				}

				log.Println("Remembering factoid:", msg)

				// sorry, not creative with names so we're reusing msg
				msg = fmt.Sprintf("Okay, %s, remembering '%s'.",
					message.User.Name, msg)
				p.Bot.SendMessage(message.Channel, msg)
				p.recordMsg(message)
				return true

			}
		}

		p.Bot.SendMessage(message.Channel, "Sorry, I don't know that phrase.")
		p.recordMsg(message)
		return true
	}
	p.recordMsg(message)
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *RememberPlugin) Help(channel string, parts []string) {

	msg := "!remember will let you quote your idiot friends. Just type " +
		"!remember <nick> <snippet> to remember what they said. Snippet can " +
		"be any part of their message. Later on, you can ask for a random " +
		"!quote."

	p.Bot.SendMessage(channel, msg)
}

// deliver a random quote out of the db.
// Note: this is the same cache for all channels joined. This plugin needs to be
// expanded to have this function execute a quote for a particular channel
func (p *RememberPlugin) randQuote() string {

	var f factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := p.db.QueryRow(`select * from factoid where fact like '%quotes'
		order by random() limit 1;`).Scan(
		&f.id,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	if err != nil {
		log.Println("Error getting quotes: ", err)
		return "I had a problem getting your quote."
	}
	f.created = time.Unix(tmpCreated, 0)
	f.accessed = time.Unix(tmpAccessed, 0)

	return f.Tidbit
}

// Empty event handler because this plugin does not do anything on event recv
func (p *RememberPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Record what the bot says in the log
func (p *RememberPlugin) BotMessage(message msg.Message) bool {
	p.recordMsg(message)
	return false
}

// Register any web URLs desired
func (p *RememberPlugin) RegisterWeb() *string {
	return nil
}

func (p *RememberPlugin) recordMsg(message msg.Message) {
	log.Printf("Logging message: %s: %s", message.User.Name, message.Body)
	p.Log[message.Channel] = append(p.Log[message.Channel], message)
}
