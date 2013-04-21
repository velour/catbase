package plugins

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"strings"
	"time"
)

// This is a first plugin to serve as an example and quick copy/paste for new plugins.

type FirstPlugin struct {
	First *FirstEntry
	Bot   *bot.Bot
	Coll  *mgo.Collection
}

type FirstEntry struct {
	Day  time.Time
	Time time.Time
	Body string
	Nick string
}

// NewFirstPlugin creates a new FirstPlugin with the Plugin interface
func NewFirstPlugin(b *bot.Bot) *FirstPlugin {
	coll := b.Db.C("first")
	var firsts []FirstEntry
	query := bson.M{"day": midnight(time.Now().UTC())}
	coll.Find(query).All(&firsts)

	var first *FirstEntry
	if len(firsts) > 0 {
		first = &firsts[0]
	}

	return &FirstPlugin{
		Bot:   b,
		Coll:  coll,
		First: first,
	}
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
		if isToday(p.First.Time) && p.allowed(message) {
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
	for _, msg := range p.Bot.Config.Bad.Msgs {
		if strings.ToLower(msg) == strings.ToLower(message.Body) {
			log.Println("Disallowing first: ", message.User.Name, ":", message.Body)
			return false
		}
	}
	for _, host := range p.Bot.Config.Bad.Hosts {
		if host == message.Host {
			log.Println("Disallowing first: ", message.User.Name, ":", message.Body)
			return false
		}
	}
	for _, nick := range p.Bot.Config.Bad.Nicks {
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
		Day:  midnight(time.Now()),
		Time: message.Time,
		Body: message.Body,
		Nick: message.User.Name,
	}
	p.Coll.Insert(p.First)
	p.announceFirst(message)
}

func (p *FirstPlugin) announceFirst(message bot.Message) {
	c := message.Channel
	if p.First != nil {
		p.Bot.SendMessage(c, fmt.Sprintf("%s had first at %s with the message: \"%s\"",
			p.First.Nick, p.First.Time.Format(time.Kitchen), p.First.Body))
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
