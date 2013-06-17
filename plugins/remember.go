package plugins

import (
	"fmt"
	"github.com/chrissexton/alepale/bot"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"math/rand"
	"strings"
	"time"
)

// This is a skeleton plugin to serve as an example and quick copy/paste for new
// plugins.

type RememberPlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
	Log  map[string][]bot.Message
}

// NewRememberPlugin creates a new RememberPlugin with the Plugin interface
func NewRememberPlugin(b *bot.Bot) *RememberPlugin {
	p := RememberPlugin{
		Bot: b,
		Log: make(map[string][]bot.Message),
	}
	p.LoadData()
	// for _, channel := range b.Config.Channels {
	// 	go p.quoteTimer(channel)
	// }
	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *RememberPlugin) Message(message bot.Message) bool {

	if message.Body == "quote" && message.Command {
		q := p.randQuote()
		p.Bot.SendMessage(message.Channel, q)

		// is it evil not to remember that the user said quote?
		return true
	}

	user := message.User
	parts := strings.Split(message.Body, " ")
	if message.Command && len(parts) >= 3 && parts[0] == "remember" {
		// we have a remember!
		// look through the logs and find parts[1] as a user, if not,
		// fuck this hoser
		nick := parts[1]
		snip := strings.Join(parts[2:], " ")

		for i := len(p.Log[message.Channel]) - 1; i >= 0; i-- {
			entry := p.Log[message.Channel][i]
			if strings.ToLower(entry.User.Name) == strings.ToLower(nick) &&
				strings.Contains(
					strings.ToLower(entry.Body),
					strings.ToLower(snip),
				) {
				// insert new remember entry
				var msg string

				// check if it's an action
				if entry.Action {
					msg = fmt.Sprintf("*%s* %s", entry.User.Name, entry.Body)
				} else {
					msg = fmt.Sprintf("<%s> %s", entry.User.Name, entry.Body)
				}

				trigger := fmt.Sprintf("%s quotes", entry.User.Name)

				var funcres bson.M
				err := p.Bot.Db.Run(
					bson.M{"eval": "return counter(\"factoid\");"},
					&funcres,
				)

				if err != nil {
					panic(err)
				}
				id := int(funcres["retval"].(float64))

				fact := Factoid{
					Id:           bson.NewObjectId(),
					Idx:          id,
					Trigger:      strings.ToLower(trigger),
					Operator:     "reply",
					FullText:     msg,
					Action:       msg,
					CreatedBy:    user.Name,
					DateCreated:  time.Now(),
					LastAccessed: time.Now(),
					AccessCount:  0,
				}
				if err = p.Coll.Insert(fact); err != nil {
					log.Println("ERROR!!!!:", err)
				}

				// sorry, not creative with names so we're reusing msg
				msg = fmt.Sprintf("Okay, %s, remembering '%s'.",
					message.User.Name, msg)
				p.Bot.SendMessage(message.Channel, msg)
				p.Log[message.Channel] = append(p.Log[message.Channel], message)
				return true
			}
		}
		p.Bot.SendMessage(message.Channel, "Sorry, I don't know that phrase.")
		p.Log[message.Channel] = append(p.Log[message.Channel], message)
		return true
	}
	p.Log[message.Channel] = append(p.Log[message.Channel], message)
	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly
// necessary other than the fact that the Plugin interface demands it exist.
// This may be deprecated at a later date.
func (p *RememberPlugin) LoadData() {
	p.Coll = p.Bot.Db.C("factoid")
	rand.Seed(time.Now().Unix())
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
	var quotes []Factoid
	// todo: find anything with the word "quotes" in the trigger
	query := bson.M{
		"trigger": bson.M{
			"$regex": "quotes$",
		},
	}
	iter := p.Coll.Find(query).Iter()
	err := iter.All(&quotes)
	if err != nil {
		panic(iter.Err())
	}

	// rand quote idx
	nquotes := len(quotes)
	if nquotes == 0 {
		return "Sorry, I don't know any quotes."
	}
	quote := quotes[rand.Intn(nquotes)]
	return quote.FullText
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
