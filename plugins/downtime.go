package plugins

import "bitbucket.org/phlyingpenguin/godeepintir/bot"

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strings"
	"time"
)

// This is a downtime plugin to monitor how much our users suck

type DowntimePlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
}

type idleEntry struct {
	Nick     string
	LastSeen time.Time
}

// NewDowntimePlugin creates a new DowntimePlugin with the Plugin interface
func NewDowntimePlugin(bot *bot.Bot) *DowntimePlugin {
	p := DowntimePlugin{
		Bot: bot,
	}
	p.LoadData()
	return &p
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *DowntimePlugin) Message(message bot.Message) bool {
	// If it's a command and the payload is idle <nick>, give it. Log everything.

	parts := strings.Fields(strings.ToLower(message.Body))
	channel := message.Channel
	ret := false

	if parts[0] == "idle" && len(parts) == 2 {
		nick := parts[1]
		// parts[1] must be the userid, or we don't know them
		var entry idleEntry
		p.Coll.Find(bson.M{"nick": nick}).One(&entry)
		if entry.Nick != nick {
			// couldn't find em
			p.Bot.SendMessage(channel, fmt.Sprintf("Sorry, I don't know %s.", nick))
		} else {
			p.Bot.SendMessage(channel, fmt.Sprintf("%s has been idle for: %s",
				nick, time.Now().Sub(entry.LastSeen)))
		}
		ret = true
	}

	p.record(strings.ToLower(message.User.Name))

	return ret
}

func (p *DowntimePlugin) record(user string) {
	var entry idleEntry
	p.Coll.Find(bson.M{"nick": user}).One(&entry)
	if entry.Nick != user {
		// insert a new entry
		p.Coll.Insert(idleEntry{
			Nick:     user,
			LastSeen: time.Now(),
		})
	} else {
		// Update their entry, they were baaaaaad
		entry.LastSeen = time.Now()
		p.Coll.Upsert(bson.M{"nick": entry.Nick}, entry)

	}
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *DowntimePlugin) LoadData() {
	p.Coll = p.Bot.Db.C("downtime")
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *DowntimePlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Ask me how long one of your friends has been idele with, \"idle <nick>\"")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *DowntimePlugin) Event(kind string, message bot.Message) bool {
	if kind == "JOIN" && message.User.Name != p.Bot.Config.Nick {
		// user joined, let's nail them for it
		p.record(strings.ToLower(message.User.Name))
	}
	return false
}
