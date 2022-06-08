// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package emojifyme

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
)

type EmojifyMePlugin struct {
	Bot         bot.Bot
	GotBotEmoji bool
	Emoji       map[string]string
}

func New(b bot.Bot) *EmojifyMePlugin {
	resp, err := http.Get("https://raw.githubusercontent.com/github/gemoji/master/db/emoji.json")
	if err != nil {
		log.Fatal().Err(err).Msg("Error generic emoji list")
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal().Err(err).Msg("Error generic emoji list body")
	}

	type Emoji struct {
		Aliases []string `json:"aliases"`
	}

	var emoji []Emoji
	err = json.Unmarshal(body, &emoji)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing emoji list")
	}

	emojiMap := map[string]string{}
	for _, e := range emoji {
		for _, alias := range e.Aliases {
			emojiMap[alias] = alias
		}
	}

	ep := &EmojifyMePlugin{
		Bot:         b,
		GotBotEmoji: false,
		Emoji:       emojiMap,
	}
	b.RegisterRegex(ep, bot.Message, regexp.MustCompile(`.*`), ep.message)
	return ep
}

func (p *EmojifyMePlugin) message(r bot.Request) bool {
	c := r.Conn
	message := r.Msg
	if !p.GotBotEmoji {
		p.GotBotEmoji = true
		emojiMap := p.Bot.GetEmojiList(false)
		for e := range emojiMap {
			p.Emoji[e] = e
		}
	}

	inertTokens := p.Bot.Config().GetArray("Emojify.Scoreless", []string{})
	emojied := 0.0
	emojys := []string{}
	msg := strings.Replace(strings.ToLower(message.Body), "_", " ", -1)
	for k, v := range p.Emoji {
		k = strings.Replace(k, "_", " ", -1)
		candidates := []string{
			k,
			k + "es",
			k + "s",
		}
		for _, c := range candidates {
			if strings.Contains(msg, " "+c+" ") ||
				strings.HasPrefix(msg, c+" ") ||
				strings.HasSuffix(msg, " "+c) ||
				msg == c {
				emojys = append(emojys, v)
				if !stringsContain(inertTokens, k) && len(v) > 2 {
					emojied += 1
				}
			}
		}
	}

	if emojied > 0 && rand.Float64() <= p.Bot.Config().GetFloat64("Emojify.Chance", 0.02)*emojied {
		for _, e := range emojys {
			p.Bot.Send(c, bot.Reaction, message.Channel, e, message)
		}
		return false
	}
	return false
}

func stringsContain(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
