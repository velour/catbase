// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/irc"
	"github.com/velour/catbase/plugins/admin"
	"github.com/velour/catbase/plugins/babbler"
	"github.com/velour/catbase/plugins/beers"
	"github.com/velour/catbase/plugins/couldashouldawoulda"
	"github.com/velour/catbase/plugins/counter"
	"github.com/velour/catbase/plugins/db"
	"github.com/velour/catbase/plugins/dice"
	"github.com/velour/catbase/plugins/emojifyme"
	"github.com/velour/catbase/plugins/fact"
	"github.com/velour/catbase/plugins/first"
	"github.com/velour/catbase/plugins/inventory"
	"github.com/velour/catbase/plugins/leftpad"
	"github.com/velour/catbase/plugins/nerdepedia"
	"github.com/velour/catbase/plugins/picker"
	"github.com/velour/catbase/plugins/reaction"
	"github.com/velour/catbase/plugins/reminder"
	"github.com/velour/catbase/plugins/rpgORdie"
	"github.com/velour/catbase/plugins/rss"
	"github.com/velour/catbase/plugins/sisyphus"
	"github.com/velour/catbase/plugins/talker"
	"github.com/velour/catbase/plugins/tell"
	"github.com/velour/catbase/plugins/twitch"
	"github.com/velour/catbase/plugins/your"
	"github.com/velour/catbase/plugins/zork"
)

var (
	key    = flag.String("set", "", "Configuration key to set")
	val    = flag.String("val", "", "Configuration value to set")
	initDB = flag.Bool("init", false, "Initialize the configuration DB")
)

func main() {
	rand.Seed(time.Now().Unix())

	var dbpath = flag.String("db", "catbase.db",
		"Database file to load. (Defaults to catbase.db)")
	flag.Parse() // parses the logging flags.

	c := config.ReadConfig(*dbpath)

	if *key != "" && *val != "" {
		c.Set(*key, *val)
		log.Printf("Set config %s: %s", *key, *val)
		return
	}
	if (*initDB && len(flag.Args()) != 2) || (!*initDB && c.GetInt("init", 0) != 1) {
		log.Fatal(`You must run "catbase -init <channel> <nick>"`)
	} else if *initDB {
		c.SetDefaults(flag.Arg(0), flag.Arg(1))
		return
	}

	var client bot.Connector

	switch c.Get("type", "slack") {
	case "irc":
		client = irc.New(c)
	case "slack":
		//client = slack.New(c)
	default:
		log.Fatalf("Unknown connection type: %s", c.Get("type", "UNSET"))
	}

	b := bot.New(c, client)

	b.AddPlugin("admin", admin.New(b))
	b.AddPlugin("first", first.New(b))
	b.AddPlugin("leftpad", leftpad.New(b))
	b.AddPlugin("talker", talker.New(b))
	b.AddPlugin("dice", dice.New(b))
	b.AddPlugin("picker", picker.New(b))
	b.AddPlugin("beers", beers.New(b))
	b.AddPlugin("remember", fact.NewRemember(b))
	b.AddPlugin("your", your.New(b))
	b.AddPlugin("counter", counter.New(b))
	b.AddPlugin("reminder", reminder.New(b))
	b.AddPlugin("babbler", babbler.New(b))
	b.AddPlugin("zork", zork.New(b))
	b.AddPlugin("rss", rss.New(b))
	b.AddPlugin("reaction", reaction.New(b))
	b.AddPlugin("emojifyme", emojifyme.New(b))
	b.AddPlugin("twitch", twitch.New(b))
	b.AddPlugin("inventory", inventory.New(b))
	b.AddPlugin("rpgORdie", rpgORdie.New(b))
	b.AddPlugin("sisyphus", sisyphus.New(b))
	b.AddPlugin("tell", tell.New(b))
	b.AddPlugin("couldashouldawoulda", couldashouldawoulda.New(b))
	b.AddPlugin("nedepedia", nerdepedia.New(b))
	// catches anything left, will always return true
	b.AddPlugin("factoid", fact.New(b))
	b.AddPlugin("db", db.New(b))

	for {
		err := client.Serve()
		log.Println(err)
	}
}
