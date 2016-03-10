package bot

type Connector interface {
	RegisterEventRecieved(func(message Message))
	RegisterMessageRecieved(func(message Message))

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
