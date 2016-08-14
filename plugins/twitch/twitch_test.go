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

func makeMessage(payload string) msg.Message {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeTwitchPlugin(t *testing.T) (*TwitchPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Twitch.Users = map[string][]string{ "test" : []string{"drseabass"}}
	assert.NotNil(t, c)

	c.twitchList["drseabass"] = &Twitcher{
		name: "drseabass",
		game: "",
	}

	return c, mb
}

func TestTwitch(t *testing.T) {
	b, mb := makeTwitchPlugin(t)
	b.Message(makeMessage("!twitch status"))
	assert.NotEmpty(t, mb.Messages)
}
