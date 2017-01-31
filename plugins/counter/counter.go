// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package counter

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

// This is a counter plugin to count arbitrary things.

type CounterPlugin struct {
	Bot bot.Bot
	DB  *sqlx.DB
}

type Item struct {
	*sqlx.DB

	ID    int64
	Nick  string
	Item  string
	Count int
}

// GetItems returns all counters for a subject
func GetItems(db *sqlx.DB, nick string) ([]Item, error) {
	var items []Item
	err := db.Select(&items, `select * from counter where nick = ?`, nick)
	if err != nil {
		return nil, err
	}
	// Don't forget to embed the DB into all of that shiz
	for i := range items {
		items[i].DB = db
	}
	return items, nil
}

// GetItem returns a specific counter for a subject
func GetItem(db *sqlx.DB, nick, itemName string) (Item, error) {
	var item Item
	item.DB = db
	log.Printf("GetItem db: %#v", db)
	err := db.Get(&item, `select * from counter where nick = ? and item= ?`,
		nick, itemName)
	switch err {
	case sql.ErrNoRows:
		item.ID = -1
		item.Nick = nick
		item.Item = itemName
	case nil:
	default:
		return Item{}, err
	}
	log.Printf("Got item %s.%s: %#v", nick, itemName, item)
	return item, nil
}

// Create saves a counter
func (i *Item) Create() error {
	res, err := i.Exec(`insert into counter (nick, item, count) values (?, ?, ?);`,
		i.Nick, i.Item, i.Count)
	id, _ := res.LastInsertId()
	// hackhackhack?
	i.ID = id
	return err
}

// UpdateDelta sets a value
// This will create or delete the item if necessary
func (i *Item) Update(value int) error {
	i.Count = value
	if i.Count == 0 && i.ID != -1 {
		return i.Delete()
	}
	if i.ID == -1 {
		i.Create()
	}
	log.Printf("Updating item: %#v, value: %d", i, value)
	_, err := i.Exec(`update counter set count = ? where id = ?`, i.Count, i.ID)
	return err
}

// UpdateDelta changes a value according to some delta
// This will create or delete the item if necessary
func (i *Item) UpdateDelta(delta int) error {
	i.Count += delta
	return i.Update(i.Count)
}

// Delete removes a counter from the database
func (i *Item) Delete() error {
	_, err := i.Exec(`delete from counter where id = ?`, i.ID)
	i.ID = -1
	return err
}

