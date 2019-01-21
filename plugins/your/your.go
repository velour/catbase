// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package your

import (
	"math/rand"
	"strconv"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type YourPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// NewYourPlugin creates a new YourPlugin with the Plugin interface
func New(bot bot.Bot) *YourPlugin {
	return &YourPlugin{
		bot:    bot,
		config: bot.Config(),
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *YourPlugin) Message(message msg.Message) bool {
	maxLen := p.config.GetInt("your.maxlength")
	if maxLen == 0 {
		maxLen = 140
		p.config.Set("your.maxlength", strconv.Itoa(maxLen))
	}
	if len(message.Body) > maxLen {
		return false
	}
	msg := message.Body
	for _, replacement := range p.config.GetArray("Your.Replacements") {
		freq := p.config.GetFloat64("your.replacements." + replacement + ".freq")
		this := p.config.Get("your.replacements." + replacement + ".this")
		that := p.config.Get("your.replacements." + replacement + ".that")
		if rand.Float64() < freq {
			r := strings.NewReplacer(this, that)
			msg = r.Replace(msg)
		}
	}
	if msg != message.Body {
		p.bot.SendMessage(message.Channel, msg)
		return true
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *YourPlugin) Help(channel string, parts []string) {
	p.bot.SendMessage(channel, "Your corrects people's grammar.")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *YourPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *YourPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *YourPlugin) RegisterWeb() *string {
	return nil
}

func (p *YourPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
