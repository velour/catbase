package emojy

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/connectors/discord"
)

func (p *EmojyPlugin) registerCmds(d *discord.Discord) {
	log.Debug().Msg("About to register some startup commands")
	cmd := discordgo.ApplicationCommand{
		Name:        "emojy",
		Description: "swap in an emojy",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "emojy",
				Description: "which emojy you want swapped in",
				Required:    true,
			},
		},
	}
	if err := d.RegisterSlashCmd(cmd, p.overlayCB); err != nil {
		log.Error().Err(err).Msg("could not register emojy command")
	}
}

func (p *EmojyPlugin) overlayCB(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lastEmojy := p.c.Get("emojy.lastEmojy", "rust")
	list := map[string]string{}
	var err error
	msg := "hello, user"

	name := i.ApplicationCommandData().Options[0].StringValue()

	if ok, _, _, err := p.isKnownEmojy(name); !ok || err != nil {
		msg = "I could not find your emojy"
		goto resp
	}

	err = p.RmEmojy(p.b.DefaultConnector(), lastEmojy)
	if err != nil {
		msg = err.Error()
		goto resp
	}

	err = p.UploadEmojyInCache(p.b.DefaultConnector(), name)
	if err != nil {
		msg = err.Error()
		goto resp
	}

	p.c.Set("emojy.lastEmojy", name)

	list = InvertEmojyList(p.b.DefaultConnector().GetEmojiList(true))
	msg = fmt.Sprintf("You replaced %s with a new emojy %s <:%s:%s>, pardner!", lastEmojy, name, name, list[name])

resp:
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   uint64(discordgo.MessageFlagsEphemeral),
		},
	})
}
