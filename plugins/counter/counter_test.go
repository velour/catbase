// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package counter

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/velour/catbase/plugins/cli"

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

func makeMessage(payload string, r *regexp.Regexp) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	values := bot.ParseValues(r, payload)
	return bot.Request{
		Conn: &cli.CliPlugin{},
		Msg: msg.Message{
			User:    &user.User{Name: "tester", ID: "id"},
			Body:    payload,
			Command: isCmd,
		},
		Values: values,
	}
}

func TestMkAlias(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.mkAliasCmd(makeMessage("mkalias fuck mornings", mkAliasRegex))
	c.incrementCmd(makeMessage("fuck++", incrementRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", "mornings")
	assert.Nil(t, err)
	assert.Equal(t, 1, item.Count)
}

func TestRmAlias(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.mkAliasCmd(makeMessage("mkalias fuck mornings", mkAliasRegex))
	c.rmAliasCmd(makeMessage("rmalias fuck", rmAliasRegex))
	c.incrementCmd(makeMessage("fuck++", incrementRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", "mornings")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestThreeSentencesExists(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.incrementCmd(makeMessage(":beer:++", incrementRegex))
	c.teaMatchCmd(makeMessage(":beer:. Earl Grey. Hot.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":beer:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestThreeSentencesNotExists(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	item, err := GetUserItem(mb.DB(), "tester", "id", ":beer:")
	c.teaMatchCmd(makeMessage(":beer:. Earl Grey. Hot.", teaRegex))
	item, err = GetUserItem(mb.DB(), "tester", "id", ":beer:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaEarlGreyHot(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea. Earl Grey. Hot.", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. Earl Grey. Hot.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestTeaTwoPeriods(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea. Earl Grey.", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. Earl Grey.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaMultiplePeriods(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea. Earl Grey. Spiked. Hot.", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. Earl Grey. Spiked. Hot.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 2, item.Count)
}

func TestTeaGreenHot(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea. Green. Hot.", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. Green. Hot", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. Green. Iced.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 3, item.Count)
}

func TestTeaUnrelated(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea.", teaRegex))
	c.teaMatchCmd(makeMessage("Tea. It's great.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}

func TestTeaSkieselQuote(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("blah, this is a whole page of explanation where \"we did local search and used a tabu list\" would have sufficed", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 0, item.Count)
}
func TestTeaUnicodeJapanese(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.teaMatchCmd(makeMessage("Tea. おちや. Hot.", teaRegex))
	item, err := GetUserItem(mb.DB(), "tester", "id", ":tea:")
	assert.Nil(t, err)
	assert.Equal(t, 1, item.Count)
}

func TestResetMe(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.incrementCmd(makeMessage("test++", incrementRegex))
	c.resetCmd(makeMessage("!reset me", resetRegex))
	items, err := GetItems(mb.DB(), "tester", "id")
	assert.Nil(t, err)
	assert.Len(t, items, 0)
}

func TestCounterOne(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.incrementCmd(makeMessage("test++", incrementRegex))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, mb.Messages[0], "tester has 1 test.")
}

func TestCounterOneWithSpace(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.incrementCmd(makeMessage(":test: ++", incrementRegex))
	assert.Len(t, mb.Messages, 1)
	assert.Equal(t, mb.Messages[0], "tester has 1 :test:.")
}

func TestCounterFour(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("test++", incrementRegex))
	}
	assert.Len(t, mb.Messages, 4)
	assert.Equal(t, mb.Messages[3], "tester has 4 test.")
}

func TestCounterDecrement(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	c.decrementCmd(makeMessage("test--", decrementRegex))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "tester has 3 test.")
}

func TestFriendCounterDecrement(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("other.test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("other has %d test.", i+1))
	}
	c.decrementCmd(makeMessage("other.test--", decrementRegex))
	assert.Len(t, mb.Messages, 5)
	assert.Equal(t, mb.Messages[4], "other has 3 test.")
}

func TestDecrementZero(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	j := 4
	for i := 4; i > 0; i-- {
		c.decrementCmd(makeMessage("test--", decrementRegex))
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
		c.incrementCmd(makeMessage("test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.clearCmd(makeMessage("!clear test", clearRegex))
	assert.True(t, res)
	assert.Len(t, mb.Actions, 1)
	assert.Equal(t, mb.Actions[0], "chops a few test out of his brain")
}

func TestCount(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	res := c.countCmd(makeMessage("!count test", countRegex))
	assert.True(t, res)
	if assert.Len(t, mb.Messages, 5) {
		assert.Equal(t, "tester has 4 test.", mb.Messages[4])
	}
}

func TestInspectMe(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	for i := 0; i < 4; i++ {
		c.incrementCmd(makeMessage("test++", incrementRegex))
		assert.Equal(t, mb.Messages[i], fmt.Sprintf("tester has %d test.", i+1))
	}
	for i := 0; i < 2; i++ {
		c.incrementCmd(makeMessage("fucks++", incrementRegex))
		assert.Equal(t, mb.Messages[i+4], fmt.Sprintf("tester has %d fucks.", i+1))
	}
	for i := 0; i < 20; i++ {
		c.incrementCmd(makeMessage("cheese++", incrementRegex))
		assert.Equal(t, mb.Messages[i+6], fmt.Sprintf("tester has %d cheese.", i+1))
	}
	res := c.inspectCmd(makeMessage("!inspect me", inspectRegex))
	assert.True(t, res)
	assert.Len(t, mb.Messages, 27)
	assert.Equal(t, mb.Messages[26], "tester has the following counters: test: 4, fucks: 2, cheese: 20.")
}

func TestHelp(t *testing.T) {
	mb, c := setup(t)
	assert.NotNil(t, c)
	c.help(&cli.CliPlugin{}, bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Greater(t, len(mb.Messages), 1)
}
