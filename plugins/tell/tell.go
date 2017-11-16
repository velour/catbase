package tell

import (
	"fmt"
	"log"
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
	return &TellPlugin{b, make(map[string][]string)}
}

func (t *TellPlugin) Message(message msg.Message) bool {
	if strings.HasPrefix(message.Body, "tell") {
		parts := strings.Split(message.Body, " ")
		target := parts[1]
		newMessage := strings.Join(parts[2:], " ")
		newMessage = fmt.Sprintf("Hey, %s. %s said: %s", target, message.User.Name, newMessage)
		t.users[target] = append(t.users[target], newMessage)
		t.b.SendMessage(message.Channel, fmt.Sprintf("Okay. I'll tell %s.", target))
		return true
	}
	log.Printf("current pending tells: %+v\nuser is: %s", t.users, message.User.Name)
	if msg, ok := t.users[message.User.Name]; ok {
		for _, m := range msg {
			t.b.SendMessage(message.Channel, string(m))
		}
		t.users[message.User.Name] = []string{}
		return true
	}
	return false
}

func (t *TellPlugin) Event(kind string, message msg.Message) bool { return false }
func (t *TellPlugin) ReplyMessage(msg.Message, string) bool       { return false }
func (t *TellPlugin) BotMessage(message msg.Message) bool         { return false }
func (t *TellPlugin) Help(channel string, parts []string)         {}
func (t *TellPlugin) RegisterWeb() *string                        { return nil }
