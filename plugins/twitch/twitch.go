package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

const (
	isStreamingTplFallback      = "{{.Name}} is streaming {{.Game}} at {{.URL}}"
	notStreamingTplFallback     = "{{.Name}} is not streaming"
	stoppedStreamingTplFallback = "{{.Name}} just stopped streaming"
)

type TwitchPlugin struct {
	bot        bot.Bot
	config     *config.Config
	twitchList map[string]*Twitcher
}

type Twitcher struct {
	name   string
	gameID string
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
		bot:        b,
		config:     b.Config(),
		twitchList: map[string]*Twitcher{},
	}

	for _, ch := range p.config.GetArray("Twitch.Channels", []string{}) {
		for _, twitcherName := range p.config.GetArray("Twitch."+ch+".Users", []string{}) {
			if _, ok := p.twitchList[twitcherName]; !ok {
				p.twitchList[twitcherName] = &Twitcher{
					name:   twitcherName,
					gameID: "",
				}
			}
		}
		go p.twitchLoop(b.DefaultConnector(), ch)
	}

	b.Register(p, bot.Message, p.message)
	b.Register(p, bot.Help, p.help)
	p.registerWeb()

	return p
}

func (p *TwitchPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/", p.serveStreaming)
	p.bot.RegisterWeb(r, "/isstreaming")
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
	if twitcher.gameID != "" {
		status = "YES."
	}
	context := map[string]any{"Name": twitcher.name, "Status": status}

	t, err := template.New("streaming").Parse(page)
	if err != nil {
		log.Error().Err(err).Msg("Could not parse template!")
		return
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Error().Err(err).Msg("Could not execute template!")
	}
}

func (p *TwitchPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	body := strings.ToLower(message.Body)
	if body == "twitch status" {
		channel := message.Channel
		if users := p.config.GetArray("Twitch."+channel+".Users", []string{}); len(users) > 0 {
			for _, twitcherName := range users {
				if _, ok := p.twitchList[twitcherName]; ok {
					p.checkTwitch(c, channel, p.twitchList[twitcherName], true)
				}
			}
		}
		return true
	} else if body == "reset twitch" {
		p.config.Set("twitch.istpl", isStreamingTplFallback)
		p.config.Set("twitch.nottpl", notStreamingTplFallback)
		p.config.Set("twitch.stoppedtpl", stoppedStreamingTplFallback)
	}

	return false
}

func (p *TwitchPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	msg := "You can set the templates for streams with\n"
	msg += fmt.Sprintf("twitch.istpl (default: %s)\n", isStreamingTplFallback)
	msg += fmt.Sprintf("twitch.nottpl (default: %s)\n", notStreamingTplFallback)
	msg += fmt.Sprintf("twitch.stoppedtpl (default: %s)\n", stoppedStreamingTplFallback)
	msg += "You can reset all messages with `!reset twitch`"
	msg += "And you can ask who is streaming with `!twitch status`"
	p.bot.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *TwitchPlugin) twitchLoop(c bot.Connector, channel string) {
	frequency := p.config.GetInt("Twitch.Freq", 60)
	if p.config.Get("twitch.clientid", "") == "" || p.config.Get("twitch.authorization", "") == "" {
		log.Info().Msgf("Disabling twitch autochecking.")
		return
	}

	log.Info().Msgf("Checking every %d seconds", frequency)

	for {
		time.Sleep(time.Duration(frequency) * time.Second)

		for _, twitcherName := range p.config.GetArray("Twitch."+channel+".Users", []string{}) {
			p.checkTwitch(c, channel, p.twitchList[twitcherName], false)
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
	log.Error().Err(err)
	return []byte{}, false
}

func (p *TwitchPlugin) checkTwitch(c bot.Connector, channel string, twitcher *Twitcher, alwaysPrintStatus bool) {
	baseURL, err := url.Parse("https://api.twitch.tv/helix/streams")
	if err != nil {
		log.Error().Msg("Error parsing twitch stream URL")
		return
	}

	query := baseURL.Query()
	query.Add("user_login", twitcher.name)

	baseURL.RawQuery = query.Encode()

	cid := p.config.Get("Twitch.ClientID", "")
	auth := p.config.Get("Twitch.Authorization", "")
	if cid == auth && cid == "" {
		log.Info().Msgf("Twitch plugin not enabled.")
		return
	}

	body, ok := getRequest(baseURL.String(), cid, auth)
	if !ok {
		return
	}

	var s stream
	err = json.Unmarshal(body, &s)
	if err != nil {
		log.Error().Err(err)
		return
	}

	games := s.Data
	gameID, title := "", ""
	if len(games) > 0 {
		gameID = games[0].GameID
		title = games[0].Title
	}

	notStreamingTpl := p.config.Get("Twitch.NotTpl", notStreamingTplFallback)
	isStreamingTpl := p.config.Get("Twitch.IsTpl", isStreamingTplFallback)
	stoppedStreamingTpl := p.config.Get("Twitch.StoppedTpl", stoppedStreamingTplFallback)
	buf := bytes.Buffer{}

	info := struct {
		Name string
		Game string
		URL  string
	}{
		twitcher.name,
		title,
		twitcher.URL(),
	}

	if alwaysPrintStatus {
		if gameID == "" {
			t, err := template.New("notStreaming").Parse(notStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.bot.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("notStreaming").Parse(notStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.bot.Send(c, bot.Message, channel, buf.String())
		} else {
			t, err := template.New("isStreaming").Parse(isStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.bot.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("isStreaming").Parse(isStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.bot.Send(c, bot.Message, channel, buf.String())
		}
	} else if gameID == "" {
		if twitcher.gameID != "" {
			t, err := template.New("stoppedStreaming").Parse(stoppedStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.bot.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("stoppedStreaming").Parse(stoppedStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.bot.Send(c, bot.Message, channel, buf.String())
		}
		twitcher.gameID = ""
	} else {
		if twitcher.gameID != gameID {
			t, err := template.New("isStreaming").Parse(isStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.bot.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("isStreaming").Parse(isStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.bot.Send(c, bot.Message, channel, buf.String())
		}
		twitcher.gameID = gameID
	}
}
