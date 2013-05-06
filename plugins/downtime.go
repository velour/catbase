package plugins

import "github.com/chrissexton/alepale/bot"

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"sort"
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

type idleEntries []*idleEntry

func (ie idleEntries) Len() int {
	return len(ie)
}

func (ie idleEntries) Less(i, j int) bool {
	return ie[i].LastSeen.Before(ie[j].LastSeen)
}

func (ie idleEntries) Swap(i, j int) {
	ie[i], ie[j] = ie[j], ie[i]
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
	} else if parts[0] == "idle" && len(parts) == 1 {
		// Find all idle times, report them.
		var entries idleEntries
		p.Coll.Find(nil).All(&entries)
		sort.Sort(entries)
		tops := "The top entries are: "
		for _, e := range entries {

			// filter out ZNC entries and ourself
			if strings.HasPrefix(e.Nick, "*") || strings.ToLower(p.Bot.Config.Nick) == e.Nick {
				p.remove(e.Nick)
			} else {
				tops = fmt.Sprintf("%s%s: %s ", tops, e.Nick, time.Now().Sub(e.LastSeen))
			}
		}
		p.Bot.SendMessage(channel, tops)
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
		p.Coll.Upsert(bson.M{"nick": user}, idleEntry{
			Nick:     user,
			LastSeen: time.Now(),
		})
		log.Println("Inserted downtime for:", user)
	} else {
		// Update their entry, they were baaaaaad
		entry.LastSeen = time.Now()
		p.Coll.Upsert(bson.M{"nick": entry.Nick}, entry)
	}
}

func (p *DowntimePlugin) remove(user string) {
	p.Coll.RemoveAll(bson.M{"nick": user})
	log.Println("Removed downtime for:", user)
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
	log.Println(kind, "\t", message)
	if kind != "PART" && message.User.Name != p.Bot.Config.Nick {
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
