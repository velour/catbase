// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot/history"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/msglog"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
	"golang.org/x/crypto/bcrypt"
)

// bot type provides storage for bot-wide information, configs, and database connections
type bot struct {
	// Each plugin must be registered in our plugins handler. To come: a map so that this
	// will allow plugins to respond to specific kinds of events
	plugins        map[string]Plugin
	pluginOrdering []string

	// channel -> plugin
	pluginBlacklist map[string]bool

	// plugin, this is bot-wide
	pluginWhitelist map[string]bool

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

	quiet bool

	router *chi.Mux

	history *history.History
}

type EndPoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
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

	historySz := config.GetInt("bot.historysz", 100)

	users := []user.User{
		{
			Name: config.Get("Nick", "bot"),
		},
	}

	bot := &bot{
		config:          config,
		plugins:         make(map[string]Plugin),
		pluginOrdering:  make([]string, 0),
		pluginBlacklist: make(map[string]bool),
		pluginWhitelist: make(map[string]bool),
		conn:            connector,
		users:           users,
		me:              users[0],
		logIn:           logIn,
		logOut:          logOut,
		httpEndPoints:   make([]EndPoint, 0),
		filters:         make(map[string]func(string) string),
		callbacks:       make(CallbackMap),
		router:          chi.NewRouter(),
		history:         history.New(historySz),
	}

	bot.migrateDB()

	bot.RefreshPluginBlacklist()
	bot.RefreshPluginWhitelist()

	log.Debug().Msgf("created web router")

	bot.setupHTTP()

	connector.RegisterEvent(bot.Receive)

	return bot
}

func (b *bot) setupHTTP() {
	// Make the http logger optional
	// It has never served a purpose in production and with the emojy page, can make a rather noisy log
	if b.Config().GetInt("bot.useLogger", 0) == 1 {
		b.router.Use(middleware.Logger)
	}

	reqCount := b.Config().GetInt("bot.httprate.requests", 500)
	reqTime := time.Duration(b.Config().GetInt("bot.httprate.seconds", 5))
	if reqCount > 0 && reqTime > 0 {
		b.router.Use(httprate.LimitByIP(reqCount, reqTime*time.Second))
	}

	b.router.Use(middleware.RequestID)
	b.router.Use(middleware.Recoverer)
	b.router.Use(middleware.StripSlashes)

	b.router.HandleFunc("/", b.serveRoot)
	b.router.HandleFunc("/nav", b.serveNav)
	b.router.HandleFunc("/navHTML", b.serveNavHTML)
	b.router.HandleFunc("/navHTML/{currentPage}", b.serveNavHTML)
}

func (b *bot) ListenAndServe() {
	addr := b.config.Get("HttpAddr", "127.0.0.1:1337")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		log.Debug().Msgf("starting web service at %s", addr)
		log.Fatal().Err(http.ListenAndServe(addr, b.router)).Msg("bot killed")
	}()
	<-stop
	b.DefaultConnector().Shutdown()
	b.Receive(b.DefaultConnector(), Shutdown, msg.Message{})
}

func (b *bot) RegisterWeb(r http.Handler, root string) {
	b.router.Mount(root, r)
}

func (b *bot) RegisterWebName(r http.Handler, root, name string) {
	b.httpEndPoints = append(b.httpEndPoints, EndPoint{name, root})
	b.router.Mount(root, r)
}

// DefaultConnector is the main connector used for the bot
// If more than one connector is on, some users may not see all messages if this is used.
// Usage should be limited to out-of-band communications such as timed messages.
func (b *bot) DefaultConnector() Connector {
	return b.conn
}

// WhoAmI returns the bot's current registered name
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
		log.Fatal().Err(err).Msgf("Initial db migration create variables table")
	}
	if _, err := b.DB().Exec(`create table if not exists pluginBlacklist (
			channel string,
			name string,
			primary key (channel, name)
		);`); err != nil {
		log.Fatal().Err(err).Msgf("Initial db migration create blacklist table")
	}
	if _, err := b.DB().Exec(`create table if not exists pluginWhitelist (
			name string primary key
		);`); err != nil {
		log.Fatal().Err(err).Msgf("Initial db migration create whitelist table")
	}
}

// Adds a constructed handler to the bots handlers list
func (b *bot) AddPlugin(h Plugin) {
	name := reflect.TypeOf(h).String()
	b.plugins[name] = h
	b.pluginOrdering = append(b.pluginOrdering, name)
}

// Who returns users for a channel the bot is in
// Check the particular connector for channel values
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

func (b *bot) CheckAdmin(ID string) bool {
	admins := b.Config().GetArray("Admins", []string{})
	log.Info().Interface("admins", admins).Msgf("Checking admin for %s", ID)
	for _, u := range admins {
		if ID == u {
			log.Info().Msgf("%s admin check: passed", u)
			return true
		}
	}
	log.Info().Msg("%s admin check: failed")
	return false
}

// Register a text filter which every outgoing message is passed through
func (b *bot) RegisterFilter(name string, f func(string) string) {
	b.filters[name] = f
}

