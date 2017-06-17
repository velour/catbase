// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
	"time"
	"math/rand"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type ReactionPlugin struct {
	Bot bot.Bot
}

func New(bot bot.Bot) *ReactionPlugin {
	rand.Seed(time.Now().Unix())

	return &ReactionPlugin{
		Bot: bot,
	}
}

func (p *ReactionPlugin) Message(message msg.Message) bool {
	if rand.Intn(100) == 0 {
		p.Bot.React(message.Channel, "+1", message)
	}
	return false
}

func (p *ReactionPlugin) Help(channel string, parts []string) {

}

func (p *ReactionPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *ReactionPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *ReactionPlugin) RegisterWeb() *string {
	return nil
}
