// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/velour/catbase/config"
)

type MockBot struct {
	mock.Mock
	db *sqlx.DB

	Messages []string
	Actions  []string
}

func (mb *MockBot) Config() *config.Config            { return &config.Config{} }
func (mb *MockBot) DBVersion() int64                  { return 1 }
func (mb *MockBot) DB() *sqlx.DB                      { return mb.db }
func (mb *MockBot) Who(string) []User                 { return []User{} }
func (mb *MockBot) AddHandler(name string, f Handler) {}
func (mb *MockBot) SendMessage(ch string, msg string) {
	mb.Messages = append(mb.Messages, msg)
}
func (mb *MockBot) SendAction(ch string, msg string) {
	mb.Actions = append(mb.Actions, msg)
}
func (mb *MockBot) MsgReceived(msg Message)                {}
func (mb *MockBot) EventReceived(msg Message)              {}
func (mb *MockBot) Filter(msg Message, s string) string    { return "" }
func (mb *MockBot) LastMessage(ch string) (Message, error) { return Message{}, nil }

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
