package tldr

import (
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func init() {
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func makeMessageBy(payload, by string) (bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Message, msg.Message{
		User:    &user.User{Name: by},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeMessage(payload string) (bot.Kind, msg.Message) {
	return makeMessageBy(payload, "tester")
}

func setup(t *testing.T) (*TLDRPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	r := New(mb)
	return r, mb
}

func Test(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("The quick brown fox jumped over the lazy dog"))
	res = c.message(makeMessage("The cow jumped over the moon"))
	res = c.message(makeMessage("The little dog laughed to see such fun"))
	res = c.message(makeMessage("tl;dr"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
}

func TestDoubleUp(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("The quick brown fox jumped over the lazy dog"))
	res = c.message(makeMessage("The cow jumped over the moon"))
	res = c.message(makeMessage("The little dog laughed to see such fun"))
	res = c.message(makeMessage("tl;dr"))
	res = c.message(makeMessage("tl;dr"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[1], "Slow down, cowboy.")
}
