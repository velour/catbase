package impossible

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type Impossible struct {
	b        bot.Bot
	c        *config.Config
	handlers bot.HandlerTable

	title   string
	content []string
	updated time.Time
}

func New(b bot.Bot) *Impossible {
	i := &Impossible{
		b:       b,
		c:       b.Config(),
		title:   "",
		content: []string{},
		updated: getTodaysMidnight().Add(time.Hour * -24),
	}

	b.Register(i, bot.Help, i.help)
	i.register()

	return i
}

func (p *Impossible) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	p.b.Send(c, bot.Message, message.Channel, "You don't need to do anything. I'll take care of it. But guess what I'm thinking.")
	return true
}

func (p *Impossible) tryRefresh(r bot.Request) (sent bool) {
	if p.updated.Before(time.Now()) {
		if p.title != "" {
			p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("The last impossible wikipedia article was: \"%s\"", p.title))
			sent = true
		}
		for !p.refreshImpossible() {
		}

	}
	return sent
}

func (p *Impossible) register() {
	p.handlers = bot.HandlerTable{
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^hint|clue$`),
			Handler: func(r bot.Request) bool {
				if p.tryRefresh(r) {
					return true
				}
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, p.content[rand.Intn(len(p.content))])
				return true
			}},
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`(?i)^i friggin give up.?$`),
			Handler: func(r bot.Request) bool {
				if p.tryRefresh(r) {
					return true
				}
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("You guessed the last impossible wikipedia article: \"%s\"", p.title))
				for !p.refreshImpossible() {
				}
				return true
			}},
		{Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				if p.tryRefresh(r) {
					return true
				}

				if strings.Contains(strings.ToLower(r.Msg.Body), strings.ToLower(p.title)) {
					p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("You guessed the last impossible wikipedia article: \"%s\"", p.title))
					for !p.refreshImpossible() {
					}
					return true
				}
				return false
			}},
	}
	p.b.RegisterTable(p, p.handlers)
}

func (p *Impossible) refreshImpossible() bool {
	p.updated = getTodaysMidnight()
	resp, err := http.Get("https://en.wikipedia.org/wiki/Special:Random")
	if err != nil {
		log.Fatal().Err(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	titleRegex := regexp.MustCompile(`id="firstHeading"[^>]*(?P<Title>[^<]*)`)
	results := titleRegex.FindStringSubmatch(string(body))
	title := results[1][1:] //remove the leading <

	if title == "" {
		return false
	}

	p.title = title
	p.content = []string{}

	resp, err = http.Get("https://en.wikipedia.org/w/api.php?format=json&action=query&prop=extracts&explaintext&titles=" + url.PathEscape(title))
	if err != nil {
		log.Fatal().Err(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)

	var object map[string]any
	json.Unmarshal([]byte(body), &object)

	pages := object["query"].(map[string]any)["pages"].(map[string]any)
	for _, page := range pages {
		descriptionText := page.(map[string]any)["extract"].(string)
		sentences := strings.Split(strings.ReplaceAll(descriptionText, "\n", " "), ". ")
		for _, sentence := range sentences {
			trimmed := strings.ToLower(strings.TrimSpace(sentence))
			if len(trimmed) == 0 || strings.HasPrefix(trimmed, "==") || len(strings.Split(trimmed, " ")) < 5 {
				continue
			}

			censored := strings.ReplaceAll(trimmed, strings.ToLower(title), "?????")

			p.content = append(p.content, censored)
		}
	}
	return true
}

func getTodaysMidnight() time.Time {
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}
