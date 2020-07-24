// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package reaction

import (
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
	br    *bayesReactor
}

func New(b bot.Bot) *ReactionPlugin {
	model, err := sentiment.Restore()
	if err != nil {
		log.Fatal().Err(err).Msg("Couldn't restore sentiment model")
	}
	c := b.Config()
	path := c.GetString("reaction.modelpath", "emojy.model.json")
	rp := &ReactionPlugin{
		bot:    b,
		config: c,
		model:  model,
		br:     newBayesReactor(path),
	}
	b.Register(rp, bot.Message, rp.message)
	return rp
}

func (p *ReactionPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	emojy, prob := p.br.React(message.Body)
	target := p.config.GetFloat64("reaction.confidence", 0.5)

	log.Debug().
		Float64("prob", prob).
		Float64("target", target).
		Bool("accept", prob > target).
		Str("emojy", emojy).
		Msgf("Reaction check")

	if prob > target {
		p.bot.Send(c, bot.Reaction, message.Channel, emojy, message)
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
