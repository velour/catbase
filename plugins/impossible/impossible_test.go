package impossible

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/velour/catbase/plugins/cli"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
)

func makeMessage(payload string, r *regexp.Regexp) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	values := bot.ParseValues(r, payload)
	return bot.Request{
		Conn:   &cli.CliPlugin{},
		Kind:   bot.Message,
		Values: values,
		Msg: msg.Message{
			User:    &user.User{Name: "tester"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func makePlugin(t *testing.T) (*Impossible, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	p := newTesting(mb)
	assert.NotNil(t, p)
	return p, mb
}

func testMessage(p *Impossible, body string) {
	for _, h := range p.handlers {
		if h.Regex.MatchString(body) && h.Handler(makeMessage(body, h.Regex)) {
			return
		}
	}
}

func TestNothing(t *testing.T) {
	p, mb := makePlugin(t)
	testMessage(p, "hi")
	testMessage(p, "nothing")
	assert.Len(t, mb.Messages, 1)
}

func TestHint(t *testing.T) {
	p, mb := makePlugin(t)
	testMessage(p, "hi")
	testMessage(p, "hint")
	assert.Len(t, mb.Messages, 2)
}

func TestCorrect(t *testing.T) {
	p, mb := makePlugin(t)
	testMessage(p, "hi")
	testMessage(p, mb.Messages[0])

	congrats := fmt.Sprintf("You guessed the last impossible wikipedia article: \"%s\"", mb.Messages[0])

	assert.Contains(t, mb.Messages[1], congrats)
}
