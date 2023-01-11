package twitch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nicklaw5/helix"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
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

type Twitch struct {
	b            bot.Bot
	c            *config.Config
	twitchList   map[string]*Twitcher
	tbl          bot.HandlerTable
	ircConnected bool
	irc          *IRC
	ircLock      sync.Mutex
	bridgeMap    map[string]string

	helix *helix.Client
	subs  map[string]bool
}

type Twitcher struct {
	id     string
	gameID string
	online bool

	Name string
	Game string
	URL  string
}

func (t Twitcher) url() string {
	u, _ := url.Parse("https://twitch.tv/")
	u2, _ := url.Parse(t.Name)
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

func New(b bot.Bot) *Twitch {
	p := &Twitch{
		b:          b,
		c:          b.Config(),
		twitchList: map[string]*Twitcher{},
		bridgeMap:  map[string]string{},
	}

	for _, twitcherName := range p.c.GetArray("Twitch.Users", []string{}) {
		twitcherName = strings.ToLower(twitcherName)
		p.twitchList[twitcherName] = &Twitcher{
			Name: twitcherName,
		}
	}

	p.register()
	p.registerWeb()

	return p
}

func (p *Twitch) registerWeb() {
	r := chi.NewRouter()
	r.HandleFunc("/online", p.onlineCB)
	r.HandleFunc("/offline", p.offlineCB)
	p.b.RegisterWeb(r, "/twitch")
}

func (p *Twitch) register() {
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
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^track (?P<twitchChannel>.+)$`),
			HelpText: "Bridge to this channel",
			Handler:  p.mkBridge,
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^untrack$`),
			HelpText: "Disconnect a bridge (only in bridged channel)",
			Handler:  p.rmBridge,
		},
		{
			Kind: bot.Message, IsCmd: false,
			Regex:   regexp.MustCompile(`.*`),
			Handler: p.bridgeMsg,
		},
		{
			Kind: bot.Startup, IsCmd: false,
			Regex:   regexp.MustCompile(`.*`),
			Handler: p.startup,
		},
		{
			Kind: bot.Shutdown, IsCmd: false,
			Regex:   regexp.MustCompile(`.*`),
			Handler: p.shutdown,
		},
	}
	p.b.Register(p, bot.Help, p.help)
	p.b.RegisterTable(p, p.tbl)
}

func (p *Twitch) shutdown(r bot.Request) bool {
	return false
}

func (p *Twitch) startup(r bot.Request) bool {
	var err error
	clientID := p.c.Get("twitch.clientid", "")
	clientSecret := p.c.Get("twitch.secret", "")

	if clientID == "" || clientSecret == "" {
		log.Info().Msg("No clientID/secret, twitch disabled")
		return false
	}

	p.helix, err = helix.NewClient(&helix.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		log.Printf("Login error: %v", err)
		return false
	}

	access, err := p.helix.RequestAppAccessToken([]string{"user:read:email"})
	if err != nil {
		log.Printf("Login error: %v", err)
		return false
	}

	fmt.Printf("%+v\n", access)

	// Set the access token on the client
	p.helix.SetAppAccessToken(access.Data.AccessToken)

	p.subs = map[string]bool{}

	for _, t := range p.twitchList {
		if err := p.follow(t); err != nil {
			log.Error().Err(err).Msg("")
		}
	}

	return false
}

func (p *Twitch) follow(twitcher *Twitcher) error {
	if twitcher.id == "" {
		users, err := p.helix.GetUsers(&helix.UsersParams{
			Logins: []string{twitcher.Name},
		})
		if err != nil {
			return err
		}

		if users.Error != "" {
			return errors.New(users.Error)
		}
		twitcher.id = users.Data.Users[0].ID
	}

	base := p.c.Get("baseurl", "") + "/twitch"

	_, err := p.helix.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeStreamOnline,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: twitcher.id,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: base + "/online",
			Secret:   "s3cre7w0rd",
		},
	})
	if err != nil {
		return err
	}

	_, err = p.helix.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    helix.EventSubTypeStreamOffline,
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: twitcher.id,
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: base + "/offline",
			Secret:   "s3cre7w0rd",
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Twitch) twitchStatus(r bot.Request) bool {
	channel := r.Msg.Channel
	if users := p.c.GetArray("Twitch."+channel+".Users", []string{}); len(users) > 0 {
		for _, twitcherName := range users {
			twitcherName = strings.ToLower(twitcherName)
			// we could re-add them here instead of needing to restart the bot.
			if t, ok := p.twitchList[twitcherName]; ok {
				if t.online {
					p.streaming(r.Conn, r.Msg.Channel, t)
				}
			}
		}
	}
	return true
}

