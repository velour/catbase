package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
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
	b          bot.Bot
	c          *config.Config
	twitchList map[string]*Twitcher
	tbl        bot.HandlerTable
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
		b:          b,
		c:          b.Config(),
		twitchList: map[string]*Twitcher{},
	}

	for _, ch := range p.c.GetArray("Twitch.Channels", []string{}) {
		for _, twitcherName := range p.c.GetArray("Twitch."+ch+".Users", []string{}) {
			twitcherName = strings.ToLower(twitcherName)
			if _, ok := p.twitchList[twitcherName]; !ok {
				p.twitchList[twitcherName] = &Twitcher{
					name:   twitcherName,
					gameID: "",
				}
			}
		}
		go p.twitchChannelLoop(b.DefaultConnector(), ch)
	}

	go p.twitchAuthLoop(b.DefaultConnector())

	p.register()
	p.registerWeb()

	return p
}

func (p *TwitchPlugin) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/{user}", p.serveStreaming)
	p.b.RegisterWeb(r, "/isstreaming")
}

func (p *TwitchPlugin) serveStreaming(w http.ResponseWriter, r *http.Request) {
	userName := strings.ToLower(chi.URLParam(r, "user"))
	if userName == "" {
		fmt.Fprint(w, "User not found.")
		return
	}

	twitcher := p.twitchList[userName]
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

func (p *TwitchPlugin) register() {
	p.tbl = bot.HandlerTable{
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^twitch status$`),
			HelpText: "Get status of all twitchers",
			Handler:  p.twitchStatus,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:    regexp.MustCompile(`(?i)^is (?P<who>.+) streaming\??$`),
			HelpText: "Check if a specific twitcher is streaming",
			Handler:  p.twitchUserStatus,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^reset twitch$`),
			HelpText: "Reset the twitch templates",
			Handler:  p.resetTwitch,
		},
	}
	p.b.Register(p, bot.Help, p.help)
	p.b.RegisterTable(p, p.tbl)
}

func (p *TwitchPlugin) twitchStatus(r bot.Request) bool {
	channel := r.Msg.Channel
	if users := p.c.GetArray("Twitch."+channel+".Users", []string{}); len(users) > 0 {
		for _, twitcherName := range users {
			twitcherName = strings.ToLower(twitcherName)
			// we could re-add them here instead of needing to restart the bot.
			if t, ok := p.twitchList[twitcherName]; ok {
				err := p.checkTwitch(r.Conn, channel, t, true)
				if err != nil {
					log.Error().Err(err).Msgf("error in checking twitch")
				}
			}
		}
	}
	return true
}

func (p *TwitchPlugin) twitchUserStatus(r bot.Request) bool {
	who := strings.ToLower(r.Values["who"])
	if t, ok := p.twitchList[who]; ok {
		err := p.checkTwitch(r.Conn, r.Msg.Channel, t, true)
		if err != nil {
			log.Error().Err(err).Msgf("error in checking twitch")
			p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I had trouble with that.")
		}
	} else {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I don't know who that is.")
	}
	return true
}

func (p *TwitchPlugin) resetTwitch(r bot.Request) bool {
	p.c.Set("twitch.istpl", isStreamingTplFallback)
	p.c.Set("twitch.nottpl", notStreamingTplFallback)
	p.c.Set("twitch.stoppedtpl", stoppedStreamingTplFallback)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "The Twitch templates have been reset.")
	return true
}

func (p *TwitchPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	msg := "You can set the templates for streams with\n"
	msg += fmt.Sprintf("twitch.istpl (default: %s)\n", isStreamingTplFallback)
	msg += fmt.Sprintf("twitch.nottpl (default: %s)\n", notStreamingTplFallback)
	msg += fmt.Sprintf("twitch.stoppedtpl (default: %s)\n", stoppedStreamingTplFallback)
	msg += "You can reset all messages with `!reset twitch`"
	msg += "And you can ask who is streaming with `!twitch status`"
	p.b.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *TwitchPlugin) twitchAuthLoop(c bot.Connector) {
	frequency := p.c.GetInt("Twitch.AuthFreq", 60*60)
	cid := p.c.Get("twitch.clientid", "")
	secret := p.c.Get("twitch.secret", "")
	if cid == "" || secret == "" {
		log.Info().Msgf("Disabling twitch autoauth.")
		return
	}

	log.Info().Msgf("Checking auth every %d seconds", frequency)

	if err := p.validateCredentials(); err != nil {
		log.Error().Err(err).Msgf("error checking twitch validity")
	}

	for {
		select {
		case <-time.After(time.Duration(frequency) * time.Second):
			if err := p.validateCredentials(); err != nil {
				log.Error().Err(err).Msgf("error checking twitch validity")
			}
		}
	}
}

