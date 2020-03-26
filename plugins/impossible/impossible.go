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
	b bot.Bot
	c *config.Config

	title   string
	content []string
	updated time.Time
	testing bool
}

func New(b bot.Bot) *Impossible {
	i := &Impossible{
		b:       b,
		c:       b.Config(),
		title:   "",
		content: []string{},
		updated: getTodaysMidnight().Add(time.Hour * -24),
		testing: false,
	}

	b.Register(i, bot.Help, i.help)
	b.Register(i, bot.Message, i.message)

	return i
}

func newTesting(b bot.Bot) *Impossible {
	i := &Impossible{
		b:       b,
		c:       b.Config(),
		title:   "",
		content: []string{},
		updated: getTodaysMidnight().Add(time.Hour * -24),
		testing: true,
	}

	b.Register(i, bot.Help, i.help)
	b.Register(i, bot.Message, i.message)

	return i
}

func (p *Impossible) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.b.Send(c, bot.Message, message.Channel, "You don't need to do anything. I'll take care of it. But guess what I'm thinking.")
	return true
}

func (p *Impossible) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	messaged := false
	if p.updated.Before(time.Now()) {
		if p.title != "" {
			p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("The last impossible wikipedia article was: \"%s\"", p.title))
			messaged = true
		}
		for !p.refreshImpossible() {
		}

		if p.testing {
			p.b.Send(c, bot.Message, message.Channel, p.title)
			messaged = true
		}
	}

	lowercase := strings.ToLower(message.Body)
	if lowercase == "hint" || lowercase == "clue" {
		messaged = true
		p.b.Send(c, bot.Message, message.Channel, p.content[rand.Intn(len(p.content))])
	} else if strings.Contains(lowercase, strings.ToLower(p.title)) {
		messaged = true
		p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("You guessed the last impossible wikipedia article: \"%s\"", p.title))
		for !p.refreshImpossible() {
		}
	} else if strings.Contains(lowercase, "i friggin give up") {
		messaged = true
		p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("You're a failure the last impossible wikipedia article: \"%s\"", p.title))
		for !p.refreshImpossible() {
		}
	}

	return messaged
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

	var object map[string]interface{}
	json.Unmarshal([]byte(body), &object)

	pages := object["query"].(map[string]interface{})["pages"].(map[string]interface{})
	for _, page := range pages {
		descriptionText := page.(map[string]interface{})["extract"].(string)
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
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 24, 0, 0, 0, now.Location())
}
