// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package leftpad

import (
	"testing"

	"github.com/velour/catbase/plugins/cli"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
)

func makeMessage(payload string) bot.Request {
	values := bot.ParseValues(leftpadRegex, payload)
	return bot.Request{
		Kind:   bot.Message,
		Conn:   &cli.CliPlugin{},
		Values: values,
		Msg: msg.Message{
			User:    &user.User{Name: "tester"},
			Channel: "test",
			Body:    payload,
		},
	}

}

func testMessage(p *LeftpadPlugin, body string) {
	if leftpadRegex.MatchString(body) {
		p.leftpadCmd(makeMessage(body))
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
	testMessage(p, "leftpad test 8 test")
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "testtest")
	}
}

func TestNotCommand(t *testing.T) {
	p, mb := makePlugin(t)
	testMessage(p, "leftpad test fuck test")
	assert.Len(t, mb.Messages, 0)
}

func TestNoMaxLen(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "0")
	testMessage(p, "leftpad dicks 100 dicks")
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "dicks")
	}
}

func Test50Padding(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "50")
	assert.Equal(t, 50, p.config.GetInt("LeftPad.MaxLen", 100))
	testMessage(p, "leftpad dicks 100 dicks")
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "kill me")
	}
}

func TestUnder50Padding(t *testing.T) {
	p, mb := makePlugin(t)
	p.config.Set("LeftPad.MaxLen", "50")
	testMessage(p, "leftpad dicks 49 dicks")
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "dicks")
	}
}

func TestNotPadding(t *testing.T) {
	p, mb := makePlugin(t)
	testMessage(p, "lololol")
	assert.Len(t, mb.Messages, 0)
}
