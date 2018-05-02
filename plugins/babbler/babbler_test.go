// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package babbler

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

func TestBabblerNoBabbler(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	c.Message(makeMessage("!seabass2 says"))
	res := assert.Len(t, mb.Messages, 0)
	assert.True(t, res)
	// assert.Contains(t, mb.Messages[0], "seabass2 babbler not found")
}

func TestBabblerNothingSaid(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	res := c.Message(makeMessage("initialize babbler for seabass"))
	assert.True(t, res)
	res = c.Message(makeMessage("!seabass says"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[0], "okay.")
	assert.Contains(t, mb.Messages[1], "seabass hasn't said anything yet.")
}

func TestBabbler(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "This is another message"
	res = c.Message(seabass)
	seabass.Body = "This is a long message"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "this is")
	assert.Contains(t, mb.Messages[0], "message")
}

func TestBabblerSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "This is another message"
	res = c.Message(seabass)
	seabass.Body = "This is a long message"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says long"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "long message")
}

func TestBabblerMultiSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "This is another message"
	res = c.Message(seabass)
	seabass.Body = "This is a long message"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says This is a long"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "this is a long message")
}

func TestBabblerMultiSeed2(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "This is another message"
	res = c.Message(seabass)
	seabass.Body = "This is a long message"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says is a long"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "is a long message")
}

func TestBabblerBadSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	c.Message(seabass)
	seabass.Body = "This is another message"
	c.Message(seabass)
	seabass.Body = "This is a long message"
	c.Message(seabass)
	c.Message(makeMessage("!seabass says noooo this is bad"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "seabass never said 'noooo this is bad'")
}

func TestBabblerBadSeed2(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is a message")
	seabass.User = &user.User{Name: "seabass"}
	c.Message(seabass)
	seabass.Body = "This is another message"
	c.Message(seabass)
	seabass.Body = "This is a long message"
	c.Message(seabass)
	c.Message(makeMessage("!seabass says This is a really"))
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "seabass never said 'this is a really'")
}

func TestBabblerSuffixSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is message one")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "It's easier to test with unique messages"
	res = c.Message(seabass)
	seabass.Body = "hi there"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-tail message one"))
	res = c.Message(makeMessage("!seabass says-tail with unique"))
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "this is message one")
	assert.Contains(t, mb.Messages[1], "it's easier to test with unique")
}

func TestBabblerBadSuffixSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("This is message one")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	seabass.Body = "It's easier to test with unique messages"
	res = c.Message(seabass)
	seabass.Body = "hi there"
	res = c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-tail anything true"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "seabass never said 'anything true'")
}

func TestBabblerBookendSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("It's easier to test with unique messages")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-bridge It's easier | unique messages"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "it's easier to test with unique messages")
}

func TestBabblerBookendSeedShort(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("It's easier to test with unique messages")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-bridge It's easier to test with | unique messages"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "it's easier to test with unique messages")
}

func TestBabblerBadBookendSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("It's easier to test with unique messages")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-bridge It's easier | not unique messages"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "seabass never said 'it's easier ... not unique messages'")
}

func TestBabblerMiddleOutSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("It's easier to test with unique messages")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-middle-out test with"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "it's easier to test with unique messages")
}

func TestBabblerBadMiddleOutSeed(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("It's easier to test with unique messages")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	res = c.Message(makeMessage("!seabass says-middle-out anything true"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Equal(t, mb.Messages[0], "seabass never said 'anything true'")
}

func TestBabblerBatch(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)
	seabass := makeMessage("batch learn for seabass This is a message! This is another message. This is not a long message? This is not a message! This is not another message. This is a long message?")
	res := c.Message(seabass)
	assert.Len(t, mb.Messages, 1)
	res = c.Message(makeMessage("!seabass says"))
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[1], "this is")
	assert.Contains(t, mb.Messages[1], "message")
}

func TestBabblerMerge(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Babbler.DefaultUsers = []string{"seabass"}
	assert.NotNil(t, c)

	seabass := makeMessage("<seabass> This is a message")
	seabass.User = &user.User{Name: "seabass"}
	res := c.Message(seabass)
	assert.Len(t, mb.Messages, 0)

	seabass.Body = "<seabass> This is another message"
	res = c.Message(seabass)

	seabass.Body = "<seabass> This is a long message"
	res = c.Message(seabass)

	res = c.Message(makeMessage("!merge babbler seabass into seabass2"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "mooooiggged")

	res = c.Message(makeMessage("!seabass2 says"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 2)

	assert.Contains(t, mb.Messages[1], "<seabass2> this is")
	assert.Contains(t, mb.Messages[1], "message")
}

func TestHelp(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Help("channel", []string{})
	assert.Len(t, mb.Messages, 1)
}

func TestBotMessage(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.False(t, c.BotMessage(makeMessage("test")))
}

func TestEvent(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.False(t, c.Event("dummy", makeMessage("test")))
}

func TestRegisterWeb(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	assert.Nil(t, c.RegisterWeb())
}
