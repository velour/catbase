package cowboy

import (
	"fmt"
	"regexp"

	"github.com/velour/catbase/connectors/discord"

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
	switch conn := b.DefaultConnector().(type) {
	case *discord.Discord:
		c.registerCmds(conn)
	}
	return &c
}

func (p *Cowboy) register() {
	tbl := bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^:cowboy_clear_cache:$`),
			Handler: func(r bot.Request) bool {
				cowboyClearCache()
				p.b.Send(r.Conn, bot.Ephemeral, r.Msg.Channel, r.Msg.User.ID, ":cowboy_cache_cleared:")
				return true
			},
		},
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
	what := r.Values["what"]
	// This'll add the image to the cowboy_cache before discord tries to access it over http
	i, err := cowboy(p.c, p.emojyPath, p.baseEmojyURL, what)
	if err != nil {
		log.Error().Err(err).Msg(":cowboy_fail:")
		p.b.Send(r.Conn, bot.Ephemeral, r.Msg.Channel, r.Msg.User.ID, "Hey cowboy, that image wasn't there.")
		return
	}
	log.Debug().Msgf("makeCowboy: %s", r.Values["what"])
	base := p.c.Get("baseURL", "http://127.0.0.1:1337")
	u := base + "/cowboy/img/" + r.Values["what"]
	p.b.Send(r.Conn, bot.Delete, r.Msg.Channel, r.Msg.ID)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "", bot.ImageAttachment{
		URL:    u,
		AltTxt: fmt.Sprintf("%s: %s", r.Msg.User.Name, r.Msg.Body),
		Width:  i.Bounds().Max.X,
		Height: i.Bounds().Max.Y,
	})
}

func (p *Cowboy) registerCmds(d *discord.Discord) {
	//d.RegisterSlashCmd()
}
