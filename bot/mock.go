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

	Cfg config.Config

	Messages []string
	Actions  []string
}

func (mb *MockBot) Config() *config.Config            { return &mb.Cfg }
func (mb *MockBot) DBVersion() int64                  { return 1 }
func (mb *MockBot) DB() *sqlx.DB                      { return mb.db }
func (mb *MockBot) Conn() Connector                   { return nil }
func (mb *MockBot) Who(string) []user.User            { return []user.User{} }
func (mb *MockBot) AddHandler(name string, f Handler) {}
func (mb *MockBot) SendMessage(ch string, msg string) string {
	mb.Messages = append(mb.Messages, msg)
	return fmt.Sprintf("m-%d", len(mb.Actions)-1)
}
func (mb *MockBot) SendAction(ch string, msg string) string {
	mb.Actions = append(mb.Actions, msg)
	return fmt.Sprintf("a-%d", len(mb.Actions)-1)
}
func (mb *MockBot) ReplyToMessageIdentifier(channel, message, identifier string) (string, bool) { return "", false }
func (mb *MockBot) ReplyToMessage(channel, message string, replyTo msg.Message) (string, bool) { return "", false }
func (mb *MockBot) MsgReceived(msg msg.Message)                {}
func (mb *MockBot) EventReceived(msg msg.Message)              {}
func (mb *MockBot) Filter(msg msg.Message, s string) string    { return "" }
func (mb *MockBot) LastMessage(ch string) (msg.Message, error) { return msg.Message{}, nil }
func (mb *MockBot) CheckAdmin(nick string) bool                { return false }

func (mb *MockBot) React(channel, reaction string, message msg.Message) bool { return false }

func (mb *MockBot) Edit(channel, newMessage, identifier string) bool {
	isMessage := identifier[0] == 'm'
	if !isMessage && identifier[0] != 'a' {
		log.Printf("failed to parse identifier: %s", identifier)
		return false
	}

	index, err := strconv.Atoi(strings.Split(identifier, "-")[1])
	if err != nil {
		log.Printf("failed to parse identifier: %s", identifier)
		return false
	}

	if isMessage {
		if index < len(mb.Messages) {
			mb.Messages[index] = newMessage
		} else {
			return false
		}
	} else {
		if index < len(mb.Actions) {
			mb.Actions[index] = newMessage
		} else {
			return false
		}
	}
	return true
}
func (mb *MockBot) GetEmojiList() map[string]string                { return make(map[string]string) }
func (mb *MockBot) RegisterFilter(s string, f func(string) string) {}

func NewMockBot() *MockBot {
	db, err := sqlx.Open("sqlite3_custom", ":memory:")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	b := MockBot{
		db:       db,
		Messages: make([]string, 0),
		Actions:  make([]string, 0),
	}
	return &b
}
