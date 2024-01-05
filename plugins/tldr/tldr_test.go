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

func makeMessageBy(payload, by string) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}

	return bot.Request{
		Conn: &cli.CliPlugin{},
		Kind: bot.Message,
		Msg: msg.Message{
			User:    &user.User{Name: by},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func makeMessage(payload string) bot.Request {
	return makeMessageBy(payload, "tester")
}

func setup(t *testing.T) (*TLDRPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	r := New(mb)
	return r, mb
}

func TestAddHistoryLimitsDays(t *testing.T) {
	c, _ := setup(t)
	hrs := 24
	expected := 24
	c.b.Config().Set("TLDR.HistorySize", "100")
	c.b.Config().Set("TLDR.KeepHours", strconv.Itoa(hrs))
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
