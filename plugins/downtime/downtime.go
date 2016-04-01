// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package downtime

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// This is a downtime plugin to monitor how much our users suck

type DowntimePlugin struct {
	Bot bot.Bot
	db  *sqlx.DB
}

type idleEntry struct {
	id       sql.NullInt64
	nick     string
	lastSeen time.Time
}

func (entry idleEntry) saveIdleEntry(db *sqlx.DB) error {
	var err error
	if entry.id.Valid {
		log.Println("Updating downtime for: ", entry)
		_, err = db.Exec(`update downtime set
			nick=?, lastSeen=?
			where id=?;`, entry.nick, entry.lastSeen.Unix(), entry.id.Int64)
	} else {
		log.Println("Inserting downtime for: ", entry)
		_, err = db.Exec(`insert into downtime (nick, lastSeen)
			values (?, ?)`, entry.nick, entry.lastSeen.Unix())
	}
	return err
}

func getIdleEntryByNick(db *sqlx.DB, nick string) (idleEntry, error) {
	var id sql.NullInt64
	var lastSeen sql.NullInt64
	err := db.QueryRow(`select id, max(lastSeen) from downtime
		where nick = ?`, nick).Scan(&id, &lastSeen)
	if err != nil {
		log.Println("Error selecting downtime: ", err)
		return idleEntry{}, err
	}
	if !id.Valid {
		return idleEntry{
			nick:     nick,
			lastSeen: time.Now(),
		}, nil
	}
	return idleEntry{
		id:       id,
		nick:     nick,
		lastSeen: time.Unix(lastSeen.Int64, 0),
	}, nil
}

func getAllIdleEntries(db *sqlx.DB) (idleEntries, error) {
	rows, err := db.Query(`select id, nick, max(lastSeen) from downtime
	group by nick`)
	if err != nil {
		return nil, err
	}
	entries := idleEntries{}
	for rows.Next() {
		var e idleEntry
		err := rows.Scan(&e.id, &e.nick, &e.lastSeen)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, nil
}

type idleEntries []*idleEntry

func (ie idleEntries) Len() int {
	return len(ie)
}

func (ie idleEntries) Less(i, j int) bool {
	return ie[i].lastSeen.Before(ie[j].lastSeen)
}

func (ie idleEntries) Swap(i, j int) {
	ie[i], ie[j] = ie[j], ie[i]
}

// NewDowntimePlugin creates a new DowntimePlugin with the Plugin interface
func New(bot bot.Bot) *DowntimePlugin {
	p := DowntimePlugin{
		Bot: bot,
		db:  bot.DB(),
	}

	if bot.DBVersion() == 1 {
		_, err := p.db.Exec(`create table if not exists downtime (
			id integer primary key,
			nick string,
			lastSeen integer
		);`)
		if err != nil {
			log.Fatal("Error creating downtime table: ", err)
		}
	}

	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *DowntimePlugin) Message(message msg.Message) bool {
	// If it's a command and the payload is idle <nick>, give it. Log everything.

	parts := strings.Fields(strings.ToLower(message.Body))
	channel := message.Channel
	ret := false

	if len(parts) == 0 {
		return false
	}

	if parts[0] == "idle" && len(parts) == 2 {
		nick := parts[1]
		// parts[1] must be the userid, or we don't know them
		entry, err := getIdleEntryByNick(p.db, nick)
		if err != nil {
			log.Println("Error getting idle entry: ", err)
		}
		if !entry.id.Valid {
			// couldn't find em
			p.Bot.SendMessage(channel, fmt.Sprintf("Sorry, I don't know %s.", nick))
		} else {
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has been idle for: %s",
				nick, time.Now().Sub(entry.lastSeen)))
		}
		ret = true
	} else if parts[0] == "idle" && len(parts) == 1 {
		// Find all idle times, report them.
		entries, err := getAllIdleEntries(p.db)
		if err != nil {
			log.Println("Error retrieving idle entries: ", err)
		}
		sort.Sort(entries)
		tops := "The top entries are: "
		for _, e := range entries {

			// filter out ZNC entries and ourself
			if strings.HasPrefix(e.nick, "*") || strings.ToLower(p.Bot.Config().Nick) == e.nick {
				p.remove(e.nick)
			} else {
				tops = fmt.Sprintf("%s%s: %s ", tops, e.nick, time.Now().Sub(e.lastSeen))
			}
		}
		p.Bot.SendMessage(channel, tops)
		ret = true

	}

	p.record(strings.ToLower(message.User.Name))

	return ret
}

func (p *DowntimePlugin) record(user string) {
	entry, err := getIdleEntryByNick(p.db, user)
	if err != nil {
		log.Println("Error recording downtime: ", err)
	}
	entry.lastSeen = time.Now()
	entry.saveIdleEntry(p.db)
	log.Println("Inserted downtime for:", user)
}

func (p *DowntimePlugin) remove(user string) error {
	_, err := p.db.Exec(`delete from downtime where nick = ?`, user)
	if err != nil {
		log.Println("Error removing downtime for user: ", user, err)
		return err
	}
	log.Println("Removed downtime for:", user)
	return nil
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *DowntimePlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Ask me how long one of your friends has been idele with, \"idle <nick>\"")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *DowntimePlugin) Event(kind string, message msg.Message) bool {
	log.Println(kind, "\t", message)
	if kind != "PART" && message.User.Name != p.Bot.Config().Nick {
		// user joined, let's nail them for it
		if kind == "NICK" {
			p.record(strings.ToLower(message.Channel))
			p.remove(strings.ToLower(message.User.Name))
		} else {
			p.record(strings.ToLower(message.User.Name))
		}
	} else if kind == "PART" || kind == "QUIT" {
		p.remove(strings.ToLower(message.User.Name))
	} else {
		log.Println("Unknown event: ", kind, message.User, message)
		p.record(strings.ToLower(message.User.Name))
	}
	return false
}

// Handler for bot's own messages
func (p *DowntimePlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *DowntimePlugin) RegisterWeb() *string {
	return nil
}
