package cowboy

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/velour/catbase/plugins/emojy"
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

var defaultOverlays = map[string]string{"hat": "hat"}

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
			Kind: bot.Startup, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				log.Debug().Msgf("Got bot.Startup")
				switch conn := r.Conn.(type) {
				case *discord.Discord:
					log.Debug().Msg("Found a discord connection")
					p.registerCmds(conn)
				}
				return false
			},
		},
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
	overlays := p.c.GetMap("cowboy.overlays", defaultOverlays)
	hat := overlays["hat"]
	i, err := cowboy(p.emojyPath, p.baseEmojyURL, hat, what)
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
	log.Debug().Msg("About to register some startup commands")
	cmd := discordgo.ApplicationCommand{
		Name:        "cowboy",
		Description: "cowboy-ify an emojy",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emojy",
				Description: "which emojy you want cowboied",
				Required:    true,
			},
		},
	}
	overlays := p.c.GetMap("cowboy.overlays", defaultOverlays)
	hat := overlays["hat"]
	log.Debug().Msgf("Overlay: %s", hat)
	if err := d.RegisterSlashCmd(cmd, p.mkOverlayCB(hat)); err != nil {
		log.Error().Err(err).Msg("could not register cowboy command")
	}

	if p.c.GetInt("cowboy.overlaysEnabled", 0) == 0 {
		return
	}

	cmd = discordgo.ApplicationCommand{
		Name:        "overlay",
		Description: "overlay-ify an emojy",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "overlay",
				Description: "which overlay you want",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emojy",
				Description: "which emojy you want overlaid",
				Required:    true,
			},
		},
	}
	if err := d.RegisterSlashCmd(cmd, p.mkOverlayCB("")); err != nil {
		log.Error().Err(err).Msg("could not register cowboy command")
	}
}

func (p *Cowboy) mkOverlayCB(overlay string) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		lastEmojy := p.c.Get("cowboy.lastEmojy", "rust")
		emojyPlugin := emojy.NewAPI(p.b)
		list := map[string]string{}

		name := i.ApplicationCommandData().Options[0].StringValue()
		if overlay == "" {
			overlay = name
			name = i.ApplicationCommandData().Options[1].StringValue()
		}
		msg := fmt.Sprintf("You asked for %s overlaid by %s", name, overlay)
		log.Debug().Msgf("got a cowboy command for %s overlaid by %s replacing %s",
			name, overlay, lastEmojy)
		prefix := overlay

		newEmojy, err := cowboy(p.emojyPath, p.baseEmojyURL, overlay, name)
		if err != nil {
			msg = err.Error()
			goto resp
		}

		err = emojyPlugin.RmEmojy(p.b.DefaultConnector(), lastEmojy)
		if err != nil {
			msg = err.Error()
			goto resp
		}

		if overlay == "hat" {
			prefix = "cowboy"
		}
		name = emojy.SanitizeName(prefix + "_" + name)
		err = emojyPlugin.UploadEmojyImage(p.b.DefaultConnector(), name, newEmojy)
		if err != nil {
			msg = err.Error()
			goto resp
		}

		p.c.Set("cowboy.lastEmojy", name)

		list = p.b.DefaultConnector().GetEmojiList(true)
		for k, v := range list {
			if v == name {
				msg = fmt.Sprintf("You replaced %s with a new emojy %s <:%s:%s>, pardner!", lastEmojy, name, name, k)
				goto resp
			}
		}
		msg = "Something probably went wrong. I don't see your new emojy, pardner."

	resp:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   uint64(discordgo.MessageFlagsEphemeral),
			},
		})
	}
}
