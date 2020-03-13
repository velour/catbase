package impossible

import (
	"fmt"
	"github.com/velour/catbase/plugins/cli"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
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

func makePlugin(t *testing.T) (*Impossible, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	p := newTesting(mb)
	assert.NotNil(t, p)
	return p, mb
}

func TestNothing(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("hi"))
	p.message(makeMessage("nothing"))
	assert.Len(t, mb.Messages, 1)
}

func TestHint(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("hi"))
	p.message(makeMessage("!hint"))
	assert.Len(t, mb.Messages, 2)
}

func TestCorrect(t *testing.T) {
	p, mb := makePlugin(t)
	p.message(makeMessage("hi"))
	p.message(makeMessage(mb.Messages[0]))

	congrats := fmt.Sprintf("You guessed the last impossible wikipedia article: \"%s\"", mb.Messages[0])

	assert.Contains(t, mb.Messages[1], congrats)
}
