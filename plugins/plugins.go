package plugins

import "fmt"
import "github.com/chrissexton/alepale/bot"

// Plugin interface defines the methods needed to accept a plugin
type Plugin interface {
	Message(message bot.Message) bool
	Event(kind string, message bot.Message) bool
	BotMessage(message bot.Message) bool
	LoadData()
	Help()
}

// ---- Below are some example plugins

// Creates a new TestPlugin with the Plugin interface
func NewTestPlugin(bot *bot.Bot) *TestPlugin {
	tp := TestPlugin{}
	tp.LoadData()
	tp.Bot = bot
	return &tp
}

// TestPlugin type allows our plugin to store persistent state information
type TestPlugin struct {
	Bot      *bot.Bot
	Responds []string
	Name     string
	Feces    string
	helpmsg  []string
}

func (p *TestPlugin) LoadData() {
	config := GetPluginConfig("TestPlugin")
	p.Name = config.Name
	p.Feces = config.Values["Feces"].(string)
	p.helpmsg = []string{
		"TestPlugin just shows off how shit works.",
	}
}

func (p *TestPlugin) Message(message bot.Message) bool {
	user := message.User
	channel := message.Channel
	body := message.Body

	fmt.Println(user, body)
	fmt.Println("My plugin name is:", p.Name, " My feces are:", p.Feces)
	p.Bot.SendMessage(channel, body)
	return true
}

func (p *TestPlugin) Help(message bot.Message) {
	for _, msg := range p.helpmsg {
		p.Bot.SendMessage(message.Channel, msg)
	}
}

// Empty event handler because this plugin does not do anything on event recv
func (p *TestPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *TestPlugin) BotMessage(message bot.Message) bool {
	return false
}

type PluginConfig struct {
	Name   string
	Values map[string]interface{}
}

// Loads plugin config out of the DB
// Stored in db.plugins.find("name": name)
func GetPluginConfig(name string) PluginConfig {
	return PluginConfig{
		Name: "TestPlugin",
		Values: map[string]interface{}{
			"Feces":    "test",
			"Responds": "fucker",
		},
	}
}

// FalsePlugin shows how plugin fallthrough works for handling messages
type FalsePlugin struct{}

func (fp FalsePlugin) Message(user, message string) bool {
	fmt.Println("FalsePlugin returning false.")
	return false
}

func (fp FalsePlugin) LoadData() {

}

// Empty event handler because this plugin does not do anything on event recv
func (p *FalsePlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *FalsePlugin) BotMessage(message bot.Message) bool {
	return false
}
