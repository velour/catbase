package remember

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/fact"
)

func makeMessage(nick, payload string, r *regexp.Regexp) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Request{
		Kind:   bot.Message,
		Values: bot.ParseValues(r, payload),
		Msg: msg.Message{
			User:    &user.User{Name: nick},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func makePlugin(t *testing.T) (*RememberPlugin, *fact.FactoidPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	f := fact.New(mb) // for db table
	p := New(mb)
	assert.NotNil(t, p)
	return p, f, mb
}

var allMsg = regexp.MustCompile(`.*`)

// Test case
func TestCornerCaseBug(t *testing.T) {
	msgs := []bot.Request{
		makeMessage("user1", "I don’t want to personally touch a horse dick.", allMsg),
		makeMessage("user3", "idk my bff rose?", allMsg),
	}
	rememberMsg := makeMessage("user2", "!remember user1 touch", rememberRegex)

	p, _, mb := makePlugin(t)

	for _, m := range msgs {
		p.recordMsg(m)
	}
	p.rememberCmd(rememberMsg)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "horse dick")
	q, err := fact.GetSingleFact(mb.DB(), "user1 quotes")
	assert.Nil(t, err)
	assert.Contains(t, q.Tidbit, "horse dick")
}
