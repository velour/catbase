// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package counter

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
)

// This is a counter plugin to count arbitrary things.

type CounterPlugin struct {
	Bot *bot.Bot
	DB  *sqlx.DB
}

type Item struct {
	*sqlx.DB

	ID    int64
	Nick  string
	Item  string
	Count int
}

func GetItems(db *sqlx.DB, nick string) ([]Item, error) {
	var items []Item
	err := db.Select(&items, `select * from counter where nick = ?`, nick)
	if err != nil {
		return nil, err
	}
	// Don't forget to embed the DB into all of that shiz
	for _, i := range items {
		i.DB = db
	}
	return items, nil
}

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

func (i *Item) Create() error {
	res, err := i.Exec(`insert into counter (nick, item, count) values (?, ?, ?);`,
		i.Nick, i.Item, i.Count)
	id, _ := res.LastInsertId()
	// hackhackhack?
	i.ID = id
	return err
}

func (i *Item) Update(delta int) error {
	i.Count += delta
	if i.Count == 0 && i.ID != -1 {
		return i.Delete()
	}
	if i.ID == -1 {
		i.Create()
	}
	log.Printf("Updating item: %#v, delta: %d", i, delta)
	_, err := i.Exec(`update counter set count = ? where id = ?`, i.Count, i.ID)
	return err
}

func (i *Item) Delete() error {
	_, err := i.Exec(`delete from counter where id = ?`, i.ID)
	i.ID = -1
	return err
}

// NewCounterPlugin creates a new CounterPlugin with the Plugin interface
func NewCounterPlugin(bot *bot.Bot) *CounterPlugin {
	if bot.DBVersion == 1 {
		if _, err := bot.DB.Exec(`create table if not exists counter (
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
		DB:  bot.DB,
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
			subject = strings.ToLower(parts[1])
		}

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
				resp += ", "
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
			item.Update(1)
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
			item.Update(-1)
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has %d %s.", subject,
				item.Count, item.Item))
			return true
		}
	}

	return false
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

// Register any web URLs desired
func (p *CounterPlugin) RegisterWeb() *string {
	return nil
}
