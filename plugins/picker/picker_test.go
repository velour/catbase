// © 2018 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package picker

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

func TestPick2(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!pick 2 { a, b,c}"))
	assert.Len(t, mb.Messages, 1)
	if !res {
		t.Fatalf("expected a successful choice, got %q", mb.Messages[0])
	}
}

func TestPickDefault(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	_ = c.message(makeMessage("!pick { a}"))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, `I've chosen "a" for you.`, mb.Messages[0])
}
