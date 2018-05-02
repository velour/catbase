// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
	"math/rand"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type ReactionPlugin struct {
	Bot    bot.Bot
	Config *config.Config
}

func New(bot bot.Bot) *ReactionPlugin {
	rand.Seed(time.Now().Unix())

	return &ReactionPlugin{
		Bot:    bot,
		Config: bot.Config(),
	}
}

func (p *ReactionPlugin) Message(message msg.Message) bool {
	harrass := false
	for _, nick := range p.Config.Reaction.HarrassList {
		if message.User.Name == nick {
			harrass = true
			break
		}
	}

	chance := p.Config.Reaction.GeneralChance
	negativeWeight := 1
	if harrass {
		chance = p.Config.Reaction.HarrassChance
		negativeWeight = p.Config.Reaction.NegativeHarrassmentMultiplier
	}

	if rand.Float64() < chance {
		numPositiveReactions := len(p.Config.Reaction.PositiveReactions)
		numNegativeReactions := len(p.Config.Reaction.NegativeReactions)

		maxIndex := numPositiveReactions + numNegativeReactions*negativeWeight

		index := rand.Intn(maxIndex)

		reaction := ""

		if index < numPositiveReactions {
			reaction = p.Config.Reaction.PositiveReactions[index]
		} else {
			index -= numPositiveReactions
			index %= numNegativeReactions
			reaction = p.Config.Reaction.NegativeReactions[index]
		}

		p.Bot.React(message.Channel, reaction, message)
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

func (p *ReactionPlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }
