// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package plugins

import "github.com/velour/catbase/bot"

// Plugin interface defines the methods needed to accept a plugin
type Plugin interface {
	Message(message bot.Message) bool
	Event(kind string, message bot.Message) bool
	BotMessage(message bot.Message) bool
	LoadData()
	Help()
	RegisterWeb()
}
