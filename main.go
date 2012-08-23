package main

import (
	"bitbucket.org/phlyingpenguin/godeepintir/bot"
	"bitbucket.org/phlyingpenguin/godeepintir/config"
	"bitbucket.org/phlyingpenguin/godeepintir/plugins"
	"flag"
	"fmt"
)
import irc "github.com/fluffle/goirc/client"

func main() {

	// These belong in the JSON file
	// var server = flag.String("server", "irc.freenode.net", "Server to connect to.")
	// var nick = flag.String("nick", "CrappyBot", "Nick to set upon connection.")
	// var pass = flag.String("pass", "", "IRC server password to use")
	// var channel = flag.String("channel", "#AlePaleTest", "Channel to connet to.")

	var cfile = flag.String("config", "config.json", "Config file to load. (Defaults to config.json)")
	flag.Parse() // parses the logging flags.

	config := config.Readconfig(Version, *cfile)
	fmt.Println(config)

	c := irc.SimpleClient(config.Nick)
	// Optionally, enable SSL
	c.SSL = false

	// Add handlers to do things here!
	// e.g. join a channel on connect.
	c.AddHandler("connected",
		func(conn *irc.Conn, line *irc.Line) {
			for _, channel := range config.Channels {
				conn.Join(channel)
			}
		})
	// And a signal on disconnect
	quit := make(chan bool)

	c.AddHandler("disconnected",
		func(conn *irc.Conn, line *irc.Line) { quit <- true })

	b := bot.NewBot(config, c)
	// b.AddHandler(plugins.NewTestPlugin(b))
	b.AddHandler("skeleton", plugins.NewSkeletonPlugin(b))
	b.AddHandler("talker", plugins.NewTalkerPlugin(b))
	b.AddHandler("beers", plugins.NewBeersPlugin(b))
	b.AddHandler("remember", plugins.NewRememberPlugin(b))

	c.AddHandler("ACTION", func(conn *irc.Conn, line *irc.Line) {
		b.MsgRecieved(conn, line)
	})

	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		b.MsgRecieved(conn, line)
	})

	// Tell client to connect
	if err := c.Connect(config.Server, config.Pass); err != nil {
		fmt.Printf("Connection error: %s\n", err)
	}

	// Wait for disconnect
	<-quit
}
