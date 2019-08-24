package tldr

import (
	"github.com/velour/catbase/plugins/cli"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

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

func makeMessageBy(payload, by string) (bot.Connector, bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return &cli.CliPlugin{}, bot.Message, msg.Message{
		User:    &user.User{Name: by},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeMessage(payload string) (bot.Connector, bot.Kind, msg.Message) {
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

func TestAddHistoryLimitsMessages(t *testing.T) {
	c, _ := setup(t)
	max := 1000
	c.bot.Config().Set("TLDR.HistorySize", strconv.Itoa(max))
	c.bot.Config().Set("TLDR.KeepHours", "24")
	t0 := time.Now().Add(-24 * time.Hour)
	for i := 0; i < max*2; i++ {
		hist := history{
			body:      "test",
			user:      "tester",
			timestamp: t0.Add(time.Duration(i) * time.Second),
		}
		c.addHistory(hist)
	}
	assert.Len(t, c.history, max)
}

func TestAddHistoryLimitsDays(t *testing.T) {
	c, _ := setup(t)
	hrs := 24
	expected := 24
	c.bot.Config().Set("TLDR.HistorySize", "100")
	c.bot.Config().Set("TLDR.KeepHours", strconv.Itoa(hrs))
	t0 := time.Now().Add(-time.Duration(hrs*2) * time.Hour)
	for i := 0; i < 48; i++ {
		hist := history{
			body:      "test",
			user:      "tester",
			timestamp: t0.Add(time.Duration(i) * time.Hour),
		}
		c.addHistory(hist)
	}
	assert.Len(t, c.history, expected, "%d != %d", len(c.history), expected)
}
