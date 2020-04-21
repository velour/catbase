// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/msglog"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

// bot type provides storage for bot-wide information, configs, and database connections
type bot struct {
	// Each plugin must be registered in our plugins handler. To come: a map so that this
	// will allow plugins to respond to specific kinds of events
	plugins        map[string]Plugin
	pluginOrdering []string

	// Users holds information about all of our friends
	users []user.User
	// Represents the bot
	me user.User

	config *config.Config

	conn Connector

	logIn  chan msg.Message
	logOut chan msg.Messages

	version string

	// The entries to the bot's HTTP interface
	httpEndPoints []EndPoint

	// filters registered by plugins
	filters map[string]func(string) string

	callbacks CallbackMap

	password        string
	passwordCreated time.Time
}

type EndPoint struct {
	Name, URL string
}

// Variable represents a $var replacement
type Variable struct {
	Variable, Value string
}

// New creates a bot for a given connection and set of handlers.
func New(config *config.Config, connector Connector) Bot {
	logIn := make(chan msg.Message)
	logOut := make(chan msg.Messages)

	msglog.RunNew(logIn, logOut)

	users := []user.User{
		{
			Name: config.Get("Nick", "bot"),
		},
	}

	bot := &bot{
		config:         config,
		plugins:        make(map[string]Plugin),
		pluginOrdering: make([]string, 0),
		conn:           connector,
		users:          users,
		me:             users[0],
		logIn:          logIn,
		logOut:         logOut,
		httpEndPoints:  make([]EndPoint, 0),
		filters:        make(map[string]func(string) string),
		callbacks:      make(CallbackMap),
	}

	bot.migrateDB()

	http.HandleFunc("/", bot.serveRoot)

	connector.RegisterEvent(bot.Receive)

	return bot
}

func (b *bot) DefaultConnector() Connector {
	return b.conn
}

func (b *bot) WhoAmI() string {
	return b.me.Name
}

// Config gets the configuration that the bot is using
func (b *bot) Config() *config.Config {
	return b.config
}

func (b *bot) DB() *sqlx.DB {
	return b.config.DB
}

// Create any tables if necessary based on version of DB
// Plugins should create their own tables, these are only for official bot stuff
// Note: This does not return an error. Database issues are all fatal at this stage.
func (b *bot) migrateDB() {
	if _, err := b.DB().Exec(`create table if not exists variables (
			id integer primary key,
			name string,
			value string
		);`); err != nil {
		log.Fatal().Err(err).Msgf("Initial DB migration create variables table")
	}
}

// Adds a constructed handler to the bots handlers list
func (b *bot) AddPlugin(h Plugin) {
	name := reflect.TypeOf(h).String()
	b.plugins[name] = h
	b.pluginOrdering = append(b.pluginOrdering, name)
}

func (b *bot) Who(channel string) []user.User {
	names := b.conn.Who(channel)
	users := []user.User{}
	for _, n := range names {
		users = append(users, user.New(n))
	}
	return users
}

var suffixRegex *regexp.Regexp

// IsCmd checks if message is a command and returns its curtailed version
func IsCmd(c *config.Config, message string) (bool, string) {
	cmdcs := c.GetArray("CommandChar", []string{"!"})
	botnick := strings.ToLower(c.Get("Nick", "bot"))
	r := fmt.Sprintf(`(?i)\s*\W*\s*?%s\W*$`, botnick)
	if suffixRegex == nil {
		suffixRegex = regexp.MustCompile(r)
	}
	if botnick == "" {
		log.Fatal().
			Msgf(`You must run catbase -set nick -val <your bot nick>`)
	}
	iscmd := false
	lowerMessage := strings.ToLower(message)

	if strings.HasPrefix(lowerMessage, botnick) &&
		len(lowerMessage) > len(botnick) &&
		(lowerMessage[len(botnick)] == ',' || lowerMessage[len(botnick)] == ':') {

		iscmd = true
		message = message[len(botnick):]

		// trim off the customary addressing punctuation
		if message[0] == ':' || message[0] == ',' {
			message = message[1:]
		}
	} else if suffixRegex.MatchString(message) {
		iscmd = true
		message = suffixRegex.ReplaceAllString(message, "")
	} else {
		for _, cmdc := range cmdcs {
			if strings.HasPrefix(lowerMessage, cmdc) && len(cmdc) > 0 {
				iscmd = true
				message = message[len(cmdc):]
				break
			}
		}
	}

	// trim off any whitespace left on the message
	message = strings.TrimSpace(message)

	return iscmd, message
}

func (b *bot) CheckAdmin(nick string) bool {
	for _, u := range b.Config().GetArray("Admins", []string{}) {
		if nick == u {
			return true
		}
	}
	return false
}

var users = map[string]*user.User{}

func (b *bot) GetUser(nick string) *user.User {
	if _, ok := users[nick]; !ok {
		users[nick] = &user.User{
			Name:  nick,
			Admin: b.checkAdmin(nick),
		}
	}
	return users[nick]
}

func (b *bot) NewUser(nick string) *user.User {
	return &user.User{
		Name:  nick,
		Admin: b.checkAdmin(nick),
	}
}

func (b *bot) checkAdmin(nick string) bool {
	for _, u := range b.Config().GetArray("Admins", []string{}) {
		if nick == u {
			return true
		}
	}
	return false
}

// Register a text filter which every outgoing message is passed through
func (b *bot) RegisterFilter(name string, f func(string) string) {
	b.filters[name] = f
}

// Register a callback
func (b *bot) Register(p Plugin, kind Kind, cb Callback) {
	t := reflect.TypeOf(p).String()
	if _, ok := b.callbacks[t]; !ok {
		b.callbacks[t] = make(map[Kind][]Callback)
	}
	if _, ok := b.callbacks[t][kind]; !ok {
		b.callbacks[t][kind] = []Callback{}
	}
	b.callbacks[t][kind] = append(b.callbacks[t][kind], cb)
}

func (b *bot) RegisterWeb(root, name string) {
	b.httpEndPoints = append(b.httpEndPoints, EndPoint{name, root})
}

func (b *bot) GetPassword() string {
	if b.passwordCreated.Before(time.Now().Add(-24 * time.Hour)) {
		adjs := b.config.GetArray("bot.passwordAdjectives", []string{"very"})
		nouns := b.config.GetArray("bot.passwordNouns", []string{"noun"})
		verbs := b.config.GetArray("bot.passwordVerbs", []string{"do"})
		a, n, v := adjs[rand.Intn(len(adjs))], nouns[rand.Intn(len(nouns))], verbs[rand.Intn(len(verbs))]
		b.passwordCreated = time.Now()
		b.password = fmt.Sprintf("%s-%s-%s", a, n, v)
	}
	return b.password
}
