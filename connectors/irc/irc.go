// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package irc

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
	"github.com/velour/velour/irc"
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

	actionPrefix = "\x01ACTION"
)

var throttle <-chan time.Time

type Irc struct {
	Client *irc.Client
	config *config.Config
	quit   chan bool

	event bot.Callback
}

func New(c *config.Config) *Irc {
	i := Irc{}
	i.config = c

	return &i
}

func (i *Irc) RegisterEvent(f bot.Callback) {
	i.event = f
}

func (i *Irc) Send(kind bot.Kind, args ...interface{}) (string, error) {
	switch kind {
	case bot.Reply:
	case bot.Message:
		return i.sendMessage(args[0].(string), args[1].(string))
	case bot.Action:
		return i.sendAction(args[0].(string), args[1].(string))
	default:
	}
	return "", nil
}

func (i *Irc) JoinChannel(channel string) {
	log.Info().Msgf("Joining channel: %s", channel)
	i.Client.Out <- irc.Msg{Cmd: irc.JOIN, Args: []string{channel}}
}

func (i *Irc) sendMessage(channel, message string) (string, error) {
	for len(message) > 0 {
		m := irc.Msg{
			Cmd:  "PRIVMSG",
			Args: []string{channel, message},
		}
		_, err := m.RawString()
		if err != nil {
			mtl := err.(irc.MsgTooLong)
			m.Args[1] = message[:mtl.NTrunc]
			message = message[mtl.NTrunc:]
		} else {
			message = ""
		}

		if throttle == nil {
			ratePerSec := i.config.GetInt("RatePerSec", 5)
			throttle = time.Tick(time.Second / time.Duration(ratePerSec))
		}

		<-throttle

		i.Client.Out <- m
	}
	return "NO_IRC_IDENTIFIERS", nil
}

// Sends action to channel
func (i *Irc) sendAction(channel, message string) (string, error) {
	message = actionPrefix + " " + message + "\x01"

	return i.sendMessage(channel, message)
}

func (i *Irc) GetEmojiList() map[string]string {
	//we're not going to do anything because it's IRC
	return make(map[string]string)
}

func (i *Irc) Serve() error {
	if i.event == nil {
		return fmt.Errorf("Missing an event handler")
	}

	var err error
	i.Client, err = irc.DialSSL(
		i.config.Get("Irc.Server", "localhost"),
		i.config.Get("Nick", "bot"),
		i.config.Get("FullName", "bot"),
		i.config.Get("Irc.Pass", ""),
		true,
	)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	for _, c := range i.config.GetArray("channels", []string{}) {
		i.JoinChannel(c)
	}

	i.quit = make(chan bool)
	go i.handleConnection()
	<-i.quit
	return nil
}

func (i *Irc) handleConnection() {
	t := time.NewTimer(pingTime)

	defer func() {
		t.Stop()
		close(i.Client.Out)
		for err := range i.Client.Errors {
			if err != io.EOF {
				log.Error().Err(err)
			}
		}
	}()

	for {
		select {
		case msg, ok := <-i.Client.In:
			if !ok { // disconnect
				i.quit <- true
				return
			}
			t.Stop()
			t = time.NewTimer(pingTime)
			i.handleMsg(msg)

		case <-t.C:
			i.Client.Out <- irc.Msg{Cmd: irc.PING, Args: []string{i.Client.Server}}
			t = time.NewTimer(pingTime)

		case err, ok := <-i.Client.Errors:
			if ok && err != io.EOF {
				log.Error().Err(err)
				i.quit <- true
				return
			}
		}
	}
}

// HandleMsg handles IRC messages from the server.
func (i *Irc) handleMsg(msg irc.Msg) {
	botMsg := i.buildMessage(msg)

	switch msg.Cmd {
	case irc.ERROR:
		log.Info().Msgf("Received error: " + msg.Raw)

	case irc.PING:
		i.Client.Out <- irc.Msg{Cmd: irc.PONG}

	case irc.PONG:
		// OK, ignore

	case irc.ERR_NOSUCHNICK:
		fallthrough

	case irc.ERR_NOSUCHCHANNEL:
		fallthrough

	case irc.RPL_MOTD:
		fallthrough

	case irc.RPL_NAMREPLY:
		fallthrough

	case irc.RPL_TOPIC:
		fallthrough

	case irc.KICK:
		fallthrough

	case irc.TOPIC:
		fallthrough

	case irc.MODE:
		fallthrough

	case irc.JOIN:
		fallthrough

	case irc.PART:
		fallthrough

	case irc.NOTICE:
		fallthrough

	case irc.NICK:
		fallthrough

	case irc.RPL_WHOREPLY:
		fallthrough

	case irc.RPL_ENDOFWHO:
		i.event(bot.Event, botMsg)

	case irc.PRIVMSG:
		i.event(bot.Message, botMsg)

	case irc.QUIT:
		os.Exit(1)

	default:
		cmd := irc.CmdNames[msg.Cmd]
		log.Debug().Msgf("(%s) %s", cmd, msg.Raw)
	}
}

// Builds our internal message type out of a Conn & Line from irc
func (i *Irc) buildMessage(inMsg irc.Msg) msg.Message {
	// Check for the user
	u := user.User{
		Name: inMsg.Origin,
	}

	channel := inMsg.Args[0]
	if channel == i.config.Get("Nick", "bot") {
		channel = inMsg.Args[0]
	}

	isAction := false
	var message string
	if len(inMsg.Args) > 1 {
		message = inMsg.Args[1]

		isAction = strings.HasPrefix(message, actionPrefix)
		if isAction {
			message = strings.TrimRight(message[len(actionPrefix):], "\x01")
			message = strings.TrimSpace(message)
		}

	}

	iscmd := false
	filteredMessage := message
	if !isAction {
		iscmd, filteredMessage = bot.IsCmd(i.config, message)
	}

	msg := msg.Message{
		User:    &u,
		Channel: channel,
		Body:    filteredMessage,
		Raw:     message,
		Command: iscmd,
		Action:  isAction,
		Time:    time.Now(),
		Host:    inMsg.Host,
	}

	return msg
}

func (i Irc) Who(channel string) []string {
	return []string{}
}
