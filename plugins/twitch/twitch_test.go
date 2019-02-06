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

func makeMessage(payload string) (bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Message, msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeTwitchPlugin(t *testing.T) (*TwitchPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	c := New(mb)
	mb.Config().Set("twitch.clientid", "fake")
	mb.Config().Set("twitch.authorization", "fake")
	c.config.SetArray("Twitch.Channels", []string{"test"})
	c.config.SetArray("Twitch.test.Users", []string{"drseabass"})
	assert.NotNil(t, c)

	c.twitchList["drseabass"] = &Twitcher{
		name:   "drseabass",
		gameID: "",
	}

	return c, mb
}

func TestTwitch(t *testing.T) {
	b, mb := makeTwitchPlugin(t)
	b.message(makeMessage("!twitch status"))
	assert.NotEmpty(t, mb.Messages)
}
