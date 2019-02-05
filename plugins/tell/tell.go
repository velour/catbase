package tell

import (
	"fmt"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type delayedMsg string

type TellPlugin struct {
	b     bot.Bot
	users map[string][]string
}

func New(b bot.Bot) *TellPlugin {
	tp := &TellPlugin{b, make(map[string][]string)}
	b.Register(tp, bot.Message, tp.message)
	return tp
}

func (t *TellPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if strings.HasPrefix(strings.ToLower(message.Body), "tell") {
		parts := strings.Split(message.Body, " ")
		target := strings.ToLower(parts[1])
		newMessage := strings.Join(parts[2:], " ")
		newMessage = fmt.Sprintf("Hey, %s. %s said: %s", target, message.User.Name, newMessage)
		t.users[target] = append(t.users[target], newMessage)
		t.b.Send(bot.Message, message.Channel, fmt.Sprintf("Okay. I'll tell %s.", target))
		return true
	}
	uname := strings.ToLower(message.User.Name)
	if msg, ok := t.users[uname]; ok && len(msg) > 0 {
		for _, m := range msg {
			t.b.Send(bot.Message, message.Channel, string(m))
		}
		t.users[uname] = []string{}
		return true
	}
	return false
}

func (t *TellPlugin) RegisterWeb() *string { return nil }
