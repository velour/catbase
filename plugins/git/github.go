package git

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"gopkg.in/go-playground/webhooks.v5/github"
)

func (p *GitPlugin) githubEvent(w http.ResponseWriter, r *http.Request) {
	if p.ghhook == nil {
		log.Error().Msg("github hook not initialized")
		w.WriteHeader(500)
		fmt.Fprintf(w, "not initialized")
		return
	}
	icon := p.c.Get("github.icon", ":octocat:")
	payload, err := p.ghhook.Parse(r,
		github.PushEvent,
		github.PullRequestEvent,
		github.PingEvent,
		github.IssuesEvent,
		github.IssueCommentEvent,
	)
	if err != nil {
		log.Error().Err(err).Msg("unknown event")
		w.WriteHeader(500)
		fmt.Fprintf(w, "unknown event: %+v", err)
		return
	}
	msg, repo, owner := " ", "", ""
	switch payload.(type) {
	case github.PushPayload:
		push := payload.(github.PushPayload)
		repo = push.Repository.Name
		owner = push.Repository.Owner.Login
		commits := ""
		filterBranch := p.c.Get(fmt.Sprintf("github.%s.%s.branches", owner, repo), "*")
		if len(push.Commits) == 0 {
			log.Debug().Msg("GitHub sent an empty changeset")
			return
		}
		if filterBranch != "*" && !strings.Contains(push.Ref, filterBranch) {
			log.Debug().Msgf("Ignoring GitHub push to %s", push.Ref)
			return
		}
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
	case github.PullRequestPayload:
		pr := payload.(github.PullRequestPayload)
		if pr.Action != "opened" {
			w.WriteHeader(200)
			fmt.Fprintf(w, "ignoring action %s", pr.Action)
			return
		}
		repo = pr.Repository.Name
		owner = pr.Repository.Owner.Login
		msg += fmt.Sprintf("%s %s opened new pull request \"%s\" on %s: %s",
			icon,
			pr.PullRequest.User.Login,
			pr.PullRequest.Title,
			pr.Repository.Name,
			pr.PullRequest.URL,
		)
	case github.PingPayload:
		ping := payload.(github.PingPayload)
		repo = ping.Repository.Name
		owner = ping.Repository.Owner.Login
		msg += fmt.Sprintf("%s Got a ping request on %s", icon, repo)
	case github.IssueCommentPayload:
		cmt := payload.(github.IssueCommentPayload)
		commentEvents := p.c.GetMap("github.commentevents", map[string]string{
			"created": "commented",
			"deleted": "removed their comment",
		})
		action := cmt.Action
		for k, v := range commentEvents {
			if cmt.Action == k {
				action = v
				goto sendCommentEvent
			}
		}
		log.Debug().Msgf("Unknown issue comment event: %s", cmt.Action)
		return
	sendCommentEvent:
		repo = cmt.Repository.Name
		owner = cmt.Comment.User.Login
		msg += fmt.Sprintf("%s %s %s on <%s|%s #%d>",
			icon,
			owner,
			action,
			cmt.Issue.URL,
			cmt.Issue.Title,
			cmt.Issue.Number,
		)
	case github.IssuesPayload:
		issueEvent := payload.(github.IssuesPayload)
		issueEvents := p.c.GetMap("github.issueevents", map[string]string{
			"opened":   "opened",
			"reopened": "reopened",
			"closed":   "closed",
		})
		action := issueEvent.Action
		for k, v := range issueEvents {
			if issueEvent.Action == k {
				action = v
				goto sendIssueEvent
			}
		}
		log.Debug().Msgf("Unknown issue event: %s", issueEvent.Action)
		return
	sendIssueEvent:
		repo = issueEvent.Repository.Name
		owner = issueEvent.Sender.Login
		msg += fmt.Sprintf("%s %s %s issue <%s|%s #%d>",
			icon,
			owner,
			action,
			issueEvent.Issue.URL,
			issueEvent.Issue.Title,
			issueEvent.Issue.Number,
		)
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
