// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new
// plugins.

type RememberPlugin struct {
	Bot *bot.Bot
	Log map[string][]bot.Message
	db  *sqlx.DB
}

// NewRememberPlugin creates a new RememberPlugin with the Plugin interface
func NewRememberPlugin(b *bot.Bot) *RememberPlugin {
	p := RememberPlugin{
		Bot: b,
		Log: make(map[string][]bot.Message),
		db:  b.DB,
	}
	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *RememberPlugin) Message(message bot.Message) bool {

	if strings.ToLower(message.Body) == "quote" && message.Command {
		q := p.randQuote()
		p.Bot.SendMessage(message.Channel, q)

		// is it evil not to remember that the user said quote?
		return true
	}

	user := message.User
	parts := strings.Split(message.Body, " ")
	if message.Command && len(parts) >= 3 &&
		strings.ToLower(parts[0]) == "remember" {

		// we have a remember!
		// look through the logs and find parts[1] as a user, if not,
		// fuck this hoser
		snips := strings.Split(strings.Join(parts[1:], " "), "$and")
		var msgs []string
		var trigger string

		for _, snip := range snips {
			snip = strings.TrimSpace(snip)
			snipParts := strings.Split(snip, " ")
			nick := snipParts[0]
			snip := strings.Join(snipParts[1:], " ")

			for i := len(p.Log[message.Channel]) - 1; i >= 0; i-- {
				entry := p.Log[message.Channel][i]

				if strings.ToLower(entry.User.Name) == strings.ToLower(nick) &&
					strings.Contains(
						strings.ToLower(entry.Body),
						strings.ToLower(snip),
					) {

					// check if it's an action
					if entry.Action {
						msgs = append(msgs, fmt.Sprintf("*%s* %s", entry.User.Name, entry.Body))
					} else {
						msgs = append(msgs, fmt.Sprintf("<%s> %s", entry.User.Name, entry.Body))
					}

					if trigger == "" {
						trigger = fmt.Sprintf("%s quotes", entry.User.Name)
					}

				}
			}
		}

		if len(msgs) == len(snips) {
			msg := strings.Join(msgs, "$and")

			fact := factoid{
				fact:     strings.ToLower(trigger),
				verb:     "reply",
				tidbit:   msg,
				owner:    user.Name,
				created:  time.Now(),
				accessed: time.Now(),
				count:    0,
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
			p.Log[message.Channel] = append(p.Log[message.Channel], message)
			return true
		}

		p.Bot.SendMessage(message.Channel, "Sorry, I don't know that phrase.")
		p.Log[message.Channel] = append(p.Log[message.Channel], message)
		return true
	}
	p.Log[message.Channel] = append(p.Log[message.Channel], message)
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
		&f.fact,
		&f.tidbit,
		&f.verb,
		&f.owner,
		&tmpCreated,
		&tmpAccessed,
		&f.count,
	)
	if err != nil {
		log.Println("Error getting quotes: ", err)
		return "I had a problem getting your quote."
	}
	f.created = time.Unix(tmpCreated, 0)
	f.accessed = time.Unix(tmpAccessed, 0)

	return f.tidbit
}

func (p *RememberPlugin) quoteTimer(channel string) {
	for {
		// this pisses me off: You can't multiply int * time.Duration so it
		// has to look ugly as shit.
		time.Sleep(time.Duration(p.Bot.Config.QuoteTime) * time.Minute)
		chance := 1.0 / p.Bot.Config.QuoteChance
		if rand.Intn(int(chance)) == 0 {
			msg := p.randQuote()
			p.Bot.SendMessage(channel, msg)
		}
	}
}

// Empty event handler because this plugin does not do anything on event recv
func (p *RememberPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Record what the bot says in the log
func (p *RememberPlugin) BotMessage(message bot.Message) bool {
	p.Log[message.Channel] = append(p.Log[message.Channel], message)
	return false
}

// Register any web URLs desired
func (p *RememberPlugin) RegisterWeb() *string {
	return nil
}
