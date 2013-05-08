package bot

import (
	"github.com/chrissexton/alepale/config"
	irc "github.com/fluffle/goirc/client"
	"labix.org/v2/mgo"
	"strings"
	"time"
)

// Bot type provides storage for bot-wide information, configs, and database connections
type Bot struct {
	// Each plugin must be registered in our plugins handler. To come: a map so that this
	// will allow plugins to respond to specific kinds of events
	Plugins        map[string]Handler
	PluginOrdering []string

	// Users holds information about all of our friends
	Users []User
	// Represents the bot
	Me User

	// Conn allows us to send messages and modify our connection state
	Conn *irc.Conn

	Config *config.Config

	// Mongo connection and db allow botwide access to the database
	DbSession *mgo.Session
	Db        *mgo.Database

	varColl *mgo.Collection

	logIn  chan Message
	logOut chan Messages

	Version string
}

// Log provides a slice of messages in order
type Log Messages
type Messages []Message

type Logger struct {
	in      <-chan Message
	out     chan<- Messages
	entries Messages
}

func NewLogger(in chan Message, out chan Messages) *Logger {
	return &Logger{in, out, make(Messages, 0)}
}

func RunNewLogger(in chan Message, out chan Messages) {
	logger := NewLogger(in, out)
	go logger.Run()
}

func (l *Logger) sendEntries() {
	l.out <- l.entries
}

func (l *Logger) Run() {
	var msg Message
	for {
		select {
		case msg = <-l.in:
			l.entries = append(l.entries, msg)
		case l.out <- l.entries:
			go l.sendEntries()
		}
	}
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

	Admin bool
}

type Message struct {
	User          *User
	Channel, Body string
	Raw           string
	Command       bool
	Action        bool
	Time          time.Time
	Host          string
}

type Variable struct {
	Variable, Value string
}

// NewBot creates a Bot for a given connection and set of handlers.
func NewBot(config *config.Config, c *irc.Conn) *Bot {
	session, err := mgo.Dial(config.DbServer)
	if err != nil {
		panic(err)
	}

	db := session.DB(config.DbName)

	logIn := make(chan Message)
	logOut := make(chan Messages)

	RunNewLogger(logIn, logOut)

	config.Nick = c.Me.Nick

	users := []User{
		User{
			Name:       config.Nick,
			MessageLog: make([]string, 0),
		},
	}

	return &Bot{
		Config:         config,
		Plugins:        make(map[string]Handler),
		PluginOrdering: make([]string, 0),
		Users:          users,
		Me:             users[0],
		Conn:           c,
		DbSession:      session,
		Db:             db,
		varColl:        db.C("variables"),
		logIn:          logIn,
		logOut:         logOut,
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

	// Notify plugins that we've said something
	b.selfSaid(channel, message)
}

// Sends action to channel
func (b *Bot) SendAction(channel, message string) {
	b.Conn.Action(channel, message)

	// Notify plugins that we've said something
	b.selfSaid(channel, message)
}

// Handles incomming PRIVMSG requests
func (b *Bot) MsgRecieved(conn *irc.Conn, line *irc.Line) {
	msg := b.buildMessage(conn, line)

	if strings.HasPrefix(msg.Body, "help") && msg.Command {
		parts := strings.Fields(strings.ToLower(msg.Body))
		b.checkHelp(msg.Channel, parts)
		goto RET
	}

	for _, name := range b.PluginOrdering {
		p := b.Plugins[name]
		if p.Message(msg) {
			break
		}
	}

RET:
	b.logIn <- msg
	return
}
