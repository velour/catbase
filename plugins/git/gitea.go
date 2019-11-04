package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
)

func (p *GitPlugin) giteaEvent(w http.ResponseWriter, r *http.Request) {
	icon := p.c.Get("gitea.icon", ":tea:")
	evt := GiteaPush{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&evt)
	if err != nil {
		log.Error().Err(err).Msg("could not decode gitea push")
		w.WriteHeader(500)
		fmt.Fprintf(w, " Error parsing event: %s", err)
		return
	}
	org := evt.Repository.Owner.Username
	repo := evt.Repository.Name

	msg := " "
	for _, c := range evt.Commits {
		m := strings.Split(c.Message, "\n")[0]
		msg += fmt.Sprintf("%s %s pushed to %s (<%s|%s>) %s\n",
			icon,
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
