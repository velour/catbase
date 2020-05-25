// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
	"math/rand"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/chrissexton/sentiment"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type ReactionPlugin struct {
	bot    bot.Bot
	config *config.Config

	model sentiment.Models
}

func New(b bot.Bot) *ReactionPlugin {
	model, err := sentiment.Restore()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't restore sentiment model")
	}
	rp := &ReactionPlugin{
		bot:    b,
		config: b.Config(),
		model:  model,
	}
	b.Register(rp, bot.Message, rp.message)
	return rp
}

func (p *ReactionPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	chance := p.config.GetFloat64("Reaction.GeneralChance", 0.01)
	if rand.Float64() < chance {
		analysis := p.model.SentimentAnalysis(message.Body, sentiment.English)

		log.Debug().
			Uint8("score", analysis.Score).
			Str("body", message.Body).
			Msg("sentiment of statement")

		var reactions []string
		if analysis.Score > 0 {
			reactions = p.config.GetArray("Reaction.PositiveReactions", []string{})
		} else {
			reactions = p.config.GetArray("Reaction.NegativeReactions", []string{})
		}

		reaction := reactions[rand.Intn(len(reactions))]

		p.bot.Send(c, bot.Reaction, message.Channel, reaction, message)
	}

	p.checkReactions(c, message)

	return false
}

// b will always react if a message contains a check word
// Note that reactions must not be enclosed in :
func (p *ReactionPlugin) checkReactions(c bot.Connector, m msg.Message) {
	checkWords := p.config.GetArray("reaction.checkwords", []string{})
	reactions := p.config.GetArray("reaction.checkedreactions", []string{})

	for i, w := range checkWords {
		if strings.Contains(strings.ToLower(m.Body), w) {
			react := strings.Trim(reactions[i], ":")
			p.bot.Send(c, bot.Reaction, m.Channel, react, m)
		}
	}
}
