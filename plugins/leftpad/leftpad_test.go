// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package leftpad

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
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

func makePlugin(t *testing.T) (*LeftpadPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	p := New(mb)
	assert.NotNil(t, p)
	p.config.Set("LeftPad.MaxLen", "0")
	return p, mb
}

func TestLeftpad(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("!leftpad test 8 test"))
	assert.Contains(t, mb.Messages[0], "testtest")
	assert.Len(t, mb.Messages, 1)
}

func TestBadNumber(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("!leftpad test fuck test"))
	assert.Contains(t, mb.Messages[0], "Invalid")
	assert.Len(t, mb.Messages, 1)
}

func TestNotCommand(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("leftpad test fuck test"))
	assert.Len(t, mb.Messages, 0)
}

func TestNoMaxLen(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "0")
	p.message(makeMessage("!leftpad dicks 100 dicks"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "dicks")
}

func Test50Padding(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "50")
	assert.Equal(t, 50, p.config.GetInt("LeftPad.MaxLen", 100))
	p.message(makeMessage("!leftpad dicks 100 dicks"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "kill me")
}

func TestUnder50Padding(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "50")
	p.message(makeMessage("!leftpad dicks 49 dicks"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "dicks")
}

func TestNotPadding(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("!lololol"))
	assert.Len(t, mb.Messages, 0)
}

func TestRegisterWeb(t *testing.T) {
	p, _ := makePlugin(t)
	assert.Nil(t, p.RegisterWeb())
}
