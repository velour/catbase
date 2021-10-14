// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/connectors/discord"
	"github.com/velour/catbase/plugins/giphy"
	"github.com/velour/catbase/plugins/last"
	"github.com/velour/catbase/plugins/mayi"
	"github.com/velour/catbase/plugins/quotegame"
	"github.com/velour/catbase/plugins/rest"
	"github.com/velour/catbase/plugins/secrets"

	"github.com/velour/catbase/plugins/achievements"
	"github.com/velour/catbase/plugins/aoc"
	"github.com/velour/catbase/plugins/countdown"
	"github.com/velour/catbase/plugins/goals"
	"github.com/velour/catbase/plugins/meme"
	"github.com/velour/catbase/plugins/sms"
	"github.com/velour/catbase/plugins/twitter"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/connectors/irc"
	"github.com/velour/catbase/connectors/slackapp"
	"github.com/velour/catbase/plugins/admin"
	"github.com/velour/catbase/plugins/babbler"
	"github.com/velour/catbase/plugins/beers"
	"github.com/velour/catbase/plugins/cli"
	"github.com/velour/catbase/plugins/couldashouldawoulda"
	"github.com/velour/catbase/plugins/counter"
	"github.com/velour/catbase/plugins/dice"
	"github.com/velour/catbase/plugins/emojifyme"
	"github.com/velour/catbase/plugins/fact"
	"github.com/velour/catbase/plugins/first"
	"github.com/velour/catbase/plugins/git"
	"github.com/velour/catbase/plugins/impossible"
	"github.com/velour/catbase/plugins/inventory"
	"github.com/velour/catbase/plugins/leftpad"
	"github.com/velour/catbase/plugins/nerdepedia"
	"github.com/velour/catbase/plugins/newsbid"
	"github.com/velour/catbase/plugins/picker"
	"github.com/velour/catbase/plugins/reaction"
	"github.com/velour/catbase/plugins/remember"
	"github.com/velour/catbase/plugins/reminder"
	"github.com/velour/catbase/plugins/rpgORdie"
	"github.com/velour/catbase/plugins/rss"
	"github.com/velour/catbase/plugins/sisyphus"
	"github.com/velour/catbase/plugins/stock"
	"github.com/velour/catbase/plugins/talker"
	"github.com/velour/catbase/plugins/tell"
	"github.com/velour/catbase/plugins/tldr"
	"github.com/velour/catbase/plugins/twitch"
	"github.com/velour/catbase/plugins/your"
)

var (
	key       = flag.String("set", "", "Configuration key to set")
	val       = flag.String("val", "", "Configuration value to set")
	initDB    = flag.Bool("init", false, "Initialize the configuration db")
	prettyLog = flag.Bool("pretty", false, "Use pretty console logger")
	debug     = flag.Bool("debug", false, "Turn on debug logging")
)

func main() {
	rand.Seed(time.Now().Unix())

	var dbpath = flag.String("db", "catbase.db",
		"Database file to load. (Defaults to catbase.db)")
	flag.Parse() // parses the logging flags.

	var output io.Writer = os.Stdout
	if *prettyLog {
		output = zerolog.ConsoleWriter{Out: output}
	}
	log.Logger = log.Output(output).With().Caller().Stack().Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	c := config.ReadConfig(*dbpath)

	if *key != "" && *val != "" {
		c.Set(*key, *val)
		log.Info().Msgf("Set config %s: %s", *key, *val)
		return
	}
	if (*initDB && len(flag.Args()) != 2) || (!*initDB && c.GetInt("init", 0) != 1) {
		log.Fatal().Msgf(`You must run "catbase -init <channel> <nick>"`)
	} else if *initDB {
		c.SetDefaults(flag.Arg(0), flag.Arg(1))
		return
	}

	var client bot.Connector

	switch c.Get("type", "slackapp") {
	case "irc":
		client = irc.New(c)
	case "slackapp":
		client = slackapp.New(c)
	case "discord":
		client = discord.New(c)
	default:
		log.Fatal().Msgf("Unknown connection type: %s", c.Get("type", "UNSET"))
	}

	b := bot.New(c, client)

	if r, path := client.GetRouter(); r != nil {
		b.RegisterWeb(r, path)
	}

	b.AddPlugin(admin.New(b))
	b.AddPlugin(secrets.New(b))
	b.AddPlugin(mayi.New(b))
	b.AddPlugin(giphy.New(b))
	b.AddPlugin(emojifyme.New(b))
	b.AddPlugin(last.New(b))
	b.AddPlugin(first.New(b))
	b.AddPlugin(leftpad.New(b))
	b.AddPlugin(talker.New(b))
	b.AddPlugin(dice.New(b))
	b.AddPlugin(picker.New(b))
	b.AddPlugin(beers.New(b))
	b.AddPlugin(remember.New(b))
	b.AddPlugin(your.New(b))
	b.AddPlugin(counter.New(b))
	b.AddPlugin(goals.New(b))
	b.AddPlugin(reminder.New(b))
	b.AddPlugin(babbler.New(b))
	b.AddPlugin(rss.New(b))
	b.AddPlugin(reaction.New(b))
	b.AddPlugin(twitch.New(b))
	b.AddPlugin(inventory.New(b))
	b.AddPlugin(rpgORdie.New(b))
	b.AddPlugin(sisyphus.New(b))
	b.AddPlugin(tell.New(b))
	b.AddPlugin(couldashouldawoulda.New(b))
	b.AddPlugin(nerdepedia.New(b))
	b.AddPlugin(tldr.New(b))
	b.AddPlugin(stock.New(b))
	b.AddPlugin(newsbid.New(b))
	b.AddPlugin(twitter.New(b))
	b.AddPlugin(git.New(b))
	b.AddPlugin(impossible.New(b))
	b.AddPlugin(cli.New(b))
	b.AddPlugin(aoc.New(b))
	b.AddPlugin(meme.New(b))
	b.AddPlugin(achievements.New(b))
	b.AddPlugin(sms.New(b))
	b.AddPlugin(countdown.New(b))
	b.AddPlugin(rest.New(b))
	b.AddPlugin(quotegame.New(b))
	// catches anything left, will always return true
	b.AddPlugin(fact.New(b))

	if err := client.Serve(); err != nil {
		log.Fatal().Err(err)
	}

	b.Receive(client, bot.Startup, msg.Message{})

	b.ListenAndServe()
}
