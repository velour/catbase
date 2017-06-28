// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Crash contains the plugin that allows the bot to pad messages
package crash

import (
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type CrashPlugin struct {
	bot    bot.Bot
	config *config.Config
}

// New creates a new CrashPlugin with the Plugin interface
func New(bot bot.Bot) *CrashPlugin {
	p := CrashPlugin{
		bot:    bot,
		config: bot.Config(),
	}
	return &p
}

func (p *CrashPlugin) Message(message msg.Message) bool {
	if !message.Command {
		return false
	}

	if message.Body == "CRASH!" {
		p.bot.SendMessage(message.Channel, "Crashing...")
		for {
		}
	}

	return false
}

func (p *CrashPlugin) Event(e string, message msg.Message) bool {
	return false
}

func (p *CrashPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *CrashPlugin) Help(e string, m []string) {
}

func (p *CrashPlugin) RegisterWeb() *string {
	// nothing to register
	return nil
}
