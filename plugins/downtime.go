package plugins

import "bitbucket.org/phlyingpenguin/godeepintir/bot"

import (
	"labix.org/v2/mgo"
)

// This is a downtime plugin to monitor how much our users suck

type DowntimePlugin struct {
	Bot  *bot.Bot
	Coll *mgo.Collection
}

// NewDowntimePlugin creates a new DowntimePlugin with the Plugin interface
func NewDowntimePlugin(bot *bot.Bot) *DowntimePlugin {
	return &DowntimePlugin{
		Bot: bot,
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *DowntimePlugin) Message(message bot.Message) bool {
	// This bot does not reply to anything
	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *DowntimePlugin) LoadData() {
	p.Coll = p.Bot.Db.C("downtime")
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *DowntimePlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Sorry, Downtime does not do a goddamn thing.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *DowntimePlugin) Event(kind string, message bot.Message) bool {
	return false
}
