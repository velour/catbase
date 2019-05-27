// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
	"math/rand"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type ReactionPlugin struct {
	bot    bot.Bot
	config *config.Config
}

func New(b bot.Bot) *ReactionPlugin {
	rp := &ReactionPlugin{
		bot:    b,
		config: b.Config(),
	}
	b.Register(rp, bot.Message, rp.message)
	return rp
}

func (p *ReactionPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	harrass := false
	for _, nick := range p.config.GetArray("Reaction.HarrassList", []string{}) {
		if message.User.Name == nick {
			harrass = true
			break
		}
	}

	chance := p.config.GetFloat64("Reaction.GeneralChance", 0.01)
	negativeWeight := 1
	if harrass {
		chance = p.config.GetFloat64("Reaction.HarrassChance", 0.05)
		negativeWeight = p.config.GetInt("Reaction.NegativeHarrassmentMultiplier", 2)
	}

	if rand.Float64() < chance {
		numPositiveReactions := len(p.config.GetArray("Reaction.PositiveReactions", []string{}))
		numNegativeReactions := len(p.config.GetArray("Reaction.NegativeReactions", []string{}))

		maxIndex := numPositiveReactions + numNegativeReactions*negativeWeight

		index := rand.Intn(maxIndex)

		reaction := ""

		if index < numPositiveReactions {
			reaction = p.config.GetArray("Reaction.PositiveReactions", []string{})[index]
		} else {
			index -= numPositiveReactions
			index %= numNegativeReactions
			reaction = p.config.GetArray("Reaction.NegativeReactions", []string{})[index]
		}

		p.bot.Send(c, bot.Reaction, message.Channel, reaction, message)
	}

	return false
}
