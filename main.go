// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package main

//go:generate templ generate

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/connectors/discord"
	"github.com/velour/catbase/plugins"
	"io"
	"os"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/connectors/irc"
)

var (
	key       = flag.String("set", "", "Configuration key to set")
	val       = flag.String("val", "", "Configuration value to set")
	initDB    = flag.Bool("init", false, "Initialize the configuration db")
	prettyLog = flag.Bool("pretty", false, "Use pretty console logger")
	debug     = flag.Bool("debug", false, "Turn on debug logging")
	dbpath    = flag.String("db", "catbase.db", "Database file to load. (Defaults to catbase.db)")
	kvSpace   = flag.String("kv", "catbase", "Namespace for the charm store")
)

func main() {
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

	c := config.ReadConfig(*dbpath, *kvSpace)

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
	case "discord":
		client = discord.New(c)
	default:
		log.Fatal().Msgf("Unknown connection type: %s", c.Get("type", "UNSET"))
	}

	b := bot.New(c, client)

	if r, path := client.GetRouter(); r != nil {
		b.GetWeb().RegisterWeb(r, path)
	}

	plugins.Register(b)

	if err := client.Serve(); err != nil {
		log.Fatal().Err(err)
	}

	log.Debug().Msgf("Sending bot.Startup message")
	b.Receive(client, bot.Startup, msg.Message{})

	b.ListenAndServe()
}
