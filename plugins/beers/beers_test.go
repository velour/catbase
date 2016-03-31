// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package beers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/plugins/counter"
)

func makeMessage(payload string) bot.Message {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return bot.Message{
		User:    &bot.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func makeBeersPlugin(t *testing.T) (*BeersPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	b := New(mb)
	assert.NotNil(t, b)
	return b, mb
}

func TestBeersPlusPlus(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers++"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("beers++"))
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 2, it.Count)
}

func TestBeersMinusMinus(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers = 5"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("beers--"))
	assert.Len(t, mb.Actions, 1)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 4, it.Count)
}

func TestImbibe(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!imbibe"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("!imbibe"))
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 2, it.Count)
}

func TestBourbon(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("bourbon++"))
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 2, it.Count)
}

func TestEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!beers = 3"))
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 3, it.Count)
}

func TestEqNeg(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("!beers = -3"))
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestEqZero(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	b.Message(makeMessage("!beers = 0"))
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[1], "reversal of fortune")
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersPlusEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	assert.Len(t, mb.Messages, 1)
	b.Message(makeMessage("beers += 5"))
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 10, it.Count)
}

func TestPuke(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 5, it.Count)
	b.Message(makeMessage("puke"))
	it, err = counter.GetItem(mb.DB(), "tester", "booze")
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersReport(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.Message(makeMessage("beers += 5"))
	it, err := counter.GetItem(mb.DB(), "tester", "booze")
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
