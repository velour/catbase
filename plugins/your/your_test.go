// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package your

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

func setup(t *testing.T) (*YourPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	c := New(mb)
	mb.DB().MustExec(`delete from config;`)
	return c, mb
}

func TestReplacement(t *testing.T) {
	c, mb := setup(t)
	c.config.Set("Your.MaxLength", "1000")
	c.config.SetArray("your.replacements", []string{"0"})
	c.config.Set("your.replacements.0.freq", "1.0")
	c.config.Set("your.replacements.0.this", "fuck")
	c.config.Set("your.replacements.0.that", "duck")
	res := c.message(makeMessage("fuck a duck"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "duck a duck")
}

func TestNoReplacement(t *testing.T) {
	c, mb := setup(t)
	c.config.Set("Your.MaxLength", "1000")
	c.config.SetArray("your.replacements", []string{"0", "1", "2"})
	c.config.Set("your.replacements.0.freq", "1.0")
	c.config.Set("your.replacements.0.this", "nope")
	c.config.Set("your.replacements.0.that", "duck")

	c.config.Set("your.replacements.1.freq", "1.0")
	c.config.Set("your.replacements.1.this", "nope")
	c.config.Set("your.replacements.1.that", "duck")

	c.config.Set("your.replacements.2.freq", "1.0")
	c.config.Set("your.replacements.2.this", "Fuck")
	c.config.Set("your.replacements.2.that", "duck")
	c.message(makeMessage("fuck a duck"))
	assert.Len(t, mb.Messages, 0)
}
