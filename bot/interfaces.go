// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

type Connector interface {
	RegisterEventReceived(func(message Message))
	RegisterMessageReceived(func(message Message))

	SendMessage(channel, message string)
	SendAction(channel, message string)
	Serve()
}

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(message Message) bool
	Event(kind string, message Message) bool
	BotMessage(message Message) bool
	Help(channel string, parts []string)
	RegisterWeb() *string
}
