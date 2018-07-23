// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package picker

import (
	"strings"

	"fmt"
	"math/rand"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type PickerPlugin struct {
	Bot bot.Bot
}

// NewPickerPlugin creates a new PickerPlugin with the Plugin interface
func New(bot bot.Bot) *PickerPlugin {
	return &PickerPlugin{
		Bot: bot,
	}
}

// Message responds to the bot hook on recieving messages.
// This function returns true if the plugin responds in a meaningful way to the users message.
// Otherwise, the function returns false and the bot continues execution of other plugins.
func (p *PickerPlugin) Message(message msg.Message) bool {
	body := message.Body
	pfx, sfx := "pick {", "}"

	if strings.HasPrefix(body, pfx) && strings.HasSuffix(body, sfx) {
		body = strings.TrimSuffix(strings.TrimPrefix(body, pfx), sfx)
		items := strings.Split(body, ",")
		item := items[rand.Intn(len(items))]

		out := fmt.Sprintf("I've chosen \"%s\" for you.", strings.TrimSpace(item))

		p.Bot.SendMessage(message.Channel, out)

		return true
	}
	return false
}

// Help responds to help requests. Every plugin must implement a help function.
func (p *PickerPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "Choose from a list of options. Try \"pick {a,b,c}\".")
}

// Empty event handler because this plugin does not do anything on event recv
func (p *PickerPlugin) Event(kind string, message msg.Message) bool {
	return false
}

// Handler for bot's own messages
func (p *PickerPlugin) BotMessage(message msg.Message) bool {
	return false
}

// Register any web URLs desired
func (p *PickerPlugin) RegisterWeb() *string {
	return nil
}

func (p *PickerPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
