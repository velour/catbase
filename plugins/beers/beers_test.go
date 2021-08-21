// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package beers

import (
	"regexp"
	"strings"
	"testing"

	"github.com/velour/catbase/plugins/cli"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/plugins/counter"
)

func makeMessage(payload string, r *regexp.Regexp) bot.Request {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	c := &cli.CliPlugin{}
	values := bot.ParseValues(r, payload)
	return bot.Request{
		Conn:   c,
		Kind:   bot.Message,
		Values: values,
		Msg: msg.Message{
			User:    &user.User{Name: "tester", ID: "id"},
			Channel: "test",
			Body:    payload,
			Command: isCmd,
		},
	}
}

func testMessage(p *BeersPlugin, msg string) bool {
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

func makeBeersPlugin(t *testing.T) (*BeersPlugin, *bot.MockBot) {
	mb := bot.NewMockBot()
	counter.New(mb)
	mb.DB().MustExec(`delete from counter; delete from counter_alias;`)
	b := New(mb)
	counter.MkAlias(mb.DB(), "beer", DEFAULT_ITEM)
	counter.MkAlias(mb.DB(), "beers", DEFAULT_ITEM)
	return b, mb
}

func TestCounter(t *testing.T) {
	_, mb := makeBeersPlugin(t)
	i, err := counter.GetUserItem(mb.DB(), "tester", "id", "test")
	if !assert.Nil(t, err) {
		t.Log(err)
		t.Fatal()
	}
	err = i.Update(nil, 5)
	assert.Nil(t, err)
}

func TestImbibe(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "imbibe")
	assert.Len(t, mb.Messages, 1)
	testMessage(b, "imbibe")
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 2, it.Count)
}
func TestEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "beers = 3")
	assert.Len(t, mb.Messages, 1)
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 3, it.Count)
}

func TestEqZero(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "beers += 5")
	testMessage(b, "beers = 0")
	assert.Len(t, mb.Messages, 2)
	assert.Contains(t, mb.Messages[1], "reversal of fortune")
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersPlusEq(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "beers += 5")
	assert.Len(t, mb.Messages, 1)
	testMessage(b, "beers += 5")
	assert.Len(t, mb.Messages, 2)
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 10, it.Count)
}

func TestPuke(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "beers += 5")
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 5, it.Count)
	testMessage(b, "puke")
	it, err = counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 0, it.Count)
}

func TestBeersReport(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	testMessage(b, "beers += 5")
	it, err := counter.GetUserItem(mb.DB(), "tester", "id", DEFAULT_ITEM)
	assert.Nil(t, err)
	assert.Equal(t, 5, it.Count)
	testMessage(b, "beers")
	if assert.Len(t, mb.Messages, 2) {
		assert.Contains(t, mb.Messages[1], "5 beers")
	}
}

func TestHelp(t *testing.T) {
	b, mb := makeBeersPlugin(t)
	b.help(&cli.CliPlugin{}, bot.Help, msg.Message{Channel: "channel"}, []string{})
	assert.Len(t, mb.Messages, 1)
}
