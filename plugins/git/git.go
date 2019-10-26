package git

import (
	"encoding/json"
	"fmt"
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
	p.b.RegisterWeb("/git", "Git")
}

func (p *GitPlugin) giteaEvent(w http.ResponseWriter, r *http.Request) {
	evt := GiteaPush{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&evt)
	if err != nil {
		log.Error().Err(err).Msg("could not decode gitea push")
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error parsing event: %s", err)
		return
	}
	org := evt.Repository.Owner.Username
	repo := evt.Repository.Name

	msg := ""
	for _, c := range evt.Commits {
		m := strings.Split(c.Message, "\n")[0]
		msg += fmt.Sprintf("%s pushed to %s (<%s|%s>) %s",
			c.Author.Name,
			repo,
			c.URL,
			c.ID[:7],
			m,
		)
	}

	chs := p.c.GetArray(fmt.Sprintf("gitea.%s.%s.channels", org, repo), []string{})
	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
	}
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
	msg, repo, owner := "", "", ""
	switch payload.(type) {
	case gitlab.PushEventPayload:
		push := payload.(gitlab.PushEventPayload)
		repo = push.Repository.Name
		owner = strings.ReplaceAll(push.Project.PathWithNamespace, "/", ".")
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
	chs := p.c.GetArray(fmt.Sprintf("gitlab.%s.channels", owner), []string{})
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
	msg, repo, owner := "", "", ""
	switch payload.(type) {
	case github.PushPayload:
		push := payload.(github.PushPayload)
		repo = push.Repository.Name
		owner = push.Repository.Owner.Login
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
		owner = pr.Repository.Owner.Login
		msg = fmt.Sprintf("%s opened new pull request \"%s\" on %s: %s",
			pr.PullRequest.User.Login,
			pr.PullRequest.Title,
			pr.Repository.Name,
			pr.PullRequest.URL,
		)
	case github.PingPayload:
		ping := payload.(github.PingPayload)
		repo = ping.Repository.Name
		owner = ping.Repository.Owner.Login
		msg = fmt.Sprintf("Got a ping request on %s", repo)
	default:
		log.Error().Interface("payload", payload).Msg("unknown event payload")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event payload: %+v", payload)
		return
	}

	chs := p.c.GetArray(fmt.Sprintf("github.%s.%s.channels", owner, repo), []string{})
	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg)
	}
}
