// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/velour/catbase/bot"
)

// This is a counter plugin to count arbitrary things.

type CounterPlugin struct {
	Bot *bot.Bot
	DB  *sql.DB
}

type Item struct {
	ID    int64
	Nick  string
	Item  string
	Count int
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
			subject = parts[1]
		}

		// pull all of the items associated with "subject"
		rows, err := p.DB.Query(`select * from counter where nick = ?`, subject)
		if err != nil {
			log.Fatal(err)
		}

		resp := fmt.Sprintf("%s has the following counters:", subject)
		count := 0
		for rows.Next() {
			count += 1
			var it Item
			rows.Scan(&it.Nick, &it.Item, &it.Count)

			if count > 1 {
				resp = fmt.Sprintf("%s, ", resp)
			}
			resp = fmt.Sprintf("%s %s: %d", resp, it.Item, it.Count)
			if count > 20 {
				fmt.Sprintf("%s, and a few others", resp)
				break
			}
		}
		resp = fmt.Sprintf("%s.", resp)

		if count == 0 {
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has no counters.", subject))
			return true
		}

		p.Bot.SendMessage(channel, resp)
		return true
	} else if message.Command && len(parts) == 2 && parts[0] == "clear" {
		subject := strings.ToLower(nick)
		itemName := strings.ToLower(parts[1])

		if _, err := p.DB.Exec(`delete from counter where nick = ? and item = ?`, subject, itemName); err != nil {
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
		err := p.DB.QueryRow(`select nick, item, count from counter
			where nick = ? and item = ?`, subject, itemName).Scan(
			&item.Nick, &item.Item, &item.Count,
		)
		switch {
		case err == sql.ErrNoRows:
			p.Bot.SendMessage(channel, fmt.Sprintf("I don't think %s has any %s.",
				subject, itemName))
			return true
		case err != nil:
			log.Println(err)
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
	err := p.DB.QueryRow(`select id, nick, item, count from counter
		where nick = ? and item = ?;`, subject, itemName).Scan(
		&item.ID, &item.Nick, &item.Item, &item.Count,
	)
	switch {
	case err == sql.ErrNoRows:
		// insert it
		res, err := p.DB.Exec(`insert into counter (nick, item, count)
			values (?, ?, ?)`, subject, itemName, delta)
		if err != nil {
			log.Println(err)
		}
		id, err := res.LastInsertId()
		return Item{id, subject, itemName, delta}
	case err != nil:
		log.Println(err)
		return item
	default:
		item.Count += delta
		_, err := p.DB.Exec(`update counter set count = ? where id = ?`, item.Count, item.ID)
		if err != nil {
			log.Println(err)
		}
		return item
	}
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
