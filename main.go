// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/irc"
	"github.com/velour/catbase/plugins"
	"github.com/velour/catbase/plugins/admin"
	"github.com/velour/catbase/plugins/beers"
	"github.com/velour/catbase/plugins/counter"
	"github.com/velour/catbase/plugins/dice"
	"github.com/velour/catbase/plugins/downtime"
	"github.com/velour/catbase/plugins/fact"
	"github.com/velour/catbase/plugins/talker"
	"github.com/velour/catbase/plugins/your"
	"github.com/velour/catbase/slack"
)

func main() {
	var cfile = flag.String("config", "config.json",
		"Config file to load. (Defaults to config.json)")
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

	b := bot.NewBot(c, client)

	// b.AddHandler(plugins.NewTestPlugin(b))
	b.AddHandler("admin", admin.NewAdminPlugin(b))
	// b.AddHandler("first", plugins.NewFirstPlugin(b))
	b.AddHandler("downtime", downtime.NewDowntimePlugin(b))
	b.AddHandler("talker", talker.NewTalkerPlugin(b))
	b.AddHandler("dice", dice.NewDicePlugin(b))
	b.AddHandler("beers", beers.NewBeersPlugin(b))
	b.AddHandler("counter", counter.NewCounterPlugin(b))
	b.AddHandler("remember", fact.NewRememberPlugin(b))
	b.AddHandler("skeleton", plugins.NewSkeletonPlugin(b))
	b.AddHandler("your", your.NewYourPlugin(b))
	// catches anything left, will always return true
	b.AddHandler("factoid", fact.NewFactoidPlugin(b))

	client.Serve()
}
