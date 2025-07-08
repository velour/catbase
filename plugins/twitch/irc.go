package twitch

import (
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/config"
	"github.com/velour/velour/irc"
	"io"
	"time"
)

var throttle <-chan time.Time

type eventFunc func(channel, who, body string)

type IRC struct {
	t      *Twitch
	c      *config.Config
	client *irc.Client
	event  eventFunc
	quit   chan bool
}

func (t *Twitch) ConnectIRC(server, user, pass string, handler eventFunc, disconnect func()) (*IRC, error) {
	log.Debug().Msgf("Connecting to %s, %s:%s", server, user, pass)
	i := &IRC{
		t:     t,
		c:     t.c,
		event: handler}
	wait := make(chan bool)
	go i.serve(server, user, pass, wait, disconnect)
	<-wait
	return i, nil
}

func (i *IRC) Join(channel string) error {
	i.client.Out <- irc.Msg{Cmd: irc.JOIN, Args: []string{channel}}
	return nil
}

func (i *IRC) Say(channel, body string) error {
	if _, err := i.sendMessage(channel, body); err != nil {
		return err
	}
	return nil
}

func (i *IRC) serve(server, user, pass string, wait chan bool, disconnect func()) {
	if i.event == nil {
		log.Error().Msgf("Missing event handler")
		wait <- true
		return
	}

	var err error
	i.client, err = irc.DialSSL(server, user, user, pass, true)
	if err != nil {
		log.Error().
			Err(err).
			Strs("args", []string{server, user, pass}).
			Msgf("Could not connect")
		wait <- true
		return
	}

	i.client.Out <- irc.Msg{Cmd: "CAP REQ", Args: []string{":twitch.tv/membership"}}

	i.quit = make(chan bool)
	go i.handleConnection()
	wait <- true
	<-i.quit
	disconnect()
}

func (i *IRC) handleMsg(msg irc.Msg) {
	switch msg.Cmd {
	case irc.ERROR:
		log.Info().Str("raw", msg.Raw).Msgf("Received error")

	case irc.PING:
		i.client.Out <- irc.Msg{Cmd: irc.PONG}

	case irc.PONG:
		// OK, ignore

	case irc.KICK:
		fallthrough

	case irc.TOPIC:
		fallthrough

	case irc.NOTICE:
		fallthrough

	case irc.PRIVMSG:
		if len(msg.Args) < 2 {
			break
		}
		i.event(msg.Args[0], msg.Origin, msg.Args[1])

	case irc.QUIT:
		log.Debug().
			Interface("msg", msg).
			Msgf("QUIT")
		i.quit <- true

	default:
		log.Debug().
			Interface("msg", msg).
			Msgf("IRC EVENT")
	}
}

func (i *IRC) sendMessage(channel, message string, args ...any) (string, error) {
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
			ratePerSec := i.c.GetInt("RatePerSec", 5)
			throttle = time.Tick(time.Second / time.Duration(ratePerSec))
		}

		<-throttle

		i.client.Out <- m
	}
	return "NO_IRC_IDENTIFIERS", nil
}

func (i *IRC) handleConnection() {
	pingTime := time.Duration(i.c.GetInt("twitch.pingtime", 60)) * time.Second
	t := time.NewTimer(pingTime)

	defer func() {
		t.Stop()
		close(i.client.Out)
		for err := range i.client.Errors {
			if err != io.EOF {
				log.Error().Err(err)
			}
		}
	}()

	for {
		select {
		case msg, ok := <-i.client.In:
			if !ok { // disconnect
				i.quit <- true
				return
			}
			t.Stop()
			t = time.NewTimer(pingTime)
			i.handleMsg(msg)

		case <-t.C:
			i.client.Out <- irc.Msg{Cmd: irc.PING, Args: []string{i.client.Server}}
			t = time.NewTimer(pingTime)

		case err, ok := <-i.client.Errors:
			if ok && err != io.EOF {
				log.Error().Err(err)
				i.quit <- true
				return
			}
		}
	}
}
