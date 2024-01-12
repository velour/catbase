// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

import (
	"regexp"
	"strings"
	"testing"

	"github.com/velour/catbase/plugins/cli"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeMessage(payload string, r *regexp.Regexp) bot.Request {
	c := &cli.CliPlugin{}
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Request{
		Conn:   c,
		Kind:   bot.Message,
		Values: bot.ParseValues(r, payload),
		Msg: msg.Message{
			User:    &user.User{Name: "tester"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func newBabblerPlugin(mb *bot.MockBot) *BabblerPlugin {
	bp := New(mb)
	bp.WithGoRoutines = false
	mb.DB().MustExec(`
		delete from babblers;
		delete from babblerWords;
		delete from babblerNodes;
		delete from babblerArcs;
	`)
	return bp
}

func testMessage(p *BabblerPlugin, msg string) bool {
	for _, h := range p.handlers {
		if h.Regex.MatchString(msg) {
			req := makeMessage(msg, h.Regex)
			if h.Handler(req) {
				return true
			}
		}
	}
	return false
}

func TestBabblerNoBabbler(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)
	testMessage(bp, "!seabass2 says")
	res := assert.Len(t, mb.Messages, 0)
	assert.True(t, res)
	// assert.Contains(t, mb.Messages[0], "seabass2 babbler not found")
}

func TestBabblerNothingSaid(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)
	res := testMessage(bp, "initialize babbler for seabass")
	assert.True(t, res)
	res = testMessage(bp, "seabass says")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 2) {
		assert.Contains(t, mb.Messages[0], "okay.")
		assert.Contains(t, mb.Messages[1], "seabass hasn't said anything yet.")
	}
}

func testBabbler(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)
	testMessage(bp, "!initialize babbler for tester")
	testMessage(bp, "This is a message")
	testMessage(bp, "This is another message")
	testMessage(bp, "This is a long message")
	res := testMessage(bp, "!tester says")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "this is")
		assert.Contains(t, mb.Messages[0], "message")
	}
}

func TestBabblerSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is a message")
	testMessage(bp, "This is another message")
	testMessage(bp, "This is a long message")
	res := testMessage(bp, "tester says long")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "long message")
	}
}

func TestBabblerMultiSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is a message")
	testMessage(bp, "This is another message")
	testMessage(bp, "This is a long message")
	res := testMessage(bp, "tester says is another")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "is another")
	}
}

func TestBabblerBadSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is a message")
	testMessage(bp, "This is another message")
	testMessage(bp, "This is a long message")
	res := testMessage(bp, "tester says this is bad")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "tester never said 'this is bad'")
	}
}

func TestBabblerBadSeed2(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is a message")
	testMessage(bp, "This is another message")
	testMessage(bp, "This is a long message")
	res := testMessage(bp, "tester says This is a really")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "tester never said 'this is a really'")
	}
}

func TestBabblerSuffixSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	testMessage(bp, "tester says-tail message one")
	res := testMessage(bp, "tester says-tail with unique")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 2) {
		assert.Contains(t, mb.Messages[0], "this is message one")
		assert.Contains(t, mb.Messages[1], "it's easier to test with unique")
	}
}

func TestBabblerBadSuffixSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	res := testMessage(bp, "tester says-tail anything true")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "tester never said 'anything true'")
	}
}

func TestBabblerBookendSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	res := testMessage(bp, "tester says-bridge it's easier | unique messages")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "it's easier to test with unique messages")
	}
}

func TestBabblerBadBookendSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	res := testMessage(bp, "tester says-bridge says-bridge It's easier | not unique messages")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "tester never said 'it's easier ... not unique messages'")
	}
}

func TestBabblerMiddleOutSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	res := testMessage(bp, "tester says-middle-out test with")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "it's easier to test with unique messages")
	}
}

func TestBabblerBadMiddleOutSeed(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "This is message one")
	testMessage(bp, "It's easier to test with unique messages")
	res := testMessage(bp, "tester says-middle-out anything true")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "tester never said 'anything true'")
	}
}

func TestBabblerMerge(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)

	testMessage(bp, "<tester> This is a message")
	assert.Len(t, mb.Messages, 0)

	testMessage(bp, "<tester> This is another message")
	testMessage(bp, "<tester> This is a long message")
	res := testMessage(bp, "merge babbler tester into tester2")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 1) {
		assert.Contains(t, mb.Messages[0], "mooooiggged")
	}

	res = testMessage(bp, "!tester2 says")
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 2) {
		assert.Contains(t, mb.Messages[1], "<tester2> this is")
		assert.Contains(t, mb.Messages[1], "message")
	}
}

func TestHelp(t *testing.T) {
	mb := bot.NewMockBot()
	bp := newBabblerPlugin(mb)
	assert.NotNil(t, bp)
	c := &cli.CliPlugin{}
	bp.help(c, bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Len(t, mb.Messages, 1)
}
