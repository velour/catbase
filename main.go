// © 2013 the AlePale Authors under the WTFPL. See AUTHORS for the list of authors.

package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"code.google.com/p/velour/irc"
	"github.com/chrissexton/alepale/bot"
	"github.com/chrissexton/alepale/config"
	"github.com/chrissexton/alepale/plugins"
)

const (
	// DefaultPort is the port used to connect to
	// the server if one is not specified.
	defaultPort = "6667"

	// InitialTimeout is the initial amount of time
	// to delay before reconnecting.  Each failed
	// reconnection doubles the timout until
	// a connection is made successfully.
	initialTimeout = 2 * time.Second

	// PingTime is the amount of inactive time
	// to wait before sending a ping to the server.
	pingTime = 120 * time.Second
)

var (
	Client *irc.Client
	Bot    *bot.Bot
	Config *config.Config
)

func main() {
	var err error
	var cfile = flag.String("config", "config.json",
		"Config file to load. (Defaults to config.json)")
	flag.Parse() // parses the logging flags.

	Config = config.Readconfig(Version, *cfile)

	Client, err = irc.DialServer(Config.Server,
		Config.Nick,
		Config.FullName,
		Config.Pass)

	if err != nil {
		log.Fatal(err)
	}

	Bot = bot.NewBot(Config, Client)
	// Bot.AddHandler(plugins.NewTestPlugin(Bot))
	Bot.AddHandler("admin", plugins.NewAdminPlugin(Bot))
	Bot.AddHandler("first", plugins.NewFirstPlugin(Bot))
	Bot.AddHandler("downtime", plugins.NewDowntimePlugin(Bot))
	Bot.AddHandler("talker", plugins.NewTalkerPlugin(Bot))
	Bot.AddHandler("dice", plugins.NewDicePlugin(Bot))
	Bot.AddHandler("beers", plugins.NewBeersPlugin(Bot))
	Bot.AddHandler("counter", plugins.NewCounterPlugin(Bot))
	// Bot.AddHandler("remember", plugins.NewRememberPlugin(Bot))
	Bot.AddHandler("skeleton", plugins.NewSkeletonPlugin(Bot))
	Bot.AddHandler("your", plugins.NewYourPlugin(Bot))
	// catches anything left, will always return true
	Bot.AddHandler("factoid", plugins.NewFactoidPlugin(Bot))

	handleConnection()

	// And a signal on disconnect
	quit := make(chan bool)

	// Wait for disconnect
	<-quit
}

func handleConnection() {
	t := time.NewTimer(pingTime)

	defer func() {
		t.Stop()
		close(Client.Out)
		for err := range Client.Errors {
			if err != io.EOF {
				log.Println(err)
			}
		}
	}()

	for {
		select {
		case msg, ok := <-Client.In:
			if !ok { // disconnect
				return
			}
			t.Stop()
			t = time.NewTimer(pingTime)
			handleMsg(msg)

		case <-t.C:
			Client.Out <- irc.Msg{Cmd: irc.PING, Args: []string{Client.Server}}
			t = time.NewTimer(pingTime)

		case err, ok := <-Client.Errors:
			if ok && err != io.EOF {
				log.Println(err)
				return
			}
		}
	}
}

// HandleMsg handles IRC messages from the server.
func handleMsg(msg irc.Msg) {
	switch msg.Cmd {
	case irc.ERROR:
		log.Println(1, "Received error: "+msg.Raw)

	case irc.PING:
		Client.Out <- irc.Msg{Cmd: irc.PONG}

	case irc.PONG:
		// OK, ignore

	case irc.ERR_NOSUCHNICK:
		Bot.EventRecieved(Client, msg)

	case irc.ERR_NOSUCHCHANNEL:
		Bot.EventRecieved(Client, msg)

	case irc.RPL_MOTD:
		Bot.EventRecieved(Client, msg)

	case irc.RPL_NAMREPLY:
		Bot.EventRecieved(Client, msg)

	case irc.RPL_TOPIC:
		Bot.EventRecieved(Client, msg)

	case irc.KICK:
		Bot.EventRecieved(Client, msg)

	case irc.TOPIC:
		Bot.EventRecieved(Client, msg)

	case irc.MODE:
		Bot.EventRecieved(Client, msg)

	case irc.JOIN:
		Bot.EventRecieved(Client, msg)

	case irc.PART:
		Bot.EventRecieved(Client, msg)

	case irc.QUIT:
		os.Exit(1)

	case irc.NOTICE:
		Bot.EventRecieved(Client, msg)

	case irc.PRIVMSG:
		Bot.MsgRecieved(Client, msg)

	case irc.NICK:
		Bot.EventRecieved(Client, msg)

	case irc.RPL_WHOREPLY:
		Bot.EventRecieved(Client, msg)

	case irc.RPL_ENDOFWHO:
		Bot.EventRecieved(Client, msg)

	default:
		cmd := irc.CmdNames[msg.Cmd]
		log.Println("(" + cmd + ") " + msg.Raw)
	}
}
