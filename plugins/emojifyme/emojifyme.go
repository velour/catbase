// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package emojifyme

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type EmojifyMePlugin struct {
	Bot         bot.Bot
	GotBotEmoji bool
	Emoji       map[string]string
}

func New(bot bot.Bot) *EmojifyMePlugin {
	resp, err := http.Get("https://raw.githubusercontent.com/github/gemoji/master/db/emoji.json")
	if err != nil {
		log.Fatalf("Error generic emoji list: %s", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("Error generic emoji list body: %s", err)
	}

	type Emoji struct {
		Aliases []string `json:"aliases"`
	}

	var emoji []Emoji
	err = json.Unmarshal(body, &emoji)
	if err != nil {
		log.Fatalf("Error parsing emoji list: %s", err)
	}

	emojiMap := map[string]string{}
	for _, e := range emoji {
		for _, alias := range e.Aliases {
			emojiMap[alias] = alias
		}
	}

	return &EmojifyMePlugin{
		Bot:         bot,
		GotBotEmoji: false,
		Emoji:       emojiMap,
	}
}

func (p *EmojifyMePlugin) Message(message msg.Message) bool {
	if !p.GotBotEmoji {
		p.GotBotEmoji = true
		emojiMap := p.Bot.GetEmojiList()
		for e := range emojiMap {
			p.Emoji[e] = e
		}
	}

	inertTokens := p.Bot.Config().GetArray("Emojify.Scoreless")
	emojied := 0.0
	emojys := []string{}
	msg := strings.Replace(strings.ToLower(message.Body), "_", " ", -1)
	for k, v := range p.Emoji {
		k = strings.Replace(k, "_", " ", -1)
		candidates := []string{
			k + "es",
			k + "s",
		}
		for _, c := range candidates {
			if strings.Contains(msg, " "+c+" ") ||
				strings.HasPrefix(msg, c) ||
				strings.HasSuffix(msg, c) {
				emojys = append(emojys, v)
				if !stringsContain(inertTokens, k) || len(k) <= 2 {
					emojied++
				}
			}
		}
	}

	if emojied > 0 && rand.Float64() <= p.Bot.Config().GetFloat64("Emojify.Chance")*emojied {
		for _, e := range emojys {
			p.Bot.React(message.Channel, e, message)
		}
		return true
	}
	return false
}

func (p *EmojifyMePlugin) Help(channel string, parts []string) {

}

func (p *EmojifyMePlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *EmojifyMePlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *EmojifyMePlugin) RegisterWeb() *string {
	return nil
}

func (p *EmojifyMePlugin) ReplyMessage(message msg.Message, identifier string) bool { return false }

func stringsContain(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
