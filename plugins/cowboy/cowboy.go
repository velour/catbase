package cowboy

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/velour/catbase/plugins/emojy"
	"regexp"
	"strings"

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
				switch conn := r.Conn.(type) {
				case *discord.Discord:
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
	overlays := p.c.GetMap("cowboy.overlays", defaultOverlays)
	hat := overlays["hat"]
	what := r.Values["what"]
	_, err := p.mkEmojy(what, hat)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Couldn't cowboify that, pardner.")
		return
	}
	p.b.Send(r.Conn, bot.Delete, r.Msg.Channel, r.Msg.ID)
	e := ":cowboy_" + what + ":"

	switch c := r.Conn.(type) {
	case *discord.Discord:
		list := emojy.InvertEmojyList(c.GetEmojiList(true))
		e = strings.Trim(e, ":")
		e = fmt.Sprintf("<:%s:%s>", e, list[e])
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, r.Msg.User.Name+":")
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, e)
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

func (p *Cowboy) mkEmojy(name, overlay string) (string, error) {
	lastEmojy := p.c.Get("cowboy.lastEmojy", "rust")
	emojyPlugin := emojy.NewAPI(p.b)
	list := map[string]string{}

	msg := fmt.Sprintf("You asked for %s overlaid by %s", name, overlay)
	log.Debug().Msgf("got a cowboy command for %s overlaid by %s replacing %s",
		name, overlay, lastEmojy)
	prefix := overlay

	newEmojy, err := cowboy(p.emojyPath, p.baseEmojyURL, overlay, name)
	if err != nil {
		return "", err
	}

	err = emojyPlugin.RmEmojy(p.b.DefaultConnector(), lastEmojy)
	if err != nil {
		return "", err
	}

	if overlay == "hat" {
		prefix = "cowboy"
	}
	name = emojy.SanitizeName(prefix + "_" + name)
	err = emojyPlugin.UploadEmojyImage(p.b.DefaultConnector(), name, newEmojy)
	if err != nil {
		return "", err
	}

	p.c.Set("cowboy.lastEmojy", name)

	list = emojy.InvertEmojyList(p.b.DefaultConnector().GetEmojiList(true))
	msg = fmt.Sprintf("You replaced %s with a new emojy %s <:%s:%s>, pardner!",
		lastEmojy, name, name, list[name])

	return msg, nil
}

func (p *Cowboy) mkOverlayCB(overlay string) func(s *discordgo.Session, i *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		name := i.ApplicationCommandData().Options[0].StringValue()
		if overlay == "" {
			overlay = name
			name = i.ApplicationCommandData().Options[1].StringValue()
		}

		msg, err := p.mkEmojy(name, overlay)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: err.Error(),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
