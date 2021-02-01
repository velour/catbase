package admin

import (
	"regexp"
	"strings"
	"testing"

	"github.com/velour/catbase/plugins/cli"

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
	err := mb.Config().Set("admins", "tester")
	if err != nil {
		t.FailNow()
	}
	return a, mb
}

func makeMessage(payload string, r *regexp.Regexp) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	c := cli.CliPlugin{}
	values := bot.ParseValues(r, payload)
	return bot.Request{
		Conn:   &c,
		Kind:   bot.Message,
		Values: values,
		Msg: msg.Message{
			User:    &user.User{Name: "admin"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func TestSet(t *testing.T) {
	a, mb := setup(t)
	expected := "test value"
	a.setConfigCmd(makeMessage("!set test.key "+expected, setConfigRegex))
	actual := mb.Config().Get("test.key", "ERR")
	assert.Equal(t, expected, actual)
}

func TestGetValue(t *testing.T) {
	a, mb := setup(t)
	expected := "value"
	mb.Config().Set("test.key", "value")
	a.getConfigCmd(makeMessage("!get test.key", getConfigRegex))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], expected)
}

func TestGetEmpty(t *testing.T) {
	a, mb := setup(t)
	expected := "test.key: <unknown>"
	a.getConfigCmd(makeMessage("!get test.key", getConfigRegex))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, expected, mb.Messages[0])
}

func TestGetForbidden(t *testing.T) {
	a, mb := setup(t)
	expected := "cannot access"
	a.getConfigCmd(makeMessage("!get slack.token", getConfigRegex))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], expected)
}
