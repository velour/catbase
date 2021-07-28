package git

import (
	"fmt"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/go-playground/webhooks.v5/github"
	"gopkg.in/go-playground/webhooks.v5/gitlab"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type GitPlugin struct {
	b        bot.Bot
	c        *config.Config
	glhook   *gitlab.Webhook
	ghhook   *github.Webhook
	handlers bot.HandlerTable
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
	p.register()
	return p
}

func validService(service string) bool {
	return service == "gitea" || service == "gitlab" || service == "github"
}

func (p *GitPlugin) register() {
	p.handlers = bot.HandlerTable{
		{Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^regrepo (?P<service>\S+)\.(?P<owner>\S+)\.(?P<repo>\S+)$`),
			Handler: func(r bot.Request) bool {
				service := r.Values["service"]
				owner := r.Values["owner"]
				repo := r.Values["repo"]
				spec := fmt.Sprintf("%s.%s.%s", service, owner, repo)
				if !validService(service) {
					m := "Valid formats are: `service.owner.repo` and valid services are one of gitea, gitlab, or github."
					p.b.Send(r.Conn, bot.Message, r.Msg.Channel, m)
					return true
				}
				chs := p.c.GetArray(spec, []string{})
				for _, ch := range chs {
					if ch == r.Msg.Channel {
						p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "That's already registered here.")
						return true
					}
				}
				chs = append(chs, r.Msg.Channel)
				p.c.SetArray(spec+".channels", chs)
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Registered new repository.")
				return true
			}},
	}
}

func (p *GitPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/gitea/event", p.giteaEvent)
	r.HandleFunc("/github/event", p.githubEvent)
	r.HandleFunc("/gitlab/event", p.gitlabEvent)
	p.b.RegisterWeb(r, "/git")
}
