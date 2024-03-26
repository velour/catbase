// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

package bot

import (
	"github.com/gabriel-vasile/mimetype"
	"github.com/velour/catbase/bot/web"
	"net/http"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

const (
	_ = iota

	// Any will be picked up by the matcher as generic
	Any
	// Message any standard chat
	Message
	// Ephemeral sends a disappearing message to a user in chat
	Ephemeral
	// Reply something containing a message reference
	Reply
	// Action any /me action
	Action
	// Spoiler is for commented out messages
	Spoiler
	// Reaction Icon reaction if service supports it
	Reaction
	// Edit message ref'd new message to replace
	Edit
	// Event is unknown/generic
	Event
	// Help is used when the bot help system is triggered
	Help
	// SelfMessage triggers when the bot is sending a message
	SelfMessage
	// Delete removes a message by ID
	Delete
	// Startup is triggered after the connector has run the Serve function
	Startup
	// Shutdown is triggered after an OS interrupt
	Shutdown
)

type EphemeralID string

type UnfurlLinks bool

type EmbedAuthor struct {
	ID      string
	Who     string
	IconURL string
}

type File struct {
	Description string
	Data        []byte
	mime        *mimetype.MIME
}

type MessageReference struct {
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id,omitempty"`
	GuildID   string `json:"guild_id,omitempty"`
}

func (f File) Mime() *mimetype.MIME {
	if f.mime == nil {
		f.mime = mimetype.Detect(f.Data)
	}
	return f.mime
}

func (f File) ContentType() string {
	return f.Mime().String()
}

func (f File) FileName() string {
	ext := f.Mime().Extension()
	return strings.ReplaceAll(f.Description, " ", "-") + ext
}

type ImageAttachment struct {
	URL    string
	AltTxt string
	Width  int
	Height int
}

type Request struct {
	Conn   Connector
	Kind   Kind
	Msg    msg.Message
	Values RegexValues
	Args   []any
}

type Kind int
type Callback func(Connector, Kind, msg.Message, ...any) bool
type ResponseHandler func(Request) bool
type CallbackMap map[string]map[Kind][]HandlerSpec

type HandlerSpec struct {
	Kind     Kind
	IsCmd    bool
	Regex    *regexp.Regexp
	HelpText string
	Handler  ResponseHandler
}
type HandlerTable []HandlerSpec

type RegexValues map[string]string

// b interface serves to allow mocking of the actual bot
type Bot interface {
	// Config allows access to the bot's configuration system
	Config() *config.Config

	// DB gives access to the current database
	DB() *sqlx.DB

	// Who lists users in a particular channel
	// The channel should be represented as an ID for slack (check the connector for details)
	Who(string) []user.User

	// WhoAmI gives a nick for the bot
	WhoAmI() string

	// AddPlugin registers a new plugin handler
	AddPlugin(Plugin)

	// Send transmits a message to a Connector.
	// Kind is listed in the bot's enum, one of bot.Message/Reply/Action/etc
	// Usually, the first vararg should be a channel ID, but refer to the Connector for info
	Send(Connector, Kind, ...any) (string, error)

	// bot receives from a Connector.
	// The Kind arg should be one of bot.Message/Reply/Action/etc
	Receive(Connector, Kind, msg.Message, ...any) bool

	// Register a set of plugin callbacks
	// Kind will be matched to the event for the callback
	RegisterTable(Plugin, HandlerTable)

	// Register a plugin callback
	// Kind will be matched to the event for the callback
	RegisterRegex(Plugin, Kind, *regexp.Regexp, ResponseHandler)

	// Register a plugin callback filtering non-commands out
	// Kind will be matched to the event for the callback
	RegisterRegexCmd(Plugin, Kind, *regexp.Regexp, ResponseHandler)

	// Register a plugin callback
	// Kind will be matched to the event for the callback
	Register(Plugin, Kind, Callback)

	// Filter prepares a message to be sent
	// This loads the message with things like $variables
	Filter(msg.Message, string) string

	// LastMessage returns the last message the bot has seen
	LastMessage(string) (msg.Message, error)

	// CheckAdmin returns a user's admin status
	CheckAdmin(string) bool

	// GetEmojiList returns known emoji
	GetEmojiList(bool) map[string]string

	// RegisterFilter creates a filter function for message processing
	RegisterFilter(string, func(string) string)

	// Start the HTTP service
	ListenAndServe()

	// DefaultConnector returns the base connector, which may not be the only connector
	DefaultConnector() Connector

	// GetWeb returns the bot's webserver structure
	GetWeb() *web.Web

	// GetPassword generates a unique password for modification commands on the public website
	GetPassword() string

	// SetQuiet toggles the bot's ability to send messages
	SetQuiet(bool)

	// GetPluginNames returns an ordered list of registered plugins
	GetPluginNames() []string

	// RefreshPluginBlacklist reloads the list of plugins disabled per room from the DB
	RefreshPluginBlacklist() error

	// RefreshPluginWhitelist reloads the list of plugins enabled from the DB
	RefreshPluginWhitelist() error

	// Get the contents of the white list
	GetWhitelist() []string

	// Check if a particular plugin is blacklisted
	OnBlacklist(string, string) bool

	// Check valid password
	CheckPassword(secret, password string) bool

	// PubToASub publishes a message to any subscribers
	PubToASub(subject string, payload any)
}

// Connector represents a server connection to a chat service
type Connector interface {
	// RegisterEvent creates a callback to watch Connector events
	RegisterEvent(Callback)

	// Send transmits a message on the connector
	Send(Kind, ...any) (string, error)

	// GetEmojiList returns a connector's known custom emoji
	GetEmojiList(bool) map[string]string

	// Serve starts a connector's connection routine
	Serve() error

	// Shutdown cleans up after the connection
	Shutdown()

	// Who returns a user list for a channel
	Who(string) []string

	// Profile returns a user's information given an ID
	Profile(string) (user.User, error)

	// URLFormat utility
	URLFormat(title, url string) string

	// Emojy translates emojy to/from services
	Emojy(string) string

	// UploadEmojy creates a new emojy on the server related to the file at path
	UploadEmojy(emojy, path string) error

	// DeleteEmojy removes the specified emojy from the server
	DeleteEmojy(emojy string) error

	// GetChannelName returns the human-friendly name for an ID (if possible)
	GetChannelName(id string) string

	// GetChannelID returns the channel ID for a human-friendly name (if possible)
	GetChannelID(id string) string

	// GetRouter gets any web handlers the connector exposes
	GetRouter() (http.Handler, string)

	// GetRoleNames returns list of names and IDs of roles
	GetRoles() ([]Role, error)

	// SetRole toggles a role on/off for a user by ID
	SetRole(userID, roleID string) error

	// Nick sets the username of the bot on the server
	Nick(string) error
}

// Plugin interface used for compatibility with the Plugin interface
// Uhh it turned empty, but we're still using it to ID plugins
type Plugin any

type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
