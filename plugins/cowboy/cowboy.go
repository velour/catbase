package cowboy

import (
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type Cowboy struct {
	b bot.Bot
	c *config.Config

	emojyPath    string
	baseEmojyURL string
}

func New(b bot.Bot) *Cowboy {
	emojyPath := b.Config().Get("emojy.path", "emojy")
	baseURL := b.Config().Get("emojy.baseURL", "/emojy/file")
	c := Cowboy{
		b:            b,
		c:            b.Config(),
		emojyPath:    emojyPath,
		baseEmojyURL: baseURL,
	}
	c.register()
	c.registerWeb()
	return &c
}

func (p *Cowboy) register() {
	tbl := bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^:cowboy_(?P<what>.+):$`),
			Handler: func(r bot.Request) bool {
				p.makeCowboy(r)
				return true
			},
		},
	}
	p.b.RegisterTable(p, tbl)
}

func (p *Cowboy) makeCowboy(r bot.Request) {
	log.Debug().Msgf("makeCowboy: %s", r.Values["what"])
	base := p.c.Get("baseURL", "http://127.0.0.1:1337")
	u := base + "/cowboy/img/" + r.Values["what"]
	p.b.Send(r.Conn, bot.Delete, r.Msg.Channel, r.Msg.ID)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "", bot.ImageAttachment{
		URL:    u,
		AltTxt: fmt.Sprintf("%s: %s", r.Msg.User.Name, r.Msg.Body),
		Width:  64,
		Height: 64,
	})
}
