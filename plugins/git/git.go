package git

import (
	"net/http"
	"strings"

	"github.com/velour/catbase/bot/msg"

	"github.com/rs/zerolog/log"
	"gopkg.in/go-playground/webhooks.v5/github"
	"gopkg.in/go-playground/webhooks.v5/gitlab"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type GitPlugin struct {
	b      bot.Bot
	c      *config.Config
	glhook *gitlab.Webhook
	ghhook *github.Webhook
}

func New(b bot.Bot) *GitPlugin {
	glhook, err := gitlab.New()
	if err != nil {
		log.Error().Err(err).Msg("could not initialize hook")
		glhook = nil
	}
	ghhook, err := github.New()
	if err != nil {
		log.Error().Err(err).Msg("could not initialize hook")
		ghhook = nil
	}
	p := &GitPlugin{
		b:      b,
		c:      b.Config(),
		glhook: glhook,
		ghhook: ghhook,
	}
	p.registerWeb()
	b.Register(p, bot.Message, p.message)
	return p
}

func validService(service string) bool {
	return service == "gitea" || service == "gitlab" || service == "github"
}

func (p *GitPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	body := message.Body
	lower := strings.ToLower(body)
	parts := strings.Split(lower, " ")

	if !strings.HasPrefix(lower, "regrepo") || len(parts) != 2 {
		return false
	}
	u := parts[1]
	u = strings.ReplaceAll(u, "/", ".")
	uparts := strings.Split(u, ".")
	if len(uparts) != 3 || !validService(uparts[0]) {
		m := "Valid formats are: `service.owner.repo` and valid services are one of gitea, gitlab, or github."
		p.b.Send(c, bot.Message, message.Channel, m)
		return true
	}
	chs := p.c.GetArray(u+".channels", []string{})
	for _, ch := range chs {
		if ch == message.Channel {
			p.b.Send(c, bot.Message, message.Channel, "That's already registered here.")
			return true
		}
	}
	chs = append(chs, message.Channel)
	p.c.SetArray(u+".channels", chs)
	p.b.Send(c, bot.Message, message.Channel, "Registered new repository.")
	return true
}

func (p *GitPlugin) registerWeb() {
	http.HandleFunc("/git/gitea/event", p.giteaEvent)
	http.HandleFunc("/git/github/event", p.githubEvent)
	http.HandleFunc("/git/gitlab/event", p.gitlabEvent)
}
