package twitch

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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

func (t Twitcher) URL() string {
	u, _ := url.Parse("https://twitch.tv/")
	u2, _ := url.Parse(t.name)
	return u.ResolveReference(u2).String()
}

type stream struct {
	Data []struct {
		ID           string    `json:"id"`
		UserID       string    `json:"user_id"`
		GameID       string    `json:"game_id"`
		CommunityIds []string  `json:"community_ids"`
		Type         string    `json:"type"`
		Title        string    `json:"title"`
		ViewerCount  int       `json:"viewer_count"`
		StartedAt    time.Time `json:"started_at"`
		Language     string    `json:"language"`
		ThumbnailURL string    `json:"thumbnail_url"`
	} `json:"data"`
	Pagination struct {
		Cursor string `json:"cursor"`
	} `json:"pagination"`
}

func New(b bot.Bot) *TwitchPlugin {
	p := &TwitchPlugin{
		Bot:        b,
		config:     b.Config(),
		twitchList: map[string]*Twitcher{},
	}

	for _, ch := range p.config.GetArray("Twitch.Channels", []string{}) {
		for _, twitcherName := range p.config.GetArray("Twitch."+ch+".Users", []string{}) {
			if _, ok := p.twitchList[twitcherName]; !ok {
				p.twitchList[twitcherName] = &Twitcher{
					name: twitcherName,
					game: "",
				}
			}
		}
		go p.twitchLoop(ch)
	}

	b.Register(p, bot.Message, p.message)
	return p
}

func (p *TwitchPlugin) RegisterWeb() *string {
	http.HandleFunc("/isstreaming/", p.serveStreaming)
	tmp := "/isstreaming"
	return &tmp
}

func (p *TwitchPlugin) serveStreaming(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 3 {
		fmt.Fprint(w, "User not found.")
		return
	}

	twitcher := p.twitchList[pathParts[2]]
	if twitcher == nil {

		fmt.Fprint(w, "User not found.")
		return
	}

	status := "NO."
	if twitcher.game != "" {
		status = "YES."
	}
	context := map[string]interface{}{"Name": twitcher.name, "Status": status}

	t, err := template.New("streaming").Parse(page)
	if err != nil {
		log.Println("Could not parse template!", err)
		return
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Println("Could not execute template!", err)
	}
}

func (p *TwitchPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if strings.ToLower(message.Body) == "twitch status" {
		channel := message.Channel
		if users := p.config.GetArray("Twitch."+channel+".Users", []string{}); len(users) > 0 {
			for _, twitcherName := range users {
				if _, ok := p.twitchList[twitcherName]; ok {
					p.checkTwitch(channel, p.twitchList[twitcherName], true)
				}
			}
		}
		return true
	}

	return false
}

func (p *TwitchPlugin) help(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	msg := "There's no help for you here."
	p.Bot.Send(bot.Message, message.Channel, msg)
	return true
}

func (p *TwitchPlugin) twitchLoop(channel string) {
	frequency := p.config.GetInt("Twitch.Freq", 60)

	log.Println("Checking every ", frequency, " seconds")

	for {
		time.Sleep(time.Duration(frequency) * time.Second)

		for _, twitcherName := range p.config.GetArray("Twitch."+channel+".Users", []string{}) {
			p.checkTwitch(channel, p.twitchList[twitcherName], false)
		}
	}
}

func getRequest(url, clientID, authorization string) ([]byte, bool) {
	var body []byte
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		goto errCase
	}

	req.Header.Add("Client-ID", clientID)
	req.Header.Add("Authorization", authorization)

	resp, err = client.Do(req)
	if err != nil {
		goto errCase
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		goto errCase
	}
	return body, true

errCase:
	log.Println(err)
	return []byte{}, false
}

func (p *TwitchPlugin) checkTwitch(channel string, twitcher *Twitcher, alwaysPrintStatus bool) {
	baseURL, err := url.Parse("https://api.twitch.tv/helix/streams")
	if err != nil {
		log.Println("Error parsing twitch stream URL")
		return
	}

	query := baseURL.Query()
	query.Add("user_login", twitcher.name)

	baseURL.RawQuery = query.Encode()

	cid := p.config.Get("Twitch.ClientID", "")
	auth := p.config.Get("Twitch.Authorization", "")
	if cid == auth && cid == "" {
		log.Println("Twitch plugin not enabled.")
		return
	}

	body, ok := getRequest(baseURL.String(), cid, auth)
	if !ok {
		return
	}

	var s stream
	err = json.Unmarshal(body, &s)
	if err != nil {
		log.Println(err)
		return
	}

	games := s.Data
	game := ""
	if len(games) > 0 {
		game = games[0].Title
	}
	streamWord := p.config.Get("Twitch.StreamWord", "streaming")
	if alwaysPrintStatus {
		if game == "" {
			p.Bot.Send(bot.Message, channel, fmt.Sprintf("%s is not %s.", twitcher.name, streamWord))
		} else {
			p.Bot.Send(bot.Message, channel, fmt.Sprintf("%s is %s %s at %s", twitcher.name, streamWord, game, twitcher.URL()))
		}
	} else if game == "" {
		if twitcher.game != "" {
			p.Bot.Send(bot.Message, channel, fmt.Sprintf("%s just stopped %s.", twitcher.name, streamWord))
		}
		twitcher.game = ""
	} else {
		if twitcher.game != game {
			p.Bot.Send(bot.Message, channel, fmt.Sprintf("%s is %s %s at %s", twitcher.name, streamWord, game, twitcher.URL()))
		}
		twitcher.game = game
	}
}
