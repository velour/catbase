// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
	"time"
	"math/rand"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type ReactionPlugin struct {
	Bot bot.Bot
	Config *config.Config
}

func New(bot bot.Bot) *ReactionPlugin {
	rand.Seed(time.Now().Unix())

	return &ReactionPlugin{
		Bot: bot,
		Config: bot.Config(),
	}
}

func (p *ReactionPlugin) Message(message msg.Message) bool {
	outOf := int(1. / p.Config.Reaction.GeneralChance)

	for _, reaction := range p.Config.Reaction.PositiveReactions {
		if rand.Intn(outOf) == 0 {
			p.Bot.React(message.Channel, reaction, message)
			return false
		}
	}

	for _, nick := range p.Config.Reaction.HarrassList {
		if message.User.Name == nick {
			outOf = int(1. / p.Config.Reaction.HarrassChance)
			break
		}
	}

	for _, reaction := range p.Config.Reaction.NegativeReactions {
		if rand.Intn(outOf) == 0 {
			p.Bot.React(message.Channel, reaction, message)
			return false
		}
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
