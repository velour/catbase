// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package couldashouldawoulda

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type CSWPlugin struct {
	Bot    bot.Bot
	Config *config.Config
}

func New(bot bot.Bot) *CSWPlugin {
	return &CSWPlugin{
		Bot: bot,
	}
}

func (p *CSWPlugin) Message(message msg.Message) bool {
	if !message.Command {
		return false
	}

	lowercase := strings.ToLower(message.Body)
	tokens := strings.Fields(lowercase)

	if len(tokens) < 3 {
		return false
	}

	could := tokens[0] == "could"
	should := tokens[0] ==  "should"
	would := tokens[0] == "would"

	if could || should || would {
		ors := strings.Count(lowercase, " or ")
		var responses []string
		if ors == 0 || could {
			responses = []string{"Yes.", "No.", "Maybe.", "For fucks sake, how should I know?"}
		} else if ors == 1 {
			responses = []string{"The former.", "The latter.", "Obviously the former.", "Clearly the latter.", "Can't it be both?"}
		} else {
			responses = make([]string, ors*2)
			for i := 0; i < ors; i++ {
				responses[i*2] = fmt.Sprintf("I'd say option %d.", i+1)
				responses[i*2+1] = "You'd be an idiot not to choose the "
				if i == 0 {
					responses[i*2+1] += " 1st"
				} else if i == 1 {
					responses[i*2+1] += " 2nd"
				} else if i == 2 {
					responses[i*2+1] += " 3rd"
				} else {
					responses[i*2+1] += fmt.Sprintf(" %dth", i+1)
				}
			}
		}

		p.Bot.SendMessage(message.Channel, responses[rand.Intn(len(responses))])
		return true
	}

	return false
}

func (p *CSWPlugin) Help(channel string, parts []string) {}

func (p *CSWPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *CSWPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *CSWPlugin) RegisterWeb() *string {
	return nil
}
