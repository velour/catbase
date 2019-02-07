// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"fmt"
	"log"
	"net/http"
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
func (mb *MockBot) Send(kind Kind, args ...interface{}) (string, error) {
	switch kind {
	case Message:
		mb.Messages = append(mb.Messages, args[1].(string))
		return fmt.Sprintf("m-%d", len(mb.Actions)-1), nil
	case Action:
		mb.Actions = append(mb.Actions, args[1].(string))
		return fmt.Sprintf("a-%d", len(mb.Actions)-1), nil
	case Edit:
		ch, m, id := args[0].(string), args[1].(string), args[2].(string)
		return mb.edit(ch, m, id)
	case Reaction:
		ch, re, msg := args[0].(string), args[1].(string), args[2].(msg.Message)
		return mb.react(ch, re, msg)
	}
	return "ERR", fmt.Errorf("Mesasge type unhandled")
}
func (mb *MockBot) AddPlugin(f Plugin)                                           {}
func (mb *MockBot) Register(p Plugin, kind Kind, cb Callback)                    {}
func (mb *MockBot) RegisterWeb(_, _ string)                                      {}
func (mb *MockBot) Receive(kind Kind, msg msg.Message, args ...interface{}) bool { return false }
func (mb *MockBot) Filter(msg msg.Message, s string) string                      { return s }
func (mb *MockBot) LastMessage(ch string) (msg.Message, error)                   { return msg.Message{}, nil }
func (mb *MockBot) CheckAdmin(nick string) bool                                  { return false }

func (mb *MockBot) react(channel, reaction string, message msg.Message) (string, error) {
	mb.Reactions = append(mb.Reactions, reaction)
	return "", nil
}

func (mb *MockBot) edit(channel, newMessage, identifier string) (string, error) {
	isMessage := identifier[0] == 'm'
	if !isMessage && identifier[0] != 'a' {
		err := fmt.Errorf("failed to parse identifier: %s", identifier)
		log.Println(err)
		return "", err
	}

	index, err := strconv.Atoi(strings.Split(identifier, "-")[1])
	if err != nil {
		err := fmt.Errorf("failed to parse identifier: %s", identifier)
		log.Println(err)
		return "", err
	}

	if isMessage {
		if index < len(mb.Messages) {
			mb.Messages[index] = newMessage
		} else {
			return "", fmt.Errorf("No message")
		}
	} else {
		if index < len(mb.Actions) {
			mb.Actions[index] = newMessage
		} else {
			return "", fmt.Errorf("No action")
		}
	}
	return "", nil
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
	// If any plugin registered a route, we need to reset those before any new test
	http.DefaultServeMux = new(http.ServeMux)
	return &b
}
