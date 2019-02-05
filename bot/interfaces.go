// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

const (
	_ = iota

	// Message any standard chat
	Message
	// Reply something containing a message reference
	Reply
	// Action any /me action
	Action
	// Reaction Icon reaction if service supports it
	Reaction
	// Edit message ref'd new message to replace
	Edit
	// Not sure what event is
	Event
	// Help is used when the bot help system is triggered
	Help
	// SelfMessage triggers when the bot is sending a message
	SelfMessage
)

type kind int

type Callback func(int, msg.Message, ...interface{}) bool

type Bot interface {
	Config() *config.Config
	DB() *sqlx.DB
	Who(string) []user.User
	AddPlugin(string, Plugin)

	// First arg should be one of bot.Message/Reply/Action/etc
	Send(int, ...interface{}) (error, string)
	// First arg should be one of bot.Message/Reply/Action/etc
	Receive(int, msg.Message, ...interface{})
	// Register a callback
	Register(int, Callback)

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

	Send(int, ...interface{}) (error, string)

	GetEmojiList() map[string]string
	Serve() error

	Who(string) []string
}

// Interface used for compatibility with the Plugin interface
type Plugin interface {
	RegisterWeb() *string
}
