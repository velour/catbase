// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package couldashouldawoulda

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeMessage(payload string) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Request{
		Kind: bot.Message,
		Msg: msg.Message{
			User:    &user.User{Name: "tester"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func Test0(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!should I drink a beer?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"Yes.", "No.", "Maybe.", "For fucks sake, how should I know?"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}

func Test1(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!should I drink a beer or a bourbon?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"The former.", "The latter.", "Obviously the former.", "Clearly the latter.", "Can't it be both?"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}

func Test2(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!could I drink a beer or a bourbon?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"Yes.", "No.", "Maybe.", "For fucks sake, how should I know?"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}

func Test3(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!would I die if I drank too much bourbon?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"Yes.", "No.", "Maybe.", "For fucks sake, how should I know?"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}

func Test4(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!would I die or be sick if I drank all the bourbon?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"The former.", "The latter.", "Obviously the former.", "Clearly the latter.", "Can't it be both?"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}

func Test5(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!should I have another beer or bourbon or tequila?"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	possibilities := []string{"I'd say option", "You'd be an idiot not to choose the"}
	match := false
	for _, possibility := range possibilities {
		if strings.Contains(mb.Messages[0], possibility) {
			match = true
			break
		}
	}
	assert.True(t, match)
}
