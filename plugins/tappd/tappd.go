package tappd

import (
	"bytes"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/connectors/discord"
	"image"
	"os"
	"path"
	"regexp"
	"time"
)

type Tappd struct {
	b        bot.Bot
	c        *config.Config
	imageMap map[string]imageInfo
}

type imageInfo struct {
	ID       string
	SrcURL   string
	BotURL   string
	Img      image.Image
	Repr     []byte
	W        int
	H        int
	MimeType string
	FileName string
}

func New(b bot.Bot) *Tappd {
	t := &Tappd{
		b:        b,
		c:        b.Config(),
		imageMap: make(map[string]imageInfo),
	}
	t.register()
	t.registerWeb()
	t.mkDB()
	return t
}

func (p *Tappd) mkDB() error {
	db := p.b.DB()
	tx, err := db.Beginx()
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`create table if not exists tappd (
		id integer primary key autoincrement,
		who string,
		channel string,
		message string,
		ts datetime
	);`)
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *Tappd) log(who, channel, message string) error {
	db := p.b.DB()
	tx, err := db.Beginx()
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`insert into tappd (who, channel, message, ts) values (?, ?, ? ,?)`,
		who, channel, message, time.Now())
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *Tappd) registerDiscord(d *discord.Discord) {
	cmd := discordgo.ApplicationCommand{
		Name:        "tap",
		Description: "tappd a beer in",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "image",
				Description: "Picture that beer, but on Discord",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "comment",
				Description: "Comment on that beer",
				Required:    true,
			},
		},
	}
	if err := d.RegisterSlashCmd(cmd, p.tap); err != nil {
		log.Error().Err(err).Msgf("could not register")
	}
}

func (p *Tappd) tap(s *discordgo.Session, i *discordgo.InteractionCreate) {
	who := i.Interaction.Member.Nick
	channel := i.Interaction.ChannelID
	shortMsg := i.ApplicationCommandData().Options[1].StringValue()
	longMsg := fmt.Sprintf("%s checked in: %s",
		i.Interaction.Member.Nick, shortMsg)
	attachID := i.ApplicationCommandData().Options[0].Value.(string)
	attach := i.ApplicationCommandData().Resolved.Attachments[attachID]
	spec := defaultSpec()
	spec.text = shortMsg
	info, err := p.getAndOverlay(attachID, attach.URL, []textSpec{spec})
	if err != nil {
		log.Error().Err(err).Msgf("error with interaction")
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error getting the image",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Error().Err(err).Msgf("error with interaction")
		}
		return
	}
	embeds := []*discordgo.MessageEmbed{{
		Description: longMsg,
		Image: &discordgo.MessageEmbedImage{
			URL:    info.BotURL,
			Width:  info.W,
			Height: info.H,
		},
	}}
	files := []*discordgo.File{{
		Name:        info.FileName,
		ContentType: info.MimeType,
		Reader:      bytes.NewBuffer(info.Repr),
	}}
	content := info.BotURL
	// Yes, the configs are all stringly typed. Get over it.
	useEmbed := p.c.GetBool("tappd.embed", false)
	if useEmbed {
		files = nil
	} else {
		embeds = nil
		content = ""
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:  embeds,
			Files:   files,
			Content: content,
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("error with interaction")
		return
	}
	err = p.log(who, channel, shortMsg)
	if err != nil {
		log.Error().Err(err).Msgf("error recording tap")
	}
	imgPath := p.c.Get("tappd.imagepath", "tappdimg")
	err = os.MkdirAll(imgPath, 0775)
	if err != nil {
		log.Error().Err(err).Msgf("error creating directory")
		return
	}
	err = os.WriteFile(path.Join(imgPath, info.FileName), info.Repr, 0664)
	if err != nil {
		log.Error().Err(err).Msgf("error writing file")
	}
}

func (p *Tappd) register() {
	ht := bot.HandlerTable{
		{
			Kind: bot.Startup, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				switch conn := r.Conn.(type) {
				case *discord.Discord:
					p.registerDiscord(conn)
				}
				return false
			},
		},
	}
	p.b.RegisterTable(p, ht)
}
