package rss

import (
	"fmt"
	"github.com/velour/catbase/plugins/cli"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
)

func makeMessage(payload string) (bot.Connector, bot.Kind, msg.Message) {
	isCmd := strings.HasPrefix(payload, "!")
	if isCmd {
		payload = payload[1:]
	}
	return &cli.CliPlugin{}, bot.Message, msg.Message{
		User:    &user.User{Name: "tester"},
		Channel: "test",
		Body:    payload,
		Command: isCmd,
	}
}

func TestRSS(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	res := c.message(makeMessage("!rss http://rss.cnn.com/rss/edition.rss"))
	assert.Len(t, mb.Messages, 1)
	assert.True(t, res)
}

func TestRSSPaging(t *testing.T) {
	mb := bot.NewMockBot()
	c := New(mb)
	assert.NotNil(t, c)
	for i := 0; i < 20; i++ {
		res := c.message(makeMessage("!rss http://rss.cnn.com/rss/edition.rss"))
		assert.True(t, res)
	}

	assert.Len(t, mb.Messages, 20)

	for i := 0; i < len(mb.Messages); i++ {
		if i > 0 && strings.Contains(mb.Messages[i], "CNN.com - RSS Channel - Intl Homepage - News") {
			fmt.Println("----------------")
			fmt.Println(mb.Messages[i])
			fmt.Println("----------------")
			break
		}
		fmt.Println("----------------")
		fmt.Println(mb.Messages[i])
		fmt.Println("----------------")
	}
}
