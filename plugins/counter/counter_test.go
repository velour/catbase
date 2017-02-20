// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package counter

import (
	"fmt"
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

func TestTeaEarlGreyHot(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("Tea. Earl Grey. Hot."))
	c.Message(makeMessage("Tea. Earl Grey. Hot."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestTeaGreenHot(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("Tea. Green. Hot."))
	c.Message(makeMessage("Tea. Green. Hot"))
	c.Message(makeMessage("Tea. Green. Iced."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 3, item.Count)
}

func TestTeaUnrelated(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("Tea."))
	c.Message(makeMessage("Tea. It's great."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaSkieselQuote(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("blah, this is a whole page of explanation where \"we did local search and used a tabu list\" would have sufficed"))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestResetMe(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("test++"))
	c.Message(makeMessage("!reset me"))
	items, err := GetItems(mb.DB(), "tester")
	assert.Nil(t, err)
	assert.Len(t, items, 0)
}

func TestCounterOne(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	c.Message(makeMessage("test++"))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, mb.Messages[0], "tester has 1 test.")
}

func TestCounterFour(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
	}
	assert.Len(t, mb.Messages, 4)
	assert.Equal(t, mb.Messages[3], "tester has 4 test.")
}

func TestCounterDecrement(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	c.Message(makeMessage("test--"))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "tester has 3 test.")
}

func TestFriendCounterDecrement(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("other.test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("other has %d test.", i+1))
	}
	c.Message(makeMessage("other.test--"))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "other has 3 test.")
}

func TestDecrementZero(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	j := 4
	for i := 4; i > 0; i-- {
		c.Message(makeMessage("test--"))
		assert.Equal(t, mb.Messages[j], fmt.Sprintf("tester has %d test.", i-1))
		j++
	}
	assert.Len(t, mb.Messages, 8)
	assert.Equal(t, mb.Messages[7], "tester has 0 test.")
}

func TestClear(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.Message(makeMessage("!clear test"))
	assert.True(t, res)
	assert.Len(t, mb.Actions, 1)
	assert.Equal(t, mb.Actions[0], "chops a few test out of his brain")
}

func TestCount(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.Message(makeMessage("!count test"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "tester has 4 test.")
}

func TestInspectMe(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.Message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	for i := 0; i < 2; i++ {
		c.Message(makeMessage("fucks++"))
		assert.Equal(t, mb.Messages[i+4], fmt.Sprintf("tester has %d fucks.", i+1))
	}
	for i := 0; i < 20; i++ {
		c.Message(makeMessage("cheese++"))
		assert.Equal(t, mb.Messages[i+6], fmt.Sprintf("tester has %d cheese.", i+1))
	}
	res := c.Message(makeMessage("!inspect me"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 27)
	assert.Equal(t, mb.Messages[26], "tester has the following counters: test: 4, fucks: 2, cheese: 20.")
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
