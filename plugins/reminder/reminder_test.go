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

func makeMessage(payload string) (bot.Kind, msg.Message) {
	return makeMessageBy(payload, "tester")
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

func setup(t *testing.T) (*ReminderPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	r := New(mb)
	mb.DB().MustExec(`delete from reminders; delete from config;`)
	return r, mb
}

func TestMeReminder(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!remind me in 1s don't fail this test"))
	time.Sleep(2 * time.Second)
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "Okay. I'll remind you.")
	assert.Contains(t, mb.Messages[1], "Hey tester, you wanted to be reminded: don't fail this test")
}

func TestReminder(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!remind testuser in 1s don't fail this test"))
	time.Sleep(2 * time.Second)
	assert.Len(t, mb.Messages, 2)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "Hey testuser, tester wanted you to be reminded: don't fail this test")
}

func TestReminderReorder(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!remind testuser in 2s don't fail this test 2"))
	assert.True(t, res)
	res = c.message(makeMessage("!remind testuser in 1s don't fail this test 1"))
	assert.True(t, res)
	time.Sleep(5 * time.Second)
	assert.Len(t, mb.Messages, 4)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[2], "Hey testuser, tester wanted you to be reminded: don't fail this test 1")
	assert.Contains(t, mb.Messages[3], "Hey testuser, tester wanted you to be reminded: don't fail this test 2")
}

func TestReminderParse(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!remind testuser in unparseable don't fail this test"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "Easy cowboy, not sure I can parse that duration.")
}

func TestEmptyList(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!list reminders"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
	assert.Contains(t, mb.Messages[0], "no pending reminders")
}

func TestList(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessage("!remind testuser in 5m don't fail this test 1"))
	assert.True(t, res)
	res = c.message(makeMessage("!remind testuser in 5m don't fail this test 2"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[2], "1) tester -> testuser :: don't fail this test 1 @ ")
	assert.Contains(t, mb.Messages[2], "2) tester -> testuser :: don't fail this test 2 @ ")
}

func TestListBy(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessageBy("!remind testuser in 5m don't fail this test 1", "testuser"))
	assert.True(t, res)
	res = c.message(makeMessageBy("!remind testuser in 5m don't fail this test 2", "testuser2"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders from testuser"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[2], "don't fail this test 1 @ ")
	assert.NotContains(t, mb.Messages[2], "don't fail this test 2 @ ")
}

func TestListTo(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessageBy("!remind testuser2 in 5m don't fail this test 1", "testuser"))
	assert.True(t, res)
	res = c.message(makeMessageBy("!remind testuser in 5m don't fail this test 2", "testuser2"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders to testuser"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.NotContains(t, mb.Messages[2], "don't fail this test 1 @ ")
	assert.Contains(t, mb.Messages[2], "don't fail this test 2 @ ")
}

func TestToEmptyList(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessageBy("!remind testuser2 in 5m don't fail this test 1", "testuser"))
	assert.True(t, res)
	res = c.message(makeMessageBy("!remind testuser in 5m don't fail this test 2", "testuser2"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders to test"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[2], "no pending reminders")
}

func TestFromEmptyList(t *testing.T) {
	c, mb := setup(t)
	res := c.message(makeMessageBy("!remind testuser2 in 5m don't fail this test 1", "testuser"))
	assert.True(t, res)
	res = c.message(makeMessageBy("!remind testuser in 5m don't fail this test 2", "testuser2"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders from test"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[2], "no pending reminders")
}

func TestBatchMax(t *testing.T) {
	c, mb := setup(t)
	c.config.Set("Reminder.MaxBatchAdd", "10")
	assert.NotNil(t, c)
	res := c.message(makeMessage("!remind testuser every 1h for 24h yikes"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders"))
	assert.True(t, res)
	time.Sleep(6 * time.Second)
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[0], "Easy cowboy, that's a lot of reminders. I'll add some of them.")

	for i := 0; i < 10; i++ {
		assert.Contains(t, mb.Messages[1], fmt.Sprintf("%d) tester -> testuser :: yikes", i+1))
	}
}

func TestCancel(t *testing.T) {
	c, mb := setup(t)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!remind testuser in 1m don't fail this test"))
	assert.True(t, res)
	res = c.message(makeMessage("!cancel reminder 1"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 3)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "successfully canceled reminder: 1")
	assert.Contains(t, mb.Messages[2], "no pending reminders")
}

func TestCancelMiss(t *testing.T) {
	c, mb := setup(t)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!cancel reminder 1"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 1)
	assert.Contains(t, mb.Messages[0], "failed to find and cancel reminder: 1")
}

func TestLimitList(t *testing.T) {
	c, mb := setup(t)
	c.config.Set("Reminder.MaxBatchAdd", "10")
	c.config.Set("Reminder.MaxList", "25")
	assert.NotNil(t, c)

	//Someone can redo this with a single batch add, but I can't locally due to an old version of sqllite (maybe).
	res := c.message(makeMessage("!remind testuser every 1h for 10h don't fail this test"))
	assert.True(t, res)
	res = c.message(makeMessage("!remind testuser every 1h for 10h don't fail this test"))
	assert.True(t, res)
	res = c.message(makeMessage("!remind testuser every 1h for 10h don't fail this test"))
	assert.True(t, res)
	res = c.message(makeMessage("!list reminders"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 4)
	assert.Contains(t, mb.Messages[0], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[1], "Sure tester, I'll remind testuser.")
	assert.Contains(t, mb.Messages[2], "Sure tester, I'll remind testuser.")

	for i := 0; i < 25; i++ {
		assert.Contains(t, mb.Messages[3], fmt.Sprintf("%d) tester -> testuser :: don't fail this test", i+1))
	}
	assert.Contains(t, mb.Messages[3], "more...")

	assert.NotContains(t, mb.Messages[3], "26) tester -> testuser")
}

func TestHelp(t *testing.T) {
	c, mb := setup(t)
	assert.NotNil(t, c)
	c.help(bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Len(t, mb.Messages, 1)
}

func TestRegisterWeb(t *testing.T) {
	c, _ := setup(t)
	assert.NotNil(t, c)
	assert.Nil(t, c.RegisterWeb())
}
