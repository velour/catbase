// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package dice

import (
	"github.com/velour/catbase/plugins/cli"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeMessage(payload string) (bot.Connector, bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return &cli.CliPlugin{}, bot.Message, msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func TestDie(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!1d6"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "tester, you rolled:")
}

func TestDice(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!5d6"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "tester, you rolled:")
}

func TestNotCommand(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("1d6"))
	assert.False(t, res)
	assert.Len(t, mb.Messages, 0)
}

func TestBadDice(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!aued6"))
	assert.False(t, res)
	assert.Len(t, mb.Messages, 0)
}

func TestBadSides(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!1daoeu"))
	assert.False(t, res)
	assert.Len(t, mb.Messages, 0)
}

func TestLotsOfDice(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!100d100"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "You're a dick.")
}

func TestHelp(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.help(&cli.CliPlugin{}, bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Len(t, mb.Messages, 1)
}
