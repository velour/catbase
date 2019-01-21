// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package talker

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

func TestGoatse(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("goatse"))
	assert.Len(t, mb.Messages, 0)
	assert.False(t, res)
}

func TestGoatseCommand(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!goatse"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "g o a t s e")
}

func TestGoatseWithNickCommand(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!goatse seabass"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "g o a t s e")
	assert.Contains(t, mb.Messages[0], "seabass")
}

func TestSay(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("say hello"))
	assert.Len(t, mb.Messages, 0)
	assert.False(t, res)
}

func TestSayCommand(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!say hello"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "hello")
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
