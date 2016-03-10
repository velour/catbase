// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/irc"
	"github.com/velour/catbase/plugins"
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
	default:
		log.Fatalf("Unknown connection type: %s", c.Type)
	}

	b := bot.NewBot(c, client)

	// b.AddHandler(plugins.NewTestPlugin(b))
	b.AddHandler("admin", plugins.NewAdminPlugin(b))
	b.AddHandler("first", plugins.NewFirstPlugin(b))
	b.AddHandler("downtime", plugins.NewDowntimePlugin(b))
	b.AddHandler("talker", plugins.NewTalkerPlugin(b))
	b.AddHandler("dice", plugins.NewDicePlugin(b))
	b.AddHandler("beers", plugins.NewBeersPlugin(b))
	b.AddHandler("counter", plugins.NewCounterPlugin(b))
	b.AddHandler("remember", plugins.NewRememberPlugin(b))
	b.AddHandler("skeleton", plugins.NewSkeletonPlugin(b))
	b.AddHandler("your", plugins.NewYourPlugin(b))
	// catches anything left, will always return true
	b.AddHandler("factoid", plugins.NewFactoidPlugin(b))

	client.Serve()
}
