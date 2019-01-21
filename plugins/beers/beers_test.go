// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package beers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
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

func makeBeersPlugin(t *testing.T) (*BeersPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	mb.DB().MustExec(`delete from counter; delete from counter_alias;`)
	b := New(mb)
	b.Message(makeMessage("!mkalias beer :beer:"))
	b.Message(makeMessage("!mkalias beers :beer:"))
	return b, mb
}

func TestCounter(t *testing.T) {
	_, mb := makeBeersPlugin(t)
	i, err := counter.GetItem(mb.DB(), "tester", "test")
	if !assert.Nil(t, err) {
		t.Log(err)
		t.Fatal()
	}
	err = i.Update(5)
	assert.Nil(t, err)
}

func TestImbibe(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!imbibe"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("!imbibe"))
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 2, it.Count)
}
func TestEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!beers = 3"))
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 3, it.Count)
}

func TestEqNeg(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!beers = -3"))
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestEqZero(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	b.Message(makeMessage("!beers = 0"))
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[1], "reversal of fortune")
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersPlusEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("beers += 5"))
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 10, it.Count)
}

func TestPuke(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 5, it.Count)
	b.Message(makeMessage("puke"))
	it, err = counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersReport(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	it, err := counter.GetItem(mb.DB(), "tester", itemName)
	assert.Nil(t, err)
	assert.Equal(t, 5, it.Count)
	b.Message(makeMessage("beers"))
	assert.Contains(t, mb.Messages[1], "5 beers")
}

func TestHelp(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Help("channel", []string{})
	assert.Len(t, mb.Messages, 1)
}

func TestBotMessage(t *testing.T) {
	b, _ := makeBeersPlugin(t)
	assert.False(t, b.BotMessage(makeMessage("test")))
}

func TestEvent(t *testing.T) {
	b, _ := makeBeersPlugin(t)
	assert.False(t, b.Event("dummy", makeMessage("test")))
}

func TestRegisterWeb(t *testing.T) {
	b, _ := makeBeersPlugin(t)
	assert.Nil(t, b.RegisterWeb())
}
