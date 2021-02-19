package remember

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

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

	b.RegisterRegex(p, bot.Message, rememberRegex, p.rememberCmd)
	b.RegisterRegex(p, bot.Message, quoteRegex, p.quoteCmd)
	b.RegisterRegex(p, bot.Message, regexp.MustCompile(`.*`), p.recordMsg)
	b.Register(p, bot.Help, p.help)

	return p
}

var quoteRegex = regexp.MustCompile(`(?i)^quote$`)
var rememberRegex = regexp.MustCompile(`(?i)^remember (?P<who>\S+) (?P<what>.*)$`)

func (p *RememberPlugin) quoteCmd(r bot.Request) bool {
	q := p.randQuote()
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, q)
	return true
}

func (p *RememberPlugin) rememberCmd(r bot.Request) bool {
	user := r.Msg.User

	nick := r.Values["who"]
	snip := r.Values["what"]
	for i := len(p.log[r.Msg.Channel]) - 1; i >= 0; i-- {
		entry := p.log[r.Msg.Channel][i]
		log.Debug().Msgf("Comparing %s:%s with %s:%s",
			entry.User.Name, entry.Body, nick, snip)
		if strings.ToLower(entry.User.Name) == strings.ToLower(nick) &&
			strings.Contains(
				strings.ToLower(entry.Body),
				strings.ToLower(snip),
			) {
			log.Debug().Msg("Found!")

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
				log.Error().Err(err)
				p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Tell somebody I'm broke.")
			}

			log.Info().
				Str("msg", msg).
				Msg("Remembering factoid")

			// sorry, not creative with names so we're reusing msg
			msg = fmt.Sprintf("Okay, %s, remembering '%s'.",
				r.Msg.User.Name, msg)
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
			return true

		}
	}
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "Sorry, I don't know that phrase.")
	return true
}

func (p *RememberPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	msg := "remember will let you quote your idiot friends. Just type " +
		"!remember <nick> <snippet> to remember what they said. Snippet can " +
		"be any part of their message. Later on, you can ask for a random " +
		"!quote."

	p.bot.Send(c, bot.Message, message.Channel, msg)
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
		log.Error().Err(err).Msg("Error getting quotes")
		return "I had a problem getting your quote."
	}
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)

	return f.Tidbit
}

func (p *RememberPlugin) recordMsg(r bot.Request) bool {
	log.Debug().Msgf("Logging message: %s: %s", r.Msg.User.Name, r.Msg.Body)
	p.log[r.Msg.Channel] = append(p.log[r.Msg.Channel], r.Msg)
	return false
}
