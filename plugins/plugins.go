// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import "github.com/velour/catbase/bot/msg"

// Plugin interface defines the methods needed to accept a plugin
type Plugin interface {
	Message(message msg.Message) bool
	Event(kind string, message msg.Message) bool
	BotMessage(message msg.Message) bool
	LoadData()
	Help()
	RegisterWeb()
}
