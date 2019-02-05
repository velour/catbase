// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package nerdepedia

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

func TestWars(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("help me obi-wan"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}

func TestTrek(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("live long and prosper"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}

func TestDune(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("bless the maker"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}

func TestPoke(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("gotta catch em all"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}
