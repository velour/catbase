package topic

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/connectors/discord"
	"regexp"
)

type Topic struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *Topic {
	t := &Topic{
		b: b,
		c: b.Config(),
	}
	t.register()
	return t
}

func (p *Topic) register() {
	p.b.RegisterRegexCmd(p, bot.Message, regexp.MustCompile(`(?i)^topic (?P<topic>.+)$`), func(r bot.Request) bool {
		switch conn := r.Conn.(type) {
		case *discord.Discord:
			err := conn.SetTopic(r.Msg.Channel, r.Values["topic"])
			if err != nil {
				log.Error().Err(err).Msg("couldn't set topic")
				return false
			}
			topic, err := conn.Topic(r.Msg.Channel)
			if err != nil {
				log.Error().Err(err).Msg("couldn't get topic")
				return false
			}
			p.b.Send(conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Topic: %s", topic))
			return true

		}
		return false
	})
	p.b.RegisterRegexCmd(p, bot.Message, regexp.MustCompile(`(?i)^topic$`), func(r bot.Request) bool {
		switch conn := r.Conn.(type) {
		case *discord.Discord:
			topic, err := conn.Topic(r.Msg.Channel)
			if err != nil {
				log.Error().Err(err).Msg("couldn't get topic")
				return false
			}
			p.b.Send(conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Topic: %s", topic))
			return true

		}
		return false
	})
}
