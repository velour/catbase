// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package emojifyme

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type EmojifyMePlugin struct {
	Bot         bot.Bot
	GotBotEmoji bool
	Emoji       map[string]string
}

func New(bot bot.Bot) *EmojifyMePlugin {
	rand.Seed(time.Now().Unix())

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
		Aliases []string `json:aliases`
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

	emojied := 0.0
	tokens := strings.Fields(strings.ToLower(message.Body))
	for i, token := range tokens {
		if _, ok := p.Emoji[token]; ok {
			if token != "a" && token != "it" {
				emojied++
			}
			tokens[i] = ":" + token + ":"
		} else if strings.HasSuffix(token, "s") {
			//Check to see if we can strip the trailing "es" off and get an emoji
			temp := strings.TrimSuffix(token, "s")
			if _, ok := p.Emoji[temp]; ok {
				if token != "a" && token != "it" {
					emojied++
				}
				tokens[i] = ":" + temp + ":s"
			} else if strings.HasSuffix(token, "es") {
				//Check to see if we can strip the trailing "es" off and get an emoji
				temp := strings.TrimSuffix(token, "es")
				if _, ok := p.Emoji[temp]; ok {
					if token != "a" && token != "it" {
						emojied++
					}
					tokens[i] = ":" + temp + ":es"
				}
			}
		}
	}
	if emojied > 0 && rand.Float64() <= p.Bot.Config().Emojify.Chance*emojied {
		modified := strings.Join(tokens, " ")
		p.Bot.SendMessage(message.Channel, modified)
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
