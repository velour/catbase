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
	"github.com/velour/catbase/slack"
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
		client = slack.New(c)
	default:
		log.Fatalf("Unknown connection type: %s", c.Get("type", "UNSET"))
	}

	b := bot.New(c, client)

	b.AddHandler("admin", admin.New(b))
	b.AddHandler("first", first.New(b))
	b.AddHandler("leftpad", leftpad.New(b))
	b.AddHandler("talker", talker.New(b))
	b.AddHandler("dice", dice.New(b))
	b.AddHandler("picker", picker.New(b))
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
	b.AddHandler("inventory", inventory.New(b))
	b.AddHandler("rpgORdie", rpgORdie.New(b))
	b.AddHandler("sisyphus", sisyphus.New(b))
	b.AddHandler("tell", tell.New(b))
	b.AddHandler("couldashouldawoulda", couldashouldawoulda.New(b))
	b.AddHandler("nedepedia", nerdepedia.New(b))
	// catches anything left, will always return true
	b.AddHandler("factoid", fact.New(b))
	b.AddHandler("db", db.New(b))

	for {
		err := client.Serve()
		log.Println(err)
	}
}
