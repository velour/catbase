package admin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

var (
	a  *AdminPlugin
	mb *bot.MockBot
)

func setup(t *testing.T) (*AdminPlugin, *bot.MockBot) {
	mb = bot.NewMockBot()
	a = New(mb)
	mb.DB().MustExec(`delete from config`)
	return a, mb
}

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

func TestSet(t *testing.T) {
	a, mb := setup(t)
	expected := "test value"
	a.Message(makeMessage("!set test.key " + expected))
	actual := mb.Config().Get("test.key", "ERR")
	assert.Equal(t, expected, actual)
}

func TestGetValue(t *testing.T) {
	a, mb := setup(t)
	expected := "value"
	mb.Config().Set("test.key", "value")
	a.Message(makeMessage("!get test.key"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], expected)
}

func TestGetEmpty(t *testing.T) {
	a, mb := setup(t)
	expected := "test.key: <unknown>"
	a.Message(makeMessage("!get test.key"))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, expected, mb.Messages[0])
}

func TestGetForbidden(t *testing.T) {
	a, mb := setup(t)
	expected := "cannot access"
	a.Message(makeMessage("!get slack.token"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], expected)
}
