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

func setup(t *testing.T) (*bot.MockBot, *CounterPlugin) {
	mb := bot.NewMockBot()
	c := New(mb)
	mb.DB().MustExec(`delete from counter; delete from counter_alias;`)
	_, err := MkAlias(mb.DB(), "tea", ":tea:")
	assert.Nil(t, err)
	return mb, c
}

func makeMessage(payload string) (bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Message, msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func TestThreeSentencesExists(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage(":beer:++"))
	c.message(makeMessage(":beer:. Earl Grey. Hot."))
	item, err := GetItem(mb.DB(), "tester", ":beer:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestThreeSentencesNotExists(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	item, err := GetItem(mb.DB(), "tester", ":beer:")
	c.message(makeMessage(":beer:. Earl Grey. Hot."))
	item, err = GetItem(mb.DB(), "tester", ":beer:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaEarlGreyHot(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea. Earl Grey. Hot."))
	c.message(makeMessage("Tea. Earl Grey. Hot."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestTeaTwoPeriods(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea. Earl Grey."))
	c.message(makeMessage("Tea. Earl Grey."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaMultiplePeriods(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea. Earl Grey. Spiked. Hot."))
	c.message(makeMessage("Tea. Earl Grey. Spiked. Hot."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestTeaGreenHot(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea. Green. Hot."))
	c.message(makeMessage("Tea. Green. Hot"))
	c.message(makeMessage("Tea. Green. Iced."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 3, item.Count)
}

func TestTeaUnrelated(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea."))
	c.message(makeMessage("Tea. It's great."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaSkieselQuote(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("blah, this is a whole page of explanation where \"we did local search and used a tabu list\" would have sufficed"))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}
func TestTeaUnicodeJapanese(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("Tea. おちや. Hot."))
	item, err := GetItem(mb.DB(), "tester", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 1, item.Count)
}

func TestResetMe(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("test++"))
	c.message(makeMessage("!reset me"))
	items, err := GetItems(mb.DB(), "tester")
	assert.Nil(t, err)
	assert.Len(t, items, 0)
}

func TestCounterOne(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage("test++"))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, mb.Messages[0], "tester has 1 test.")
}

func TestCounterOneWithSpace(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.message(makeMessage(":test: ++"))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, mb.Messages[0], "tester has 1 :test:.")
}

func TestCounterFour(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
	}
	assert.Len(t, mb.Messages, 4)
	assert.Equal(t, mb.Messages[3], "tester has 4 test.")
}

func TestCounterDecrement(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	c.message(makeMessage("test--"))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "tester has 3 test.")
}

func TestFriendCounterDecrement(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("other.test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("other has %d test.", i+1))
	}
	c.message(makeMessage("other.test--"))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "other has 3 test.")
}

func TestDecrementZero(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	j := 4
	for i := 4; i > 0; i-- {
		c.message(makeMessage("test--"))
		assert.Equal(t, mb.Messages[j], fmt.Sprintf("tester has %d test.", i-1))
		j++
	}
	assert.Len(t, mb.Messages, 8)
	assert.Equal(t, mb.Messages[7], "tester has 0 test.")
}

func TestClear(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.message(makeMessage("!clear test"))
	assert.True(t, res)
	assert.Len(t, mb.Actions, 1)
	assert.Equal(t, mb.Actions[0], "chops a few test out of his brain")
}

func TestCount(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.message(makeMessage("!count test"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "tester has 4 test.")
}

func TestInspectMe(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.message(makeMessage("test++"))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	for i := 0; i < 2; i++ {
		c.message(makeMessage("fucks++"))
		assert.Equal(t, mb.Messages[i+4], fmt.Sprintf("tester has %d fucks.", i+1))
	}
	for i := 0; i < 20; i++ {
		c.message(makeMessage("cheese++"))
		assert.Equal(t, mb.Messages[i+6], fmt.Sprintf("tester has %d cheese.", i+1))
	}
	res := c.message(makeMessage("!inspect me"))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 27)
	assert.Equal(t, mb.Messages[26], "tester has the following counters: test: 4, fucks: 2, cheese: 20.")
}

func TestHelp(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.help(bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Len(t, mb.Messages, 1)
}

func TestRegisterWeb(t *testing.T) {
	_, c := setup(t)
	assert.NotNil(t, c)
	assert.Nil(t, c.RegisterWeb())
}
