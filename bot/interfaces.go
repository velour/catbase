// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"reflect"

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

type Kind int
type Callback func(Kind, msg.Message, ...interface{}) bool
type CallbackMap map[reflect.Type]map[Kind][]Callback

// Bot interface serves to allow mocking of the actual bot
type Bot interface {
	// Config allows access to the bot's configuration system
	Config() *config.Config
	// DB gives access to the current database
	DB() *sqlx.DB
	// Who lists users in a particular channel
	Who(string) []user.User
	// AddPlugin registers a new plugin handler
	AddPlugin(string, Plugin)
	// First arg should be one of bot.Message/Reply/Action/etc
	Send(Kind, ...interface{}) (string, error)
	// First arg should be one of bot.Message/Reply/Action/etc
	Receive(Kind, msg.Message, ...interface{})
	// Register a callback
	Register(Plugin, Kind, Callback)

	Filter(msg.Message, string) string
	LastMessage(string) (msg.Message, error)

	CheckAdmin(string) bool
	GetEmojiList() map[string]string
	RegisterFilter(string, func(string) string)
}

// Connector represents a server connection to a chat service
type Connector interface {
	RegisterEvent(func(Kind, msg.Message, ...interface{}))

	Send(Kind, ...interface{}) (string, error)

	GetEmojiList() map[string]string
	Serve() error

	Who(string) []string
}

// Plugin interface used for compatibility with the Plugin interface
// Probably can disappear once RegisterWeb gets inverted
type Plugin interface {
	RegisterWeb() *string
}