// NewCounterPlugin creates a new CounterPlugin with the Plugin interface
func New(bot bot.Bot) *CounterPlugin {
	if bot.DBVersion() == 1 {
		if _, err := bot.DB().Exec(`create table if not exists counter (
			id integer primary key,
			nick string,
			item string,
			count integer
		);`); err != nil {
			log.Fatal(err)
		}
	}
	return &CounterPlugin{
		Bot: bot,
		DB:  bot.DB(),
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the
// users message. Otherwise, the function returns false and the bot continues
// execution of other plugins.
func (p *CounterPlugin) Message(message msg.Message) bool {
	// This bot does not reply to anything
	nick := message.User.Name
	channel := message.Channel
	parts := strings.Fields(message.Body)

	if len(parts) == 0 {
		return false
	}

	if strings.ToLower(message.Body) == "tea. earl grey. hot." {
		item, err := GetItem(p.DB, nick, "🍵")
		if err != nil {
			log.Printf("Error finding item %s.%s: %s.", nick, "🍵", err)
			// Item ain't there, I guess
			return false
		}
		log.Printf("About to update item: %#v", item)
		item.UpdateDelta(1)
		p.Bot.SendMessage(channel, fmt.Sprintf("bleep-bloop-blop... %s has %d 🍵",
			nick, item.Count))
		return true
	} else if message.Command && message.Body == "reset me" {
		items, err := GetItems(p.DB, strings.ToLower(nick))
		if err != nil {
			log.Printf("Error getting items to reset %s: %s", nick, err)
			p.Bot.SendMessage(channel, "Something is technically wrong with your counters.")
			return true
		}
		log.Printf("Items: %+v", items)
		for _, item := range items {
			item.Delete()
		}
		p.Bot.SendMessage(channel, fmt.Sprintf("%s, you are as new, my son.", nick))
		return true
	} else if message.Command && parts[0] == "inspect" && len(parts) == 2 {
		var subject string

		if parts[1] == "me" {
			subject = strings.ToLower(nick)
		} else {
			subject = strings.ToLower(parts[1])
		}

		log.Printf("Getting counter for %s", subject)
		// pull all of the items associated with "subject"
		items, err := GetItems(p.DB, subject)
		if err != nil {
			log.Fatalf("Error retrieving items for %s: %s", subject, err)
			p.Bot.SendMessage(channel, "Something went wrong finding that counter;")
			return true
		}

		resp := fmt.Sprintf("%s has the following counters:", subject)
		count := 0
		for _, it := range items {
			count += 1
			if count > 1 {
				resp += ","
			}
			resp += fmt.Sprintf(" %s: %d", it.Item, it.Count)
			if count > 20 {
				resp += ", and a few others"
				break
			}
		}
		resp += "."

		if count == 0 {
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has no counters.", subject))
			return true
		}

		p.Bot.SendMessage(channel, resp)
		return true
	} else if message.Command && len(parts) == 2 && parts[0] == "clear" {
		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[1])

		it, err := GetItem(p.DB, subject, itemName)
		if err != nil {
			log.Printf("Error getting item to remove %s.%s: %s", subject, itemName, err)
			p.Bot.SendMessage(channel, "Something went wrong removing that counter;")
			return true
		}
		err = it.Delete()
		if err != nil {
			log.Printf("Error removing item %s.%s: %s", subject, itemName, err)
			p.Bot.SendMessage(channel, "Something went wrong removing that counter;")
			return true
		}

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
		item, err := GetItem(p.DB, subject, itemName)
		switch {
		case err == sql.ErrNoRows:
			p.Bot.SendMessage(channel, fmt.Sprintf("I don't think %s has any %s.",
				subject, itemName))
			return true
		case err != nil:
			log.Printf("Error retrieving item count for %s.%s: %s",
				subject, itemName, err)
			return true
		}

		p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject, item.Count,
			itemName))

		return true
	} else if len(parts) == 1 {
		// Need to have at least 3 characters to ++ or --
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
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Printf("Error finding item %s.%s: %s.", subject, itemName, err)
				// Item ain't there, I guess
				return false
			}
			log.Printf("About to update item: %#v", item)
			item.UpdateDelta(1)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		} else if strings.HasSuffix(parts[0], "--") {
			// -- those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Printf("Error finding item %s.%s: %s.", subject, itemName, err)
				// Item ain't there, I guess
				return false
			}
			item.UpdateDelta(-1)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	} else if len(parts) == 3 {
		// Need to have at least 3 characters to ++ or --
		if len(parts[0]) < 3 {
			return false
		}

		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[0])

		if nameParts := strings.SplitN(itemName, ".", 2); len(nameParts) == 2 {
			subject = nameParts[0]
			itemName = nameParts[1]
		}

		if parts[1] == "+=" {
			// += those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Printf("Error finding item %s.%s: %s.", subject, itemName, err)
				// Item ain't there, I guess
				return false
			}
			n, _ := strconv.Atoi(parts[2])
			log.Printf("About to update item by %d: %#v", n, item)
			item.UpdateDelta(n)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		} else if parts[1] == "-=" {
			// -= those fuckers
			item, err := GetItem(p.DB, subject, itemName)
			if err != nil {
				log.Printf("Error finding item %s.%s: %s.", subject, itemName, err)
				// Item ain't there, I guess
				return false
			}
			n, _ := strconv.Atoi(parts[2])
			log.Printf("About to update item by -%d: %#v", n, item)
			item.UpdateDelta(-n)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	}

	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *CounterPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "You can set counters incrementally by using "+
		"<noun>++ and <noun>--. You can see all of your counters using "+
		"\"inspect\", erase them with \"clear\", and view single counters with "+
		"\"count\".")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *CounterPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *CounterPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *CounterPlugin) RegisterWeb() *string {
	return nil
}
