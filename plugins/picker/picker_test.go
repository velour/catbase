// © 2018 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package picker

import (
	"strings"
	"testing"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/plugins/cli"

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
	assert.Contains(t, mb.Messages[0], "hot picks")
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

func TestPickDefaultWithSeprator(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	_ = c.message(makeMessage("!pick { a, b, c}"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "I've chosen")
	assert.NotContains(t, mb.Messages[0], "hot picks")
}

func TestPickDelimiter(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	_ = c.message(makeMessage("!pick; {a; b}"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "I've chosen")
	assert.NotContains(t, mb.Messages[0], "hot picks")
	log.Debug().Str("resp", mb.Messages[0]).Msg("choose")
}

func TestPickDelimiterMulti(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	_ = c.message(makeMessage("!pick; 2 {a; b}"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "hot picks")
	log.Debug().Str("resp", mb.Messages[0]).Msg("choose")
}

func TestPickDelimiterString(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	_ = c.message(makeMessage("!pick123 {a 123 b}"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "I've chosen")
	assert.NotContains(t, mb.Messages[0], "hot picks")
	log.Debug().Str("resp", mb.Messages[0]).Msg("choose")
}

func TestKnownBrokenPick(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	_ = c.message(makeMessage("!pick⌘ { bagel/egg/smoked turkey/butte/cheese ⌘ fuck all that, just have a bagel and cream cheese }"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "I've chosen")
	assert.NotContains(t, mb.Messages[0], "hot picks")
	log.Debug().Str("resp", mb.Messages[0]).Msg("choose")
}
