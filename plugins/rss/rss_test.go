package rss

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
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

func TestRSS(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!rss http://rss.cnn.com/rss/edition.rss"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}

func TestRSSCache(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!rss http://rss.cnn.com/rss/edition.rss"))
	assert.True(t, res)
	res = c.Message(makeMessage("!rss http://rss.cnn.com/rss/edition.rss"))
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
}
