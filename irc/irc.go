package irc

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
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

	eventRecieved   func(bot.Message)
	messageRecieved func(bot.Message)
}

func New(c *config.Config) *Irc {
	i := Irc{}
	i.config = c

	return &i
}

func (i *Irc) RegisterEventRecieved(f func(bot.Message)) {
	i.eventRecieved = f
}

func (i *Irc) RegisterMessageRecieved(f func(bot.Message)) {
	i.messageRecieved = f
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

func (i *Irc) Serve() {
	if i.eventRecieved == nil || i.messageRecieved == nil {
		log.Fatal("Missing an event handler")
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
		log.Fatal(err)
	}

	for _, c := range i.config.Channels {
		i.JoinChannel(c)
	}

	i.quit = make(chan bool)
	go i.handleConnection()
	<-i.quit
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
		i.eventRecieved(botMsg)

	case irc.ERR_NOSUCHCHANNEL:
		i.eventRecieved(botMsg)

	case irc.RPL_MOTD:
		i.eventRecieved(botMsg)

	case irc.RPL_NAMREPLY:
		i.eventRecieved(botMsg)

	case irc.RPL_TOPIC:
		i.eventRecieved(botMsg)

	case irc.KICK:
		i.eventRecieved(botMsg)

	case irc.TOPIC:
		i.eventRecieved(botMsg)

	case irc.MODE:
		i.eventRecieved(botMsg)

	case irc.JOIN:
		i.eventRecieved(botMsg)

	case irc.PART:
		i.eventRecieved(botMsg)

	case irc.QUIT:
		os.Exit(1)

	case irc.NOTICE:
		i.eventRecieved(botMsg)

	case irc.PRIVMSG:
		i.messageRecieved(botMsg)

	case irc.NICK:
		i.eventRecieved(botMsg)

	case irc.RPL_WHOREPLY:
		i.eventRecieved(botMsg)

	case irc.RPL_ENDOFWHO:
		i.eventRecieved(botMsg)

	default:
		cmd := irc.CmdNames[msg.Cmd]
		log.Println("(" + cmd + ") " + msg.Raw)
	}
}

// Builds our internal message type out of a Conn & Line from irc
func (i *Irc) buildMessage(inMsg irc.Msg) bot.Message {
	// Check for the user
	user := bot.User{
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
		iscmd, filteredMessage = i.isCmd(message)
	}

	msg := bot.Message{
		User:    &user,
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

// Checks if message is a command and returns its curtailed version
func (i *Irc) isCmd(message string) (bool, string) {
	cmdc := i.config.CommandChar
	botnick := strings.ToLower(i.config.Nick)
	iscmd := false
	lowerMessage := strings.ToLower(message)

	if strings.HasPrefix(lowerMessage, cmdc) && len(cmdc) > 0 {
		iscmd = true
		message = message[len(cmdc):]
		// } else if match, _ := regexp.MatchString(rex, lowerMessage); match {
	} else if strings.HasPrefix(lowerMessage, botnick) &&
		len(lowerMessage) > len(botnick) &&
		(lowerMessage[len(botnick)] == ',' || lowerMessage[len(botnick)] == ':') {

		iscmd = true
		message = message[len(botnick):]

		// trim off the customary addressing punctuation
		if message[0] == ':' || message[0] == ',' {
			message = message[1:]
		}
	}

	// trim off any whitespace left on the message
	message = strings.TrimSpace(message)

	return iscmd, message
}
