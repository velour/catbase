// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/velour/catbase/config"

	_ "github.com/mattn/go-sqlite3"
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

	Config *config.Config

	Conn Connector

	// SQL DB
	// TODO: I think it'd be nice to use https://github.com/jmoiron/sqlx so that
	//       the select/update/etc statements could be simplified with struct
	//       marshalling.
	DB        *sql.DB
	DBVersion int64

	logIn  chan Message
	logOut chan Messages

	Version string

	// The entries to the bot's HTTP interface
	httpEndPoints map[string]string
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
func NewBot(config *config.Config, connector Connector) *Bot {
	sqlDB, err := sql.Open("sqlite3", config.DB.File)
	if err != nil {
		log.Fatal(err)
	}

	logIn := make(chan Message)
	logOut := make(chan Messages)

	RunNewLogger(logIn, logOut)

	users := []User{
		User{
			Name: config.Nick,
		},
	}

	bot := &Bot{
		Config:         config,
		Plugins:        make(map[string]Handler),
		PluginOrdering: make([]string, 0),
		Conn:           connector,
		Users:          users,
		Me:             users[0],
		DB:             sqlDB,
		logIn:          logIn,
		logOut:         logOut,
		Version:        config.Version,
		httpEndPoints:  make(map[string]string),
	}

	bot.migrateDB()

	http.HandleFunc("/", bot.serveRoot)
	if config.HttpAddr == "" {
		config.HttpAddr = "127.0.0.1:1337"
	}
	go http.ListenAndServe(config.HttpAddr, nil)

	connector.RegisterMessageRecieved(bot.MsgRecieved)
	connector.RegisterEventRecieved(bot.EventRecieved)

	return bot
}

// Create any tables if necessary based on version of DB
// Plugins should create their own tables, these are only for official bot stuff
// Note: This does not return an error. Database issues are all fatal at this stage.
func (b *Bot) migrateDB() {
	_, err := b.DB.Exec(`create table if not exists version (version integer);`)
	if err != nil {
		log.Fatal("Initial DB migration create version table: ", err)
	}
	var version sql.NullInt64
	err = b.DB.QueryRow("select max(version) from version").Scan(&version)
	if err != nil {
		log.Fatal("Initial DB migration get version: ", err)
	}
	if version.Valid {
		b.DBVersion = version.Int64
		log.Printf("Database version: %v\n", b.DBVersion)
	} else {
		log.Printf("No versions, we're the first!.")
		_, err := b.DB.Exec(`insert into version (version) values (1)`)
		if err != nil {
			log.Fatal("Initial DB migration insert: ", err)
		}
	}

	if b.DBVersion == 1 {
		if _, err := b.DB.Exec(`create table if not exists variables (
			id integer primary key,
			name string,
			perms string,
			type string
		);`); err != nil {
			log.Fatal("Initial DB migration create variables table: ", err)
		}
		if _, err := b.DB.Exec(`create table if not exists 'values' (
			id integer primary key,
			varId integer,
			value string
		);`); err != nil {
			log.Fatal("Initial DB migration create values table: ", err)
		}
	}
}

// Adds a constructed handler to the bots handlers list
func (b *Bot) AddHandler(name string, h Handler) {
	b.Plugins[strings.ToLower(name)] = h
	b.PluginOrdering = append(b.PluginOrdering, name)
	if entry := h.RegisterWeb(); entry != nil {
		b.httpEndPoints[name] = *entry
	}
}

func (b *Bot) Who(channel string) []User {
	out := []User{}
	for _, u := range b.Users {
		if u.Name != b.Config.Nick {
			out = append(out, u)
		}
	}
	return out
}

var rootIndex string = `
<!DOCTYPE html>
<html>
	<head>
		<title>Factoids</title>
		<link rel="stylesheet" href="http://yui.yahooapis.com/pure/0.1.0/pure-min.css">
                <meta name="viewport" content="width=device-width, initial-scale=1">
	</head>
	{{if .EndPoints}}
	<div style="padding-top: 1em;">
		<table class="pure-table">
			<thead>
				<tr>
					<th>Plugin</th>
				</tr>
			</thead>

			<tbody>
				{{range $key, $value := .EndPoints}}
				<tr>
					<td><a href="{{$value}}">{{$key}}</a></td>
				</tr>
				{{end}}
			</tbody>
		</table>
	</div>
	{{end}}
</html>
`

func (b *Bot) serveRoot(w http.ResponseWriter, r *http.Request) {
	context := make(map[string]interface{})
	context["EndPoints"] = b.httpEndPoints
	t, err := template.New("rootIndex").Parse(rootIndex)
	if err != nil {
		log.Println(err)
	}
	t.Execute(w, context)
}
