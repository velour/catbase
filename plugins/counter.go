package plugins

import (
	"fmt"
	"github.com/chrissexton/alepale/bot"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
)

// This is a counter plugin to count arbitrary things.

type CounterPlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
}

type Item struct {
	Nick  string
	Item  string
	Count int
}

// NewCounterPlugin creates a new CounterPlugin with the Plugin interface
func NewCounterPlugin(bot *bot.Bot) *CounterPlugin {
	return &CounterPlugin{
		Bot:  bot,
		Coll: bot.Db.C("counter"),
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *CounterPlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything
	nick := message.User.Name
	channel := message.Channel
	parts := strings.Split(message.Body, " ")

	if len(parts) == 0 {
		return false
	}

	if message.Command && parts[0] == "inspect" && len(parts) == 2 {
		var subject string

		if parts[1] == "me" {
			subject = strings.ToLower(nick)
		} else {
			subject = parts[1]
		}

		// pull all of the items associated with "subject"
		var items []Item
		p.Coll.Find(bson.M{"nick": subject}).All(&items)

		if len(items) == 0 {
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has no counters.", subject))
			return true
		}

		resp := fmt.Sprintf("%s has the following counters:", subject)
		for i, item := range items {
			if i != 0 {
				resp = fmt.Sprintf("%s,", resp)
			}
			resp = fmt.Sprintf("%s %s: %d", resp, item.Item, item.Count)
			if i > 20 {
				fmt.Sprintf("%s, and a few others", resp)
				break
			}
		}
		resp = fmt.Sprintf("%s.", resp)

		p.Bot.SendMessage(channel, resp)
		return true
	} else if message.Command && len(parts) == 2 && parts[0] == "clear" {
		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[1])

		p.Coll.Remove(bson.M{"nick": subject, "item": itemName})

		p.Bot.SendAction(channel, fmt.Sprintf("chops a few %s out of his brain",
			itemName))

		return true

	} else if message.Command && parts[0] == "count" {
		var subject string
		var itemName string

		if len(parts) == 3 {
			// report count for parts[1]
			subject = strings.ToLower(parts[1])
			itemName = strings.ToLower(parts[2])
		} else if len(parts) == 2 {
			subject = strings.ToLower(nick)
			itemName = strings.ToLower(parts[1])
		} else {
			return false
		}

		var item Item
		err := p.Coll.Find(bson.M{"nick": subject, "item": itemName}).One(&item)
		if err != nil {
			p.Bot.SendMessage(channel, fmt.Sprintf("I don't think %s has any %s.",
				subject, itemName))
			return true
		}

		p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject, item.Count,
			itemName))

		return true
	} else if len(parts) == 1 {
		if len(parts[0]) < 3 {
			return false
		}

		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[0])[:len(parts[0])-2]

		if nameParts := strings.SplitN(itemName, ".", 2); len(nameParts) == 2 {
			subject = nameParts[0]
			itemName = nameParts[1]
		}

		if strings.HasSuffix(parts[0], "++") {
			// ++ those fuckers
			item := p.update(subject, itemName, 1)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		} else if strings.HasSuffix(parts[0], "--") {
			// -- those fuckers
			item := p.update(subject, itemName, -1)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	}

	return false
}

func (p *CounterPlugin) update(subject, itemName string, delta int) Item {
	var item Item
	err := p.Coll.Find(bson.M{"nick": subject, "item": itemName}).One(&item)
	if err != nil {
		// insert it
		item = Item{
			Nick:  subject,
			Item:  itemName,
			Count: delta,
		}
		p.Coll.Insert(item)
	} else {
		// update it
		item.Count += delta
		p.Coll.Update(bson.M{"nick": subject, "item": itemName}, item)
	}
	return item
}

// LoadData imports any configuration data into the plugin. This is not
// strictly necessary other than the fact that the Plugin interface demands it
// exist. This may be deprecated at a later date.
func (p *CounterPlugin) LoadData() {
	// This bot has no data to load
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *CounterPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "You can set counters incrementally by using "+
		"<noun>++ and <noun>--. You can see all of your counters using "+
		"\"inspect\", erase them with \"clear\", and view single counters with "+
		"\"count\".")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *CounterPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *CounterPlugin) BotMessage(message bot.Message) bool {
	return false
}