// RegisterTable registers multiple regex handlers at a time
func (b *bot) RegisterTable(p Plugin, handlers HandlerTable) {
	for _, h := range handlers {
		if h.IsCmd {
			b.RegisterRegexCmd(p, h.Kind, h.Regex, h.Handler)
		} else {
			b.RegisterRegex(p, h.Kind, h.Regex, h.Handler)
		}
	}
}

// RegisterRegex does what register does, but with a matcher
func (b *bot) RegisterRegex(p Plugin, kind Kind, r *regexp.Regexp, resp ResponseHandler) {
	t := PluginName(p)
	if _, ok := b.callbacks[t]; !ok {
		b.callbacks[t] = make(map[Kind][]HandlerSpec)
	}
	if _, ok := b.callbacks[t][kind]; !ok {
		b.callbacks[t][kind] = []HandlerSpec{}
	}
	spec := HandlerSpec{
		Kind:    kind,
		Regex:   r,
		Handler: resp,
	}
	b.callbacks[t][kind] = append(b.callbacks[t][kind], spec)
}

// RegisterRegexCmd is a shortcut to filter non-command messages from a registration
func (b *bot) RegisterRegexCmd(p Plugin, kind Kind, r *regexp.Regexp, resp ResponseHandler) {
	newResp := func(req Request) bool {
		if !req.Msg.Command {
			return false
		}
		return resp(req)
	}
	b.RegisterRegex(p, kind, r, newResp)
}

// Register a callback
// This function should be considered deprecated.
func (b *bot) Register(p Plugin, kind Kind, cb Callback) {
	r := regexp.MustCompile(`.*`)
	resp := func(r Request) bool {
		return cb(r.Conn, r.Kind, r.Msg, r.Args...)
	}
	b.RegisterRegex(p, kind, r, resp)
}

// GetPassword returns a random password generated for the bot
// Passwords expire in 24h and are used for the web interface
func (b *bot) GetPassword() string {
	if override := b.config.Get("bot.password", ""); override != "" {
		return override
	}
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

// SetQuiet is called to silence the bot from sending channel messages
func (b *bot) SetQuiet(status bool) {
	b.quiet = status
}

// RefreshPluginBlacklist loads data for which plugins are disabled for particular channels
func (b *bot) RefreshPluginBlacklist() error {
	blacklistItems := []struct {
		Channel string
		Name    string
	}{}
	if err := b.DB().Select(&blacklistItems, `select channel, name from pluginBlacklist`); err != nil {
		return fmt.Errorf("%w", err)
	}
	b.pluginBlacklist = make(map[string]bool)
	for _, i := range blacklistItems {
		b.pluginBlacklist[i.Channel+i.Name] = true
	}
	log.Debug().Interface("blacklist", b.pluginBlacklist).Msgf("Refreshed plugin blacklist")
	return nil
}

// RefreshPluginWhitelist loads data for which plugins are enabled
func (b *bot) RefreshPluginWhitelist() error {
	whitelistItems := []struct {
		Name string
	}{
		{Name: "admin"}, // we must always ensure admin is on!
	}
	if err := b.DB().Select(&whitelistItems, `select name from pluginWhitelist`); err != nil {
		return fmt.Errorf("%w", err)
	}
	b.pluginWhitelist = make(map[string]bool)
	for _, i := range whitelistItems {
		b.pluginWhitelist[i.Name] = true
	}
	log.Debug().Interface("whitelist", b.pluginWhitelist).Msgf("Refreshed plugin whitelist")
	return nil
}

// GetPluginNames returns an ordered list of plugins loaded (used for blacklisting)
func (b *bot) GetPluginNames() []string {
	names := []string{}
	for _, name := range b.pluginOrdering {
		names = append(names, pluginNameStem(name))
	}
	return names
}

func (b *bot) GetWhitelist() []string {
	list := []string{}
	for k := range b.pluginWhitelist {
		list = append(list, k)
	}
	return list
}

func (b *bot) OnBlacklist(channel, plugin string) bool {
	return b.pluginBlacklist[channel+plugin]
}

func (b *bot) onWhitelist(plugin string) bool {
	return b.pluginWhitelist[plugin]
}

func pluginNameStem(name string) string {
	return strings.Split(strings.TrimPrefix(name, "*"), ".")[0]
}

func PluginName(p Plugin) string {
	t := reflect.TypeOf(p).String()
	return t
}

func (b *bot) CheckPassword(secret, password string) bool {
	log.Debug().Msgf("CheckPassword(%s, %s) => b.password=%s, b.GetPassword()=%s", secret, password, b.password, b.GetPassword())
	if password == "" {
		return false
	}
	if b.GetPassword() == password {
		return true
	}
	parts := strings.SplitN(password, ":", 2)
	if len(parts) == 2 {
		secret = parts[0]
		password = parts[1]
	}
	q := `select encoded_pass from apppass where secret = ?`
	encodedPasswords := [][]byte{}
	b.DB().Select(&encodedPasswords, q, secret)
	for _, p := range encodedPasswords {
		if err := bcrypt.CompareHashAndPassword(p, []byte(password)); err == nil {
			return true
		}
	}
	return false
}
