package twitch

import (
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/connectors/discord"
	"strings"
)

func (t *Twitch) mkBridge(r bot.Request) bool {
	ircCh := "#" + r.Values["twitchChannel"]
	if err := t.startConn(); err != nil {
		t.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Could not connect to IRC: %s", err))
	}
	t.irc.Join(ircCh)
	t.bridgeMap[r.Msg.Channel] = ircCh
	t.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("This post is tracking %s", ircCh))
	return true
}

func (t *Twitch) rmBridge(r bot.Request) bool {
	ch, ok := t.bridgeMap[r.Msg.Channel]
	if ok {
		delete(t.bridgeMap, r.Msg.Channel)
		t.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("No longer tracking %s.", ch))
		return true
	}
	t.b.Send(r.Conn, bot.Message, r.Msg.Channel, "This is not a connected bridge channel.")
	return true
}

func (t *Twitch) bridgeMsg(r bot.Request) bool {
	if ircCh := t.bridgeMap[r.Msg.Channel]; ircCh != "" {
		if t.irc == nil {
			if err := t.startConn(); err != nil {
				t.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Could not connect to IRC: %s", err))
			}
			t.irc.Join(ircCh)
		}
		replaceSet := t.c.Get("twitch.replaceset", "\"'-")
		who := r.Msg.User.Name
		for _, c := range replaceSet {
			who = strings.ReplaceAll(who, string(c), "")
		}
		who = strings.Split(who, " ")[0][:9]
		msg := fmt.Sprintf("%s: %s", who, r.Msg.Body)
		t.irc.sendMessage(ircCh, msg)
	}
	return false
}

func (t *Twitch) ircMsg(channel, who, body string) {
	for thread, ircCh := range t.bridgeMap {
		if ircCh == channel {
			t.b.Send(t.b.DefaultConnector(), bot.Message, thread, fmt.Sprintf("%s: %s", who, body))
		}
	}
}

func (t *Twitch) startBridgeMsg(threadName, twitchChannel, msg string) error {
	if !strings.HasPrefix(twitchChannel, "#") {
		twitchChannel = "#" + twitchChannel
	}
	if err := t.startConn(); err != nil {
		return err
	}

	chID, err := t.mkForumPost(threadName, msg)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Opened thread %s", chID)

	err = t.irc.Join(twitchChannel)
	if err != nil {
		return err
	}
	t.bridgeMap[chID] = twitchChannel
	return nil
}

func (t *Twitch) startBridge(threadName, twitchChannel string) error {
	if !strings.HasPrefix(twitchChannel, "#") {
		twitchChannel = "#" + twitchChannel
	}
	msg := fmt.Sprintf("This post is tracking %s", twitchChannel)
	return t.startBridgeMsg(threadName, twitchChannel, msg)
}

func (t *Twitch) startConn() error {
	if t.irc == nil {
		err := backoff.Retry(func() error {
			err := t.connect()
			if err != nil {
				log.Error().Err(err).Msg("could not connect to IRC")
				return err
			}
			return nil
		}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Twitch) connect() error {
	t.ircLock.Lock()
	defer t.ircLock.Unlock()
	twitchServer := t.c.Get("twitch.ircserver", "irc.chat.twitch.tv:6697")
	twitchNick := t.c.Get("twitch.nick", "")
	twitchPass := t.c.Get("twitch.pass", "")
	twitchIRC, err := t.ConnectIRC(twitchServer, twitchNick, twitchPass, t.ircMsg, func() {
		t.ircLock.Lock()
		defer t.ircLock.Unlock()
		t.irc = nil
	})
	if err != nil {
		return err
	}
	t.irc = twitchIRC
	return nil
}

func (t *Twitch) mkForumPost(name, msg string) (string, error) {
	forum := t.c.Get("twitch.forum", "")
	if forum == "" {
		return "", fmt.Errorf("no forum available")
	}
	switch c := t.b.DefaultConnector().(type) {
	case *discord.Discord:
		chID, err := c.CreateRoom(name, msg, forum, t.c.GetInt("twitch.threadduration", 60))
		if err != nil {
			return "", err
		}
		return chID, nil
	default:
		return "", fmt.Errorf("non-Discord connectors not supported")
	}
}
