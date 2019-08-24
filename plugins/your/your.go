// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package your

import (
	"math/rand"
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
func New(b bot.Bot) *YourPlugin {
	yp := &YourPlugin{
		bot:    b,
		config: b.Config(),
	}
	b.Register(yp, bot.Message, yp.message)
	b.Register(yp, bot.Help, yp.help)
	return yp
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *YourPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	maxLen := p.config.GetInt("your.maxlength", 140)
	if len(message.Body) > maxLen {
		return false
	}
	msg := message.Body
	for _, replacement := range p.config.GetArray("Your.Replacements", []string{}) {
		freq := p.config.GetFloat64("your.replacements."+replacement+".freq", 0.0)
		this := p.config.Get("your.replacements."+replacement+".this", "")
		that := p.config.Get("your.replacements."+replacement+".that", "")
		if rand.Float64() < freq {
			r := strings.NewReplacer(this, that)
			msg = r.Replace(msg)
		}
	}
	if msg != message.Body {
		p.bot.Send(c, bot.Message, message.Channel, msg)
		return true
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *YourPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.bot.Send(c, bot.Message, message.Channel, "Your corrects people's grammar.")
	return true
}
