package bot

import irc "github.com/fluffle/goirc/client"

// Bot type provides storage for bot-wide information, configs, and database connections
type Bot struct {
	Plugins []Handler
	Users []User
	Conn *irc.Conn
	// mongodb connection will go here
}

// User type stores user history. This is a vehicle that will follow the user for the active 
// session
type User struct {
	// Current nickname known
	Name	 string

	// LastSeen	DateTime

	// Alternative nicknames seen
	Alts	[]string

	// Last N messages sent to the user
	MessageLog	[]string
}

// NewBot creates a Bot for a given connection and set of handlers. The handlers must not
// require the bot as input for their creation (so use AddHandler instead to add handlers)
func NewBot(c *irc.Conn, p ...Handler) *Bot {
	return &Bot{
		Plugins: p,
		Users: make([]User, 10),
		Conn: c,
	}
}

// Adds a constructed handler to the bots handlers list
func (b *Bot) AddHandler(h Handler) {
	b.Plugins = append(b.Plugins, h)
}