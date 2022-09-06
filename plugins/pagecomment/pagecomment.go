package pagecomment

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/connectors/discord"
	"net/http"
	"regexp"
	"strings"
)

type PageComment struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *PageComment {
	p := &PageComment{
		b: b,
		c: b.Config(),
	}
	p.register()
	return p
}

func (p *PageComment) register() {
	p.b.RegisterTable(p, bot.HandlerTable{
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
		{Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^url (?P<url>\S+) (?P<comment>.+)`),
			HelpText: "Comment on a URL", Handler: p.handleURLReq},
	})
}

func (p *PageComment) handleURLReq(r bot.Request) bool {
	fullText := r.Msg.Body
	fullComment := fullText[strings.Index(fullText, r.Values["comment"]):]
	u := r.Values["url"]
	if strings.HasPrefix(u, "<") && strings.HasSuffix(u, ">") {
		u = u[1 : len(u)-1]
	}
	msg := p.handleURL(u, fullComment, r.Msg.User.Name)
	p.b.Send(r.Conn, bot.Delete, r.Msg.Channel, r.Msg.ID)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}

func (p *PageComment) handleURLCmd(conn bot.Connector) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		u := i.ApplicationCommandData().Options[0].StringValue()
		cmt := i.ApplicationCommandData().Options[1].StringValue()
		who := i.Member.User.Username
		profile, err := conn.Profile(i.Member.User.ID)
		if err == nil {
			who = profile.Name
		}
		msg := p.handleURL(u, cmt, who)
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}
	}
}

func (p *PageComment) handleURL(u, cmt, who string) string {
	req, err := http.Get(u)
	if err != nil {
		return "Couldn't get that URL"
	}
	doc, err := goquery.NewDocumentFromReader(req.Body)
	if err != nil {
		return "Couldn't parse that URL"
	}
	wait := make(chan string, 1)
	doc.Find("title").First().Each(func(i int, s *goquery.Selection) {
		wait <- fmt.Sprintf("> %s\n%s: %s\n(<%s>)", s.Text(), who, cmt, u)
	})
	return <-wait
}

func (p *PageComment) registerCmds(d *discord.Discord) {
	cmd := discordgo.ApplicationCommand{
		Name:        "url",
		Description: "comment on a URL with its title",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "url",
				Description: "What URL would you like",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "comment",
				Description: "Your comment",
				Required:    true,
			},
		},
	}
	if err := d.RegisterSlashCmd(cmd, p.handleURLCmd(d)); err != nil {
		log.Error().Err(err).Msg("could not register emojy command")
	}
}
