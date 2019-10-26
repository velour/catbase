package git

import (
	"fmt"
	"net/http"
	"strings"

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
	return p
}

func (p *GitPlugin) registerWeb() {
	http.HandleFunc("/git/github/event", p.githubEvent)
	http.HandleFunc("/git/github/gitlabEvent", p.gitlabEvent)
	p.b.RegisterWeb("/git", "Git")
}

func (p *GitPlugin) gitlabEvent(w http.ResponseWriter, r *http.Request) {
	if p.glhook == nil {
		log.Error().Msg("gitlab hook not initialized")
		w.WriteHeader(500)
		fmt.Fprint(w, "not initialized")
		return
	}
	payload, err := p.glhook.Parse(r,
		gitlab.PushEvents,
	)
	if err != nil {
		log.Error().Err(err).Msg("unknown event")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event: %s", err)
		return
	}
	msg, repo := "", ""
	switch payload.(type) {
	case gitlab.PushEventPayload:
		push := payload.(gitlab.PushEventPayload)
		repo = push.Repository.Name
		commits := ""
		for _, c := range push.Commits {
			m := strings.Split(c.Message, "\n")[0]
			commits += fmt.Sprintf("%s pushed to %s (<%s|%s>) %s\n",
				c.Author.Name,
				repo,
				c.URL,
				c.ID[:7],
				m,
			)
		}
		msg = commits
	default:
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown payload: %+v", payload)
		return
	}
	chs := p.c.GetArray(fmt.Sprintf("gitlab.%s.channels", repo), []string{})
	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
	}
}

func (p *GitPlugin) githubEvent(w http.ResponseWriter, r *http.Request) {
	if p.ghhook == nil {
		log.Error().Msg("github hook not initialized")
		w.WriteHeader(500)
		fmt.Fprintf(w, "not initialized")
		return
	}
	payload, err := p.ghhook.Parse(r,
		github.PushEvent,
		github.PullRequestEvent,
		github.PingEvent,
	)
	if err != nil {
		log.Error().Err(err).Msg("unknown event")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event: %+v", err)
		return
	}
	msg := ""
	repo := ""
	switch payload.(type) {
	case github.PushPayload:
		push := payload.(github.PushPayload)
		repo = push.Repository.Name
		commits := ""
		for _, c := range push.Commits {
			m := strings.Split(c.Message, "\n")[0]
			commits += fmt.Sprintf("%s pushed to %s (<%s|%s>) %s\n",
				c.Author.Name,
				repo,
				c.URL,
				c.ID[:7],
				m,
			)
		}
		msg = commits
	case github.PullRequestPayload:
		pr := payload.(github.PullRequestPayload)
		if pr.Action != "opened" {
			w.WriteHeader(200)
			fmt.Fprintf(w, "ignoring action %s", pr.Action)
			return
		}
		repo = pr.Repository.Name
		msg = fmt.Sprintf("%s opened new pull request \"%s\" on %s: %s",
			pr.PullRequest.User.Login,
			pr.PullRequest.Title,
			pr.Repository.Name,
			pr.PullRequest.URL,
		)
	case github.PingPayload:
		ping := payload.(github.PingPayload)
		repo = ping.Repository.Name
		msg = fmt.Sprintf("Got a ping request on %s", repo)
	default:
		log.Error().Interface("payload", payload).Msg("unknown event payload")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event payload: %+v", payload)
		return
	}

	chs := p.c.GetArray(fmt.Sprintf("github.%s.channels", repo), []string{})
	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
	}
}