func (p *Twitch) twitchUserStatus(r bot.Request) bool {
	who := strings.ToLower(r.Values["who"])
	if t, ok := p.twitchList[who]; ok {
		if t.online {
			p.streaming(r.Conn, r.Msg.Channel, t)
		}
	} else {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "I don't know who that is.")
	}
	return true
}

func (p *Twitch) resetTwitch(r bot.Request) bool {
	p.c.Set("twitch.istpl", isStreamingTplFallback)
	p.c.Set("twitch.nottpl", notStreamingTplFallback)
	p.c.Set("twitch.stoppedtpl", stoppedStreamingTplFallback)
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "The Twitch templates have been reset.")
	return true
}

func (p *Twitch) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...any) bool {
	msg := "You can set the templates for streams with\n"
	msg += fmt.Sprintf("twitch.istpl (default: %s)\n", isStreamingTplFallback)
	msg += fmt.Sprintf("twitch.nottpl (default: %s)\n", notStreamingTplFallback)
	msg += fmt.Sprintf("twitch.stoppedtpl (default: %s)\n", stoppedStreamingTplFallback)
	msg += "You can reset all messages with `!reset twitch`"
	msg += "And you can ask who is streaming with `!twitch status`"
	p.b.Send(c, bot.Message, message.Channel, msg)
	return true
}

func (p *Twitch) connectBridge(c bot.Connector, ch string, info *Twitcher) {
	msg := fmt.Sprintf("This post is tracking #%s\n<%s>", info.Name, info.url())
	err := p.startBridgeMsg(
		fmt.Sprintf("%s-%s-%s", info.Name, info.Game, time.Now().Format("2006-01-02-15:04")),
		info.Name,
		msg,
	)
	if err != nil {
		log.Error().Err(err).Msgf("unable to start bridge")
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("Unable to start bridge: %s", err))
	}
}

func (p *Twitch) disconnectBridge(c bot.Connector, twitcher *Twitcher) {
	log.Debug().Msgf("Disconnecting bridge: %s -> %+v", twitcher.Name, p.bridgeMap)
	for threadID, ircCh := range p.bridgeMap {
		if strings.HasSuffix(ircCh, twitcher.Name) {
			delete(p.bridgeMap, threadID)
			p.b.Send(c, bot.Message, threadID, "Stopped tracking #"+twitcher.Name)
		}
	}
}

func (p *Twitch) stopped(c bot.Connector, ch string, info *Twitcher) {
	notStreamingTpl := p.c.Get("Twitch.StoppedTpl", stoppedStreamingTplFallback)
	buf := bytes.Buffer{}
	t, err := template.New("notStreaming").Parse(notStreamingTpl)
	if err != nil {
		log.Error().Err(err)
		p.b.Send(c, bot.Message, ch, err)
		t = template.Must(template.New("notStreaming").Parse(stoppedStreamingTplFallback))
	}
	t.Execute(&buf, info)
	p.b.Send(c, bot.Message, ch, buf.String())
}

func (p *Twitch) streaming(c bot.Connector, channel string, info *Twitcher) {
	isStreamingTpl := p.c.Get("Twitch.IsTpl", isStreamingTplFallback)
	buf := bytes.Buffer{}
	t, err := template.New("isStreaming").Parse(isStreamingTpl)
	if err != nil {
		log.Error().Err(err)
		p.b.Send(c, bot.Message, channel, err)
		t = template.Must(template.New("isStreaming").Parse(isStreamingTplFallback))
	}
	t.Execute(&buf, info)
	p.b.Send(c, bot.Message, channel, buf.String())
}

