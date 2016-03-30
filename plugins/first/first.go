// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package first

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
)

// This is a first plugin to serve as an example and quick copy/paste for new plugins.

type FirstPlugin struct {
	First *FirstEntry
	Bot   bot.Bot
	db    *sqlx.DB
}

type FirstEntry struct {
	id    int64
	day   time.Time
	time  time.Time
	body  string
	nick  string
	saved bool
}

// Insert or update the first entry
func (fe *FirstEntry) save(db *sqlx.DB) error {
	if _, err := db.Exec(`insert into first (day, time, body, nick)
		values (?, ?, ?, ?)`,
		fe.day.Unix(),
		fe.time.Unix(),
		fe.body,
		fe.nick,
	); err != nil {
		return err
	}
	return nil
}

// NewFirstPlugin creates a new FirstPlugin with the Plugin interface
func New(b bot.Bot) *FirstPlugin {
	if b.DBVersion() == 1 {
		_, err := b.DB().Exec(`create table if not exists first (
			id integer primary key,
			day integer,
			time integer,
			body string,
			nick string
		);`)
		if err != nil {
			log.Fatal("Could not create first table: ", err)
		}
	}

	log.Println("First plugin initialized with day:", midnight(time.Now()))

	first, err := getLastFirst(b.DB())
	if err != nil {
		log.Fatal("Could not initialize first plugin: ", err)
	}

	return &FirstPlugin{
		Bot:   b,
		db:    b.DB(),
		First: first,
	}
}

func getLastFirst(db *sqlx.DB) (*FirstEntry, error) {
	// Get last first entry
	var id sql.NullInt64
	var day sql.NullInt64
	var timeEntered sql.NullInt64
	var body sql.NullString
	var nick sql.NullString

	err := db.QueryRow(`select
		id, max(day), time, body, nick from first
		limit 1;
	`).Scan(
		&id,
		&day,
		&timeEntered,
		&body,
		&nick,
	)
	switch {
	case err == sql.ErrNoRows || !id.Valid:
		log.Println("No previous first entries")
		return nil, nil
	case err != nil:
		log.Println("Error on first query row: ", err)
		return nil, err
	}
	log.Println(id, day, timeEntered, body, nick)
	return &FirstEntry{
		id:    id.Int64,
		day:   time.Unix(day.Int64, 0),
		time:  time.Unix(timeEntered.Int64, 0),
		body:  body.String,
		nick:  nick.String,
		saved: true,
	}, nil
}

func midnight(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func isToday(t time.Time) bool {
	t0 := midnight(t)
	return t0.Before(midnight(time.Now()))
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *FirstPlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything

	if p.First == nil && p.allowed(message) {
		p.recordFirst(message)
		return false
	} else if p.First != nil {
		if isToday(p.First.time) && p.allowed(message) {
			p.recordFirst(message)
			return false
		}
	}

	r := strings.NewReplacer("'", "", "\"", "", ",", "", ".", "", ":", "",
		"?", "", "!", "")
	msg := strings.ToLower(message.Body)
	if r.Replace(msg) == "whos on first" {
		p.announceFirst(message)
		log.Printf("Disallowing %s: %s from first.",
			message.User.Name, message.Body)
		return true
	}

	return false
}

func (p *FirstPlugin) allowed(message bot.Message) bool {
	for _, msg := range p.Bot.Config().Bad.Msgs {
		match, err := regexp.MatchString(msg, strings.ToLower(message.Body))
		if err != nil {
			log.Println("Bad regexp: ", err)
		}
		if match {
			log.Println("Disallowing first: ", message.User.Name, ":", message.Body)
			return false
		}
	}
	for _, host := range p.Bot.Config().Bad.Hosts {
		if host == message.Host {
			log.Println("Disallowing first: ", message.User.Name, ":", message.Body)
			return false
		}
	}
	for _, nick := range p.Bot.Config().Bad.Nicks {
		if nick == message.User.Name {
			log.Println("Disallowing first: ", message.User.Name, ":", message.Body)
			return false
		}
	}
	return true
}

func (p *FirstPlugin) recordFirst(message bot.Message) {
	log.Println("Recording first: ", message.User.Name, ":", message.Body)
	p.First = &FirstEntry{
		day:  midnight(time.Now()),
		time: message.Time,
		body: message.Body,
		nick: message.User.Name,
	}
	err := p.First.save(p.db)
	if err != nil {
		log.Println("Error saving first entry: ", err)
		return
	}
	p.announceFirst(message)
}

func (p *FirstPlugin) announceFirst(message bot.Message) {
	c := message.Channel
	if p.First != nil {
		p.Bot.SendMessage(c, fmt.Sprintf("%s had first at %s with the message: \"%s\"",
			p.First.nick, p.First.time.Format(time.Kitchen), p.First.body))
	}
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *FirstPlugin) LoadData() {
	// This bot has no data to load
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *FirstPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Sorry, First does not do a goddamn thing.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *FirstPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *FirstPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *FirstPlugin) RegisterWeb() *string {
	return nil
}
