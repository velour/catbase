package bot

import irc "github.com/fluffle/goirc/client"
import "labix.org/v2/mgo"
import "bitbucket.org/phlyingpenguin/godeepintir/config"
import "strings"

// Bot type provides storage for bot-wide information, configs, and database connections
type Bot struct {
	// Each plugin must be registered in our plugins handler. To come: a map so that this
	// will allow plugins to respond to specific kinds of events
	Plugins        map[string]Handler
	PluginOrdering []string

	// Users holds information about all of our friends
	Users []User

	// Conn allows us to send messages and modify our connection state
	Conn *irc.Conn

	Config *config.Config

	// Mongo connection and db allow botwide access to the database
	DbSession *mgo.Session
	Db        *mgo.Database

	Version string
}

// User type stores user history. This is a vehicle that will follow the user for the active 
// session
type User struct {
	// Current nickname known
	Name string

	// LastSeen	DateTime

	// Alternative nicknames seen
	Alts []string

	// Last N messages sent to the user
	MessageLog []string
}

type Message struct {
	User          *User
	Channel, Body string
	Raw           string
	Command       bool
	Action        bool
}

// NewBot creates a Bot for a given connection and set of handlers.
func NewBot(config *config.Config, c *irc.Conn) *Bot {
	session, err := mgo.Dial(config.DbServer)
	if err != nil {
		panic(err)
	}

	db := session.DB(config.DbName)

	return &Bot{
		Config:         config,
		Plugins:        make(map[string]Handler),
		PluginOrdering: make([]string, 0),
		Users:          make([]User, 0),
		Conn:           c,
		DbSession:      session,
		Db:             db,
		Version:        config.Version,
	}
}

// Adds a constructed handler to the bots handlers list
func (b *Bot) AddHandler(name string, h Handler) {
	b.Plugins[strings.ToLower(name)] = h
	b.PluginOrdering = append(b.PluginOrdering, name)
}

// Sends message to channel
func (b *Bot) SendMessage(channel, message string) {
	b.Conn.Privmsg(channel, message)
}

// Sends action to channel
func (b *Bot) SendAction(channel, message string) {
	b.Conn.Action(channel, message)
}
