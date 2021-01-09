package git

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"gopkg.in/go-playground/webhooks.v5/gitlab"
)

func (p *GitPlugin) gitlabEvent(w http.ResponseWriter, r *http.Request) {
	if p.glhook == nil {
		log.Error().Msg("gitlab hook not initialized")
		w.WriteHeader(500)
		fmt.Fprint(w, "not initialized")
		return
	}
	icon := p.c.Get("gitlab.icon", ":gitlab:")
	payload, err := p.glhook.Parse(r,
		gitlab.PushEvents,
	)
	if err != nil {
		log.Error().Err(err).Msg("unknown event")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event: %s", err)
		return
	}
	msg, repo, owner := " ", "", ""
	switch payload.(type) {
	case gitlab.PushEventPayload:
		push := payload.(gitlab.PushEventPayload)
		repo = push.Repository.Name
		owner = strings.ReplaceAll(push.Project.PathWithNamespace, "/", ".")
		commits := ""
		for _, c := range push.Commits {
			m := strings.Split(c.Message, "\n")[0]
			commits += fmt.Sprintf("%s %s pushed to %s (<%s|%s>) %s\n",
				icon,
				c.Author.Name,
				repo,
				c.URL,
				c.ID[:7],
				m,
			)
		}
		msg += commits
	default:
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown payload: %+v", payload)
		return
	}
	chs := p.c.GetArray(fmt.Sprintf("gitlab.%s.channels", owner), []string{})
	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, msg, bot.UnfurlLinks(false))
	}
}
