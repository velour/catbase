package velouremon

import (
	"github.com/velour/catbase/plugins/cli"
	"os"
	"strings"
	"time"
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

func setup(t *testing.T) (*VelouremonPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	r := New(mb)
 	r.channel = "test"
	return r, mb
}

func TestStatus(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!status"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "tester: 255 HP, 0 XP")
}

func TestSimpleAppeared(t *testing.T) {
	c, mb := setup(t)
	c.timer.Reset(1 * time.Nanosecond)
	time.Sleep(1 * time.Millisecond)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "A wild")
}

func TestAddCreature(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!add_creature NewCreature 0 0"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "Added NewCreature")
}

func TestAddAbility(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!add_ability NewAbility 0 0 0 0 0"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "Added NewAbility")
}
