// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

type Bot interface {
	Config() *config.Config
	DB() *sqlx.DB
	Who(string) []user.User
	AddHandler(string, Handler)
	SendMessage(string, string) string
	SendAction(string, string) string
	ReplyToMessageIdentifier(string, string, string) (string, bool)
	ReplyToMessage(string, string, msg.Message) (string, bool)
	React(string, string, msg.Message) bool
	Edit(string, string, string) bool
	MsgReceived(msg.Message)
	ReplyMsgReceived(msg.Message, string)
	EventReceived(msg.Message)
	Filter(msg.Message, string) string
	LastMessage(string) (msg.Message, error)
	CheckAdmin(string) bool
	GetEmojiList() map[string]string
	RegisterFilter(string, func(string) string)
}

type Connector interface {
	RegisterEventReceived(func(message msg.Message))
	RegisterMessageReceived(func(message msg.Message))
	RegisterReplyMessageReceived(func(msg.Message, string))

	SendMessage(channel, message string) string
	SendAction(channel, message string) string
	ReplyToMessageIdentifier(string, string, string) (string, bool)
	ReplyToMessage(string, string, msg.Message) (string, bool)
	React(string, string, msg.Message) bool
	Edit(string, string, string) bool
	GetEmojiList() map[string]string
	Serve() error

	Who(string) []string
}

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(message msg.Message) bool
	Event(kind string, message msg.Message) bool
	ReplyMessage(msg.Message, string) bool
	BotMessage(message msg.Message) bool
	Help(channel string, parts []string)
	RegisterWeb() *string
}
