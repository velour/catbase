// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

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

func TestBabbler(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	assert.Len(t, c.babblers, 1)
	seabass.Body = "This is another message"
	res = c.Message(seabass)
	seabass.Body = "This is a long message"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "this is")
	assert.Contains(t, mb.Messages[0], "message")
}

func TestBabblerBatch(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("batch learn for seabass This is a message! This is another message. This is not a long message? This is not a message! This is not another message. This is a long message?")
	res := c.Message(seabass)
	assert.Len(t, c.babblers, 2)
	assert.Len(t, mb.Messages, 1)
	res = c.Message(makeMessage("!seabass says"))
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[1], "this is")
	assert.Contains(t, mb.Messages[1], "message")
}

func TestBabblerMerge(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)

	seabass := makeMessage("<seabass> This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	assert.Len(t, c.babblers, 1)
	assert.Len(t, mb.Messages, 0)

	seabass.Body = "<seabass> This is another message"
	res = c.Message(seabass)

	seabass.Body = "<seabass> This is a long message"
	res = c.Message(seabass)

	res = c.Message(makeMessage("!merge babbler seabass into seabass2"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "mooooiggged")

	res = c.Message(makeMessage("!seabass2 says"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 2)

	assert.Contains(t, mb.Messages[1], "<seabass2> this is")
	assert.Contains(t, mb.Messages[1], "message")
}

func TestHelp(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Help("channel", []string{})
	assert.Len(t, mb.Messages, 1)
}

func TestBotMessage(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.False(t, c.BotMessage(makeMessage("test")))
}

func TestEvent(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.False(t, c.Event("dummy", makeMessage("test")))
}

func TestRegisterWeb(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.Nil(t, c.RegisterWeb())
}
