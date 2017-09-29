// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/irc"
	"github.com/velour/catbase/plugins/admin"
	"github.com/velour/catbase/plugins/babbler"
	"github.com/velour/catbase/plugins/beers"
	"github.com/velour/catbase/plugins/counter"
	"github.com/velour/catbase/plugins/dice"
	"github.com/velour/catbase/plugins/emojifyme"
	"github.com/velour/catbase/plugins/fact"
	"github.com/velour/catbase/plugins/first"
	"github.com/velour/catbase/plugins/leftpad"
	"github.com/velour/catbase/plugins/reaction"
	"github.com/velour/catbase/plugins/reminder"
	"github.com/velour/catbase/plugins/rss"
	"github.com/velour/catbase/plugins/stats"
	"github.com/velour/catbase/plugins/talker"
	"github.com/velour/catbase/plugins/twitch"
	"github.com/velour/catbase/plugins/your"
	"github.com/velour/catbase/plugins/zork"
	"github.com/velour/catbase/slack"
)

func main() {
	var cfile = flag.String("config", "config.lua",
		"Config file to load. (Defaults to config.lua)")
	flag.Parse() // parses the logging flags.

	c := config.Readconfig(Version, *cfile)
	var client bot.Connector

	switch c.Type {
	case "irc":
		client = irc.New(c)
	case "slack":
		client = slack.New(c)
	default:
		log.Fatalf("Unknown connection type: %s", c.Type)
	}

	b := bot.New(c, client)

	// b.AddHandler(plugins.NewTestPlugin(b))
	b.AddHandler("admin", admin.New(b))
	b.AddHandler("stats", stats.New(b))
	b.AddHandler("first", first.New(b))
	b.AddHandler("leftpad", leftpad.New(b))
	// b.AddHandler("downtime", downtime.New(b))
	b.AddHandler("talker", talker.New(b))
	b.AddHandler("dice", dice.New(b))
	b.AddHandler("beers", beers.New(b))
	b.AddHandler("remember", fact.NewRemember(b))
	b.AddHandler("your", your.New(b))
	b.AddHandler("counter", counter.New(b))
	b.AddHandler("reminder", reminder.New(b))
	b.AddHandler("babbler", babbler.New(b))
	b.AddHandler("zork", zork.New(b))
	b.AddHandler("rss", rss.New(b))
	b.AddHandler("reaction", reaction.New(b))
	b.AddHandler("emojifyme", emojifyme.New(b))
	b.AddHandler("twitch", twitch.New(b))
	// catches anything left, will always return true
	b.AddHandler("factoid", fact.New(b))

	for {
		err := client.Serve()
		log.Println(err)
	}
}
