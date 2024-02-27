// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package twitch

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeRequest(payload string) bot.Request {
	c, k, m := makeMessage(payload)
	return bot.Request{
		Conn:   c,
		Kind:   k,
		Msg:    m,
		Values: nil,
		Args:   nil,
	}
}

func makeMessage(payload string) (bot.Connector, bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return nil, bot.Message, msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeTwitchPlugin(t *testing.T) (*Twitch, *bot.MockBot) {
	mb := bot.NewMockBot()
	c := New(mb)
	mb.Config().Set("twitch.clientid", "fake")
	mb.Config().Set("twitch.secret", "fake")
	c.c.SetArray("Twitch.Channels", []string{"test"})
	c.c.SetArray("Twitch.test.Users", []string{"drseabass"})
	assert.NotNil(t, c)

	c.twitchList["drseabass"] = &Twitcher{
		Name:   "drseabass",
		gameID: "",
	}

	return c, mb
}
