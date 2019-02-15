package fact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeMessage(nick, payload string) msg.Message {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return msg.Message{
		User:    &user.User{Name: nick},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makePlugin(t *testing.T) (*FactoidPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	f := New(mb) // for DB table
	return f, mb
}

func TestReact(t *testing.T) {
	msgs := []msg.Message{
		makeMessage("user1", "!testing123 <react> jesus"),
		makeMessage("user2", "testing123"),
	}
	p, mb := makePlugin(t)

	for _, m := range msgs {
		p.message(bot.Message, m)
	}
	assert.Len(t, mb.Reactions, 1)
	assert.Contains(t, mb.Reactions[0], "jesus")
}

func TestReactCantLearnSpaces(t *testing.T) {
	msgs := []msg.Message{
		makeMessage("user1", "!test <react> jesus christ"),
	}
	p, mb := makePlugin(t)

	for _, m := range msgs {
		p.message(bot.Message, m)
	}
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "not a valid")
}