func (p *TwitchPlugin) twitchChannelLoop(c bot.Connector, channel string) {
	frequency := p.c.GetInt("Twitch.Freq", 60)
	if p.c.Get("twitch.clientid", "") == "" || p.c.Get("twitch.secret", "") == "" {
		log.Info().Msgf("Disabling twitch autochecking.")
		return
	}

	log.Info().Msgf("Checking channels every %d seconds", frequency)

	for {
		time.Sleep(time.Duration(frequency) * time.Second)

		for _, twitcherName := range p.c.GetArray("Twitch."+channel+".Users", []string{}) {
			twitcherName = strings.ToLower(twitcherName)
			if err := p.checkTwitch(c, channel, p.twitchList[twitcherName], false); err != nil {
				log.Error().Err(err).Msgf("error in twitch loop")
			}
		}
	}
}

func getRequest(url, clientID, token string) ([]byte, int, bool) {
	bearer := fmt.Sprintf("Bearer %s", token)
	var body []byte
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		goto errCase
	}

	req.Header.Add("Client-ID", clientID)
	req.Header.Add("Authorization", bearer)

	resp, err = client.Do(req)
	if err != nil {
		goto errCase
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		goto errCase
	}
	return body, resp.StatusCode, true

errCase:
	log.Error().Err(err)
	return []byte{}, resp.StatusCode, false
}

func (p *TwitchPlugin) checkTwitch(c bot.Connector, channel string, twitcher *Twitcher, alwaysPrintStatus bool) error {
	baseURL, err := url.Parse("https://api.twitch.tv/helix/streams")
	if err != nil {
		err := fmt.Errorf("error parsing twitch stream URL")
		log.Error().Msg(err.Error())
		return err
	}

	query := baseURL.Query()
	query.Add("user_login", twitcher.name)

	baseURL.RawQuery = query.Encode()

	cid := p.c.Get("twitch.clientid", "")
	token := p.c.Get("twitch.token", "")
	if cid == token && cid == "" {
		log.Info().Msgf("Twitch plugin not enabled.")
		return nil
	}

	body, status, ok := getRequest(baseURL.String(), cid, token)
	if !ok {
		return fmt.Errorf("got status %d: %s", status, string(body))
	}

	var s stream
	err = json.Unmarshal(body, &s)
	if err != nil {
		log.Error().Err(err).Msgf("error reading twitch data")
		return err
	}

	games := s.Data
	gameID, title := "", ""
	if len(games) > 0 {
		gameID = games[0].GameID
		if gameID == "" {
			gameID = "unknown"
		}
		title = games[0].Title
	}

	notStreamingTpl := p.c.Get("Twitch.NotTpl", notStreamingTplFallback)
	isStreamingTpl := p.c.Get("Twitch.IsTpl", isStreamingTplFallback)
	stoppedStreamingTpl := p.c.Get("Twitch.StoppedTpl", stoppedStreamingTplFallback)
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
				p.b.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("notStreaming").Parse(notStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.b.Send(c, bot.Message, channel, buf.String())
		} else {
			t, err := template.New("isStreaming").Parse(isStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.b.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("isStreaming").Parse(isStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.b.Send(c, bot.Message, channel, buf.String())
		}
	} else if gameID == "" {
		if twitcher.gameID != "" {
			t, err := template.New("stoppedStreaming").Parse(stoppedStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.b.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("stoppedStreaming").Parse(stoppedStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.b.Send(c, bot.Message, channel, buf.String())
		}
		twitcher.gameID = ""
	} else {
		if twitcher.gameID != gameID {
			t, err := template.New("isStreaming").Parse(isStreamingTpl)
			if err != nil {
				log.Error().Err(err)
				p.b.Send(c, bot.Message, channel, err)
				t = template.Must(template.New("isStreaming").Parse(isStreamingTplFallback))
			}
			t.Execute(&buf, info)
			p.b.Send(c, bot.Message, channel, buf.String())
		}
		twitcher.gameID = gameID
	}
	return nil
}

func (p *TwitchPlugin) validateCredentials() error {
	cid := p.c.Get("twitch.clientid", "")
	token := p.c.Get("twitch.token", "")
	if token == "" {
		return p.reAuthenticate()
	}
	_, status, ok := getRequest("https://id.twitch.tv/oauth2/validate", cid, token)
	if !ok || status == http.StatusUnauthorized {
		return p.reAuthenticate()
	}
	log.Debug().Msgf("checked credentials and they were valid")
	return nil
}

func (p *TwitchPlugin) reAuthenticate() error {
	cid := p.c.Get("twitch.clientid", "")
	secret := p.c.Get("twitch.secret", "")
	if cid == "" || secret == "" {
		return fmt.Errorf("could not request a new token without config values set")
	}
	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", url.Values{
		"client_id":     {cid},
		"client_secret": {secret},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	credentials := struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}{}
	err = json.Unmarshal(body, &credentials)
	log.Debug().Int("expires", credentials.ExpiresIn).Msgf("setting new twitch token")
	return p.c.RegisterSecret("twitch.token", credentials.AccessToken)
}