func (p *Twitch) notStreaming(c bot.Connector, ch string, info *Twitcher) {
	notStreamingTpl := p.c.Get("Twitch.NotTpl", notStreamingTplFallback)
	buf := bytes.Buffer{}
	t, err := template.New("notStreaming").Parse(notStreamingTpl)
	if err != nil {
		log.Error().Err(err)
		p.b.Send(c, bot.Message, ch, err)
		t = template.Must(template.New("notStreaming").Parse(notStreamingTplFallback))
	}
	t.Execute(&buf, info)
	p.b.Send(c, bot.Message, ch, buf.String())
}

type twitchCB struct {
	Challenge    string `json:"challenge"`
	Subscription struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Version   string `json:"version"`
		Status    string `json:"status"`
		Cost      int    `json:"cost"`
		Condition struct {
			BroadcasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		CreatedAt time.Time `json:"created_at"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
		} `json:"transport"`
	} `json:"subscription"`
	Event struct {
		BroadcasterUserID    string `json:"broadcaster_user_id"`
		BroadcasterUserLogin string `json:"broadcaster_user_login"`
		BroadcasterUserName  string `json:"broadcaster_user_name"`
	} `json:"event"`
}

func (p *Twitch) offlineCB(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	defer r.Body.Close()
	// verify that the notification came from twitch using the secret.
	if !helix.VerifyEventSubNotification("s3cre7w0rd", r.Header, string(body)) {
		log.Error().Msg("no valid signature on subscription")
		return
	} else {
		log.Info().Msg("verified signature for subscription")
	}
	var vals twitchCB
	if err = json.Unmarshal(body, &vals); err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if vals.Challenge != "" {
		w.Write([]byte(vals.Challenge))
		return
	}

	log.Printf("got offline webhook: %v\n", vals)
	w.WriteHeader(200)
	w.Write([]byte("ok"))

	twitcher := p.twitchList[vals.Event.BroadcasterUserLogin]
	if !twitcher.online {
		return
	}
	twitcher.online = false
	twitcher.URL = twitcher.url()
	if ch := p.c.Get("twitch.channel", ""); ch != "" {
		p.stopped(p.b.DefaultConnector(), ch, twitcher)
		if p.c.GetBool("twitch.irc", false) {
			p.disconnectBridge(p.b.DefaultConnector(), twitcher)
		}
	}
}

func (p *Twitch) onlineCB(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}
	defer r.Body.Close()
	// verify that the notification came from twitch using the secret.
	if !helix.VerifyEventSubNotification("s3cre7w0rd", r.Header, string(body)) {
		log.Error().Msg("no valid signature on subscription")
		return
	} else {
		log.Info().Msg("verified signature for subscription")
	}
	var vals twitchCB
	if err = json.Unmarshal(body, &vals); err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	if vals.Challenge != "" {
		w.Write([]byte(vals.Challenge))
		return
	}

	log.Printf("got online webhook: %v\n", vals)
	w.WriteHeader(200)
	w.Write([]byte("ok"))

	streams, err := p.helix.GetStreams(&helix.StreamsParams{
		UserIDs: []string{vals.Event.BroadcasterUserID},
	})
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	twitcher := p.twitchList[vals.Event.BroadcasterUserLogin]

	if twitcher.online {
		return
	}

	twitcher.online = true
	twitcher.URL = twitcher.url()
	if len(streams.Data.Streams) > 0 {
		twitcher.gameID = streams.Data.Streams[0].GameID
		twitcher.Game = streams.Data.Streams[0].GameName
	} else {
		twitcher.gameID = "-1"
		twitcher.Game = p.c.Get("twitch.unknowngame", "IDK, Twitch didn't tell me")
	}
	if ch := p.c.Get("twitch.channel", ""); ch != "" {
		p.streaming(p.b.DefaultConnector(), ch, twitcher)
		if p.c.GetBool("twitch.irc", false) {
			p.connectBridge(p.b.DefaultConnector(), ch, twitcher)
		}
	}
}
