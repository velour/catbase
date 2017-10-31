// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package irc

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

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

	eventReceived   func(msg.Message)
	messageReceived func(msg.Message)
}

func New(c *config.Config) *Irc {
	i := Irc{}
	i.config = c

	return &i
}

func (i *Irc) RegisterEventReceived(f func(msg.Message)) {
	i.eventReceived = f
}

func (i *Irc) RegisterMessageReceived(f func(msg.Message)) {
	i.messageReceived = f
}

func (i *Irc) JoinChannel(channel string) {
	log.Printf("Joining channel: %s", channel)
	i.Client.Out <- irc.Msg{Cmd: irc.JOIN, Args: []string{channel}}
}

func (i *Irc) SendMessage(channel, message string) {
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
			ratePerSec := i.config.RatePerSec
			throttle = time.Tick(time.Second / time.Duration(ratePerSec))
		}

		<-throttle

		i.Client.Out <- m
	}
}

// Sends action to channel
func (i *Irc) SendAction(channel, message string) {
	message = actionPrefix + " " + message + "\x01"

	i.SendMessage(channel, message)
}

func (i *Irc) React(channel, reaction string, message msg.Message) {
	//we're not goign to do anything because it's IRC
}

func (i *Irc) Edit(channel, newMessage, identifier string) {
	//we're not goign to do anything because it's IRC
}

func (i *Irc) GetEmojiList() map[string]string {
	//we're not goign to do anything because it's IRC
	return make(map[string]string)
}

func (i *Irc) Serve() error {
	if i.eventReceived == nil || i.messageReceived == nil {
		return fmt.Errorf("Missing an event handler")
	}

	var err error
	i.Client, err = irc.DialSSL(
		i.config.Irc.Server,
		i.config.Nick,
		i.config.FullName,
		i.config.Irc.Pass,
		true,
	)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	for _, c := range i.config.Channels {
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
				log.Println(err)
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
				log.Println(err)
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
		log.Println(1, "Received error: "+msg.Raw)

	case irc.PING:
		i.Client.Out <- irc.Msg{Cmd: irc.PONG}

	case irc.PONG:
		// OK, ignore

	case irc.ERR_NOSUCHNICK:
		i.eventReceived(botMsg)

	case irc.ERR_NOSUCHCHANNEL:
		i.eventReceived(botMsg)

	case irc.RPL_MOTD:
		i.eventReceived(botMsg)

	case irc.RPL_NAMREPLY:
		i.eventReceived(botMsg)

	case irc.RPL_TOPIC:
		i.eventReceived(botMsg)

	case irc.KICK:
		i.eventReceived(botMsg)

	case irc.TOPIC:
		i.eventReceived(botMsg)

	case irc.MODE:
		i.eventReceived(botMsg)

	case irc.JOIN:
		i.eventReceived(botMsg)

	case irc.PART:
		i.eventReceived(botMsg)

	case irc.QUIT:
		os.Exit(1)

	case irc.NOTICE:
		i.eventReceived(botMsg)

	case irc.PRIVMSG:
		i.messageReceived(botMsg)

	case irc.NICK:
		i.eventReceived(botMsg)

	case irc.RPL_WHOREPLY:
		i.eventReceived(botMsg)

	case irc.RPL_ENDOFWHO:
		i.eventReceived(botMsg)

	default:
		cmd := irc.CmdNames[msg.Cmd]
		log.Println("(" + cmd + ") " + msg.Raw)
	}
}

// Builds our internal message type out of a Conn & Line from irc
func (i *Irc) buildMessage(inMsg irc.Msg) msg.Message {
	// Check for the user
	u := user.User{
		Name: inMsg.Origin,
	}

	channel := inMsg.Args[0]
	if channel == i.config.Nick {
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
