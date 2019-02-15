package remember

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/plugins/fact"
)

type RememberPlugin struct {
	bot bot.Bot
	log map[string][]msg.Message
	db  *sqlx.DB
}

func New(b bot.Bot) *RememberPlugin {
	p := &RememberPlugin{
		bot: b,
		log: make(map[string][]msg.Message),
		db:  b.DB(),
	}

	b.Register(p, bot.Message, p.message)
	b.Register(p, bot.Help, p.help)

	return p
}

func (p *RememberPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if strings.ToLower(message.Body) == "quote" && message.Command {
		q := p.randQuote()
		p.bot.Send(bot.Message, message.Channel, q)

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
		nick := parts[1]
		snip := strings.Join(parts[2:], " ")
		for i := len(p.log[message.Channel]) - 1; i >= 0; i-- {
			entry := p.log[message.Channel][i]
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

				fact := fact.Factoid{
					Fact:     strings.ToLower(trigger),
					Verb:     "reply",
					Tidbit:   msg,
					Owner:    user.Name,
					Created:  time.Now(),
					Accessed: time.Now(),
					Count:    0,
				}
				if err := fact.Save(p.db); err != nil {
					log.Println("ERROR!!!!:", err)
					p.bot.Send(bot.Message, message.Channel, "Tell somebody I'm broke.")
				}

				log.Println("Remembering factoid:", msg)

				// sorry, not creative with names so we're reusing msg
				msg = fmt.Sprintf("Okay, %s, remembering '%s'.",
					message.User.Name, msg)
				p.bot.Send(bot.Message, message.Channel, msg)
				p.recordMsg(message)
				return true

			}
		}
		p.bot.Send(bot.Message, message.Channel, "Sorry, I don't know that phrase.")
		p.recordMsg(message)
		return true
	}

	p.recordMsg(message)
	return false
}

func (p *RememberPlugin) help(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	msg := "remember will let you quote your idiot friends. Just type " +
		"!remember <nick> <snippet> to remember what they said. Snippet can " +
		"be any part of their message. Later on, you can ask for a random " +
		"!quote."

	p.bot.Send(bot.Message, message.Channel, msg)
	return true
}

// deliver a random quote out of the db.
// Note: this is the same cache for all channels joined. This plugin needs to be
// expanded to have this function execute a quote for a particular channel
func (p *RememberPlugin) randQuote() string {

	var f fact.Factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := p.db.QueryRow(`select * from factoid where fact like '%quotes'
		order by random() limit 1;`).Scan(
		&f.ID,
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
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)

	return f.Tidbit
}

func (p *RememberPlugin) recordMsg(message msg.Message) {
	log.Printf("Logging message: %s: %s", message.User.Name, message.Body)
	p.log[message.Channel] = append(p.log[message.Channel], message)
}
