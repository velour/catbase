// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package your

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
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

func TestReplacement(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.config.Your.MaxLength = 1000
	c.config.Your.Replacements = []config.Replacement{
		config.Replacement{
			This:      "fuck",
			That:      "duck",
			Frequency: 1.0,
		},
	}
	res := c.Message(makeMessage("fuck a duck"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "duck a duck")
}

func TestNoReplacement(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.config.Your.MaxLength = 1000
	c.config.Your.Replacements = []config.Replacement{
		config.Replacement{
			This:      "nope",
			That:      "duck",
			Frequency: 1.0,
		},
		config.Replacement{
			This:      " fuck",
			That:      "duck",
			Frequency: 1.0,
		},
		config.Replacement{
			This:      "Fuck",
			That:      "duck",
			Frequency: 1.0,
		},
	}
	c.Message(makeMessage("fuck a duck"))
	assert.Len(t, mb.Messages, 0)
}
