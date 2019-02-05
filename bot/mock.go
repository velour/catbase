// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type MockBot struct {
	mock.Mock
	db *sqlx.DB

	Cfg *config.Config

	Messages  []string
	Actions   []string
	Reactions []string
}

func (mb *MockBot) Config() *config.Config { return mb.Cfg }
func (mb *MockBot) DB() *sqlx.DB           { return mb.Cfg.DB }
func (mb *MockBot) Who(string) []user.User { return []user.User{} }
func (mb *MockBot) Send(kind Kind, args ...interface{}) (error, string) {
	switch kind {
	case Message:
		mb.Messages = append(mb.Messages, args[1].(string))
		return nil, fmt.Sprintf("m-%d", len(mb.Actions)-1)
	case Action:
		mb.Actions = append(mb.Actions, args[1].(string))
		return nil, fmt.Sprintf("a-%d", len(mb.Actions)-1)
	case Edit:
		ch, m, id := args[0].(string), args[1].(string), args[2].(string)
		return mb.edit(ch, m, id)
	case Reaction:
		ch, re, msg := args[0].(string), args[1].(string), args[2].(msg.Message)
		return mb.react(ch, re, msg)
	}
	return fmt.Errorf("Mesasge type unhandled"), "ERROR"
}
func (mb *MockBot) AddPlugin(name string, f Plugin)                         {}
func (mb *MockBot) Register(name string, kind Kind, cb Callback)            {}
func (mb *MockBot) Receive(kind Kind, msg msg.Message, args ...interface{}) {}
func (mb *MockBot) Filter(msg msg.Message, s string) string                 { return s }
func (mb *MockBot) LastMessage(ch string) (msg.Message, error)              { return msg.Message{}, nil }
func (mb *MockBot) CheckAdmin(nick string) bool                             { return false }

func (mb *MockBot) react(channel, reaction string, message msg.Message) (error, string) {
	mb.Reactions = append(mb.Reactions, reaction)
	return nil, ""
}

func (mb *MockBot) edit(channel, newMessage, identifier string) (error, string) {
	isMessage := identifier[0] == 'm'
	if !isMessage && identifier[0] != 'a' {
		err := fmt.Errorf("failed to parse identifier: %s", identifier)
		log.Println(err)
		return err, ""
	}

	index, err := strconv.Atoi(strings.Split(identifier, "-")[1])
	if err != nil {
		err := fmt.Errorf("failed to parse identifier: %s", identifier)
		log.Println(err)
		return err, ""
	}

	if isMessage {
		if index < len(mb.Messages) {
			mb.Messages[index] = newMessage
		} else {
			return fmt.Errorf("No message"), ""
		}
	} else {
		if index < len(mb.Actions) {
			mb.Actions[index] = newMessage
		} else {
			return fmt.Errorf("No action"), ""
		}
	}
	return nil, ""
}

func (mb *MockBot) GetEmojiList() map[string]string                { return make(map[string]string) }
func (mb *MockBot) RegisterFilter(s string, f func(string) string) {}

func NewMockBot() *MockBot {
	cfg := config.ReadConfig("file::memory:?mode=memory&cache=shared")
	b := MockBot{
		Cfg:      cfg,
		Messages: make([]string, 0),
		Actions:  make([]string, 0),
	}
	return &b
}
