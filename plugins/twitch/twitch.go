package twitch

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type TwitchPlugin struct {
	Bot        bot.Bot
	config     *config.Config
	twitchList map[string]*Twitcher
}

type Twitcher struct {
	name string
	game string
}

type Stream struct {
	Stream struct {
		Game string `json:game`
	} `json:stream,omitempty`
}

func New(bot bot.Bot) *TwitchPlugin {
	p := &TwitchPlugin{
		Bot:        bot,
		config:     bot.Config(),
		twitchList: map[string]*Twitcher{},
	}

	for _, users := range p.config.Twitch.Users {
		for _, twitcherName := range users {
			if _, ok := p.twitchList[twitcherName]; !ok {
				p.twitchList[twitcherName] = &Twitcher{
					name: twitcherName,
					game: "",
				}
			}
		}
	}

	for channel := range p.config.Twitch.Users {
		go p.twitchLoop(channel)
	}

	return p
}

func (p *TwitchPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *TwitchPlugin) RegisterWeb() *string {
	return nil
}

func (p *TwitchPlugin) Message(message msg.Message) bool {
	if strings.ToLower(message.Body) == "twitch status" {
		channel := message.Channel
		if _, ok := p.config.Twitch.Users[channel]; ok {
			for _, twitcherName := range p.config.Twitch.Users[channel] {
				if _, ok = p.twitchList[twitcherName]; ok {
					p.checkTwitch(channel, p.twitchList[twitcherName], true)
				}
			}
		}
		return true
	}

	return false
}

func (p *TwitchPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *TwitchPlugin) LoadData() {

}

func (p *TwitchPlugin) Help(channel string, parts []string) {
	msg := "There's no help for you here."
	p.Bot.SendMessage(channel, msg)
}

func (p *TwitchPlugin) twitchLoop(channel string) {
	frequency := p.config.Twitch.Freq

	log.Println("Checking every ", frequency, " seconds")

	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		// auth := p.getTwitchStreams(channel)
		// if !auth {
		// 	p.Bot.SendMessage(channel, "OAuth for catbase twitch account has expired. Renew it!")
		// }

		for _, twitcherName := range p.config.Twitch.Users[channel] {
			p.checkTwitch(channel, p.twitchList[twitcherName], false)
		}
	}
}

func getRequest(url string) ([]byte, bool) {
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return []byte{}, false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return []byte{}, false
	}
	return body, true
}

func (p *TwitchPlugin) checkTwitch(channel string, twitcher *Twitcher, alwaysPrintStatus bool) {
	baseUrl := "https://api.twitch.tv/kraken/streams/"

	body, ok := getRequest(baseUrl + twitcher.name)
	if !ok {
		return
	}

	var stream Stream
	err := json.Unmarshal(body, &stream)
	if err != nil {
		log.Println(err)
		return
	}

	game := stream.Stream.Game
	if alwaysPrintStatus {
		if game == "" {
			p.Bot.SendMessage(channel, twitcher.name+" is not streaming.")
		} else {
			p.Bot.SendMessage(channel, twitcher.name+" is streaming "+game+".")
		}
	} else if game == "" {
		if twitcher.game != "" {
			p.Bot.SendMessage(channel, twitcher.name+" just stopped streaming.")
		}
		twitcher.game = ""
	} else {
		if twitcher.game != game {
			p.Bot.SendMessage(channel, twitcher.name+" just started streaming "+game+".")
		}
		twitcher.game = game
	}
}

// func (p *TwitchPlugin) getTwitchStreams(channel string) bool {
// 	token := p.Bot.config.Twitch.Token
// 	if token == "" || token == "<Your Token>" {
// 		log.Println("No Twitch token, cannot enable plugin.")
// 		return false
// 	}

// 	access_token := "?access_token=" + token + "&scope=user_subscriptions"
// 	baseUrl := "https://api.twitch.tv/kraken/streams/followed"

// 	body, ok := getRequest(baseUrl + access_token)
// 	if !ok {
// 		return false
// 	}

// 	var subscriptions Subscriptions
// 	err := json.Unmarshal(body, &subscriptions)
// 	if err != nil {
// 		log.Println(err)
// 		return false
// 	}

// 	for _, subscription := range subscriptions.Follows {
// 		twitcherName := subscription.Channel.Name
// 		if _, ok := p.twitchList[twitcherName]; !ok {
// 			p.twitchList[twitcherName] = &Twitcher {
// 				name : twitcherName,
// 				game : "",
// 			}
// 		}
// 	}
// 	return true
// }
