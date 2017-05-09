// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reminder

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestReminder(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser in 1s don't fail this test"))
	time.Sleep(2 * time.Second)
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "Hey testuser, tester wanted you to be reminded: don't fail this test")
}

func TestReminderReorder(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser in 2s don't fail this test 2"))
	assert.True(t, res)
	res = c.Message(makeMessage("!remind testuser in 1s don't fail this test 1"))
	assert.True(t, res)
	time.Sleep(5 * time.Second)
	assert.Len(t, mb.Messages, 4)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[2], "Hey testuser, tester wanted you to be reminded: don't fail this test 1")
	assert.Contains(t, mb.Messages[3], "Hey testuser, tester wanted you to be reminded: don't fail this test 2")
}

func TestReminderParse(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser in unparseable don't fail this test"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "Easy cowboy, not sure I can parse that duration.")
}

func TestEmptyList(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!list reminders"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "no pending reminders")
}

func TestList(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser in 5m don't fail this test 1"))
	assert.True(t, res)
	res = c.Message(makeMessage("!remind testuser in 5m don't fail this test 2"))
	assert.True(t, res)
	res = c.Message(makeMessage("!list reminders"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[2], "1) tester -> testuser :: don't fail this test 1 @ ")
	assert.Contains(t, mb.Messages[2], "2) tester -> testuser :: don't fail this test 2 @ ")
}

func TestBatch(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Reminder.MaxBatchAdd = 50
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser every 1s for 5s yikes"))
	assert.True(t, res)
	time.Sleep(6 * time.Second)
	assert.Len(t, mb.Messages, 6)
	for i := 0; i < 5; i++ {
		assert.Contains(t, mb.Messages[i+1], "Hey testuser, tester wanted you to be reminded: yikes")
	}
}

func TestBatchMax(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	c.config.Reminder.MaxBatchAdd = 10
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser every 1h for 24h yikes"))
	assert.True(t, res)
	res = c.Message(makeMessage("!list reminders"))
	assert.True(t, res)
	time.Sleep(6 * time.Second)
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[0], "Easy cowboy, that's a lot of reminders. I'll add some of them.")

	for i := 0; i < 10; i++ {
		assert.Contains(t, mb.Messages[1], fmt.Sprintf("%d) tester -> testuser :: yikes", i+1))
	}
}

func TestCancel(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!remind testuser in 1m don't fail this test"))
	assert.True(t, res)
	res = c.Message(makeMessage("!cancel reminder 0"))
	assert.True(t, res)
	res = c.Message(makeMessage("!list reminders"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "successfully canceled reminder: 0")
	assert.Contains(t, mb.Messages[2], "no pending reminders")
}

func TestCancelMiss(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.Message(makeMessage("!cancel reminder 0"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "failed to find and cancel reminder: 0")
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
