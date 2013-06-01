package plugins

import (
	"math/rand"
	"strings"
	"time"

	"github.com/chrissexton/alepale/bot"
)

type YourPlugin struct {
	Bot *bot.Bot
}

// NewYourPlugin creates a new YourPlugin with the Plugin interface
func NewYourPlugin(bot *bot.Bot) *YourPlugin {
	return &YourPlugin{
		Bot: bot,
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *YourPlugin) Message(message bot.Message) bool {
	lower := strings.ToLower(message.Body)
	if strings.Contains(lower, "your") || strings.Contains(lower, "you're") {
		if rand.Float64() < 0.2 {
			r := strings.NewReplacer("Your", "You're", "your", "you're", "You're",
				"Your", "you're", "your", "Youre", "Your", "youre", "your")
			msg := r.Replace(message.Body)
			p.Bot.SendMessage(message.Channel, msg)
			return true
		}
	}
	return false
}

// LoadData imports any configuration data into the plugin. This is not strictly necessary other
// than the fact that the Plugin interface demands it exist. This may be deprecated at a later
// date.
func (p *YourPlugin) LoadData() {
	rand.Seed(time.Now().Unix())
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *YourPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Your corrects people's grammar.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *YourPlugin) Event(kind string, message bot.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *YourPlugin) BotMessage(message bot.Message) bool {
	return false
}

// Register any web URLs desired
func (p *YourPlugin) RegisterWeb() *string {
	return nil
}
