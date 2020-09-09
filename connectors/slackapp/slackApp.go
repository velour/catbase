package slackapp

import (
	"bytes"
	"container/ring"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

const DefaultIDRing = 10
const DefaultMsgRing = 50
const defaultLogFormat = "[{{fixDate .Time \"2006-01-02 15:04:05\"}}] {{if .TopicChange}}*** {{.User.Name}}{{else if .Action}}* {{.User.Name}}{{else}}<{{.User.Name}}>{{end}} {{.Body}}\n"

// 11:10AM DBG connectors/slackapp/slackApp.go:496 > Slack event dir=logs raw={"Action":false,"AdditionalData":
// {"RAW_SLACK_TIMESTAMP":"1559920235.001100"},"Body":"aoeu","Channel":"C0S04SMRC","ChannelName":"test",
// "Command":false,"Host":"","IsIM":false,"Raw":{"channel":"C0S04SMRC","channel_type":"channel",
// "event_ts":1559920235.001100,"files":null,"text":"aoeu","thread_ts":"","ts":"1559920235.001100",
// "type":"message","upload":false,"user":"U0RLUDELD"},"Time":"2019-06-07T11:10:35.0000011-04:00",
// "User":{"Admin":false,"ID":"U0RLUDELD","Name":"flyngpngn"}}

type SlackApp struct {
	config *config.Config
	api    *slack.Client

	botToken     string
	userToken    string
	verification string
	id           string

	lastRecieved time.Time

	myBotID  string
	users    map[string]*slack.User
	emoji    map[string]string
	channels map[string]*slack.Channel

	event bot.Callback

	msgIDBuffer *ring.Ring
	msgBuffer   *ring.Ring

	logFormat *template.Template
}

func fixDate(input time.Time, format string) string {
	return input.Format(format)
}
func New(c *config.Config) *SlackApp {
	token := c.Get("slack.token", "NONE")
	if token == "NONE" {
		log.Fatal().Msg("No slack token found. Set SLACKTOKEN env.")
	}

	api := slack.New(token, slack.OptionDebug(false))

	tplTxt := c.GetString("slackapp.logMessage.format", defaultLogFormat)
	funcs := template.FuncMap{
		"fixDate": fixDate,
	}
	tpl := template.Must(template.New("logMessage").Funcs(funcs).Parse(tplTxt))

	return &SlackApp{
		api:          api,
		config:       c,
		botToken:     token,
		userToken:    c.Get("slack.usertoken", "NONE"),
		verification: c.Get("slack.verification", "NONE"),
		myBotID:      c.Get("slack.botid", ""),
		lastRecieved: time.Now(),
		users:        make(map[string]*slack.User),
		emoji:        make(map[string]string),
		channels:     make(map[string]*slack.Channel),
		msgIDBuffer:  ring.New(c.GetInt("idRingSize", DefaultIDRing)),
		msgBuffer:    ring.New(c.GetInt("msgRingSize", DefaultMsgRing)),
		logFormat:    tpl,
	}
}

func (s *SlackApp) RegisterEvent(f bot.Callback) {
	s.event = f
}

func (s *SlackApp) Serve() error {
	s.populateEmojiList()

	http.HandleFunc("/evt", func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.String()
		eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: s.verification}))
		if e != nil {
			log.Error().Err(e)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				log.Error().Err(err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		} else if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			typ := innerEvent.Type
			switch ev := innerEvent.Data.(type) {
			case *slackevents.MessageAction:
				log.Debug().Interface("ev", ev).Msg("MessageAction")
			case *slackevents.MessageActionResponse:
				log.Debug().Interface("ev", ev).Msg("MessageActionResponse")
			case *slackevents.AppMentionEvent:
				// This is a bit of a problem. AppMentionEvent also needs to
				// End up in msgReceived
				//s.msgReceivd(ev)
			case *slackevents.MessageEvent:
				s.msgReceivd(ev)
			case *slackevents.ReactionAddedEvent:
				err := s.reactionReceived(ev)
				if err != nil {
					log.Error().Err(err).Msg("error with reaction recording")
				}
			default:
				log.Debug().
					Interface("ev", ev).
					Interface("type", typ).
					Msg("Unknown CallbackEvent")
			}
		} else {
			log.Debug().
				Str("type", eventsAPIEvent.Type).
				Interface("event", eventsAPIEvent).
				Msg("event")
		}
	})
	return nil
}

// checkRingOrAdd returns true if it finds the ts value
// or false if the ts isn't yet in the ring (and adds it)
func (s *SlackApp) checkRingOrAdd(ts string) bool {
	found := false
	s.msgIDBuffer.Do(func(p interface{}) {
		if p == nil {
			return
		}
		if p.(string) == ts {
			found = true
		}
	})
	if found {
		return true
	}
	s.msgIDBuffer.Value = ts
	s.msgIDBuffer = s.msgIDBuffer.Next()
	return false
}

func (s *SlackApp) msgReceivd(msg *slackevents.MessageEvent) {
	if msg.TimeStamp == "" {
		log.Debug().
			Str("type", msg.SubType).
			Msg("ignoring an unhandled event type")
		return
	}

	log.Debug().
		Interface("event", msg).
		Str("type", msg.SubType).
		Msg("accepting a message")

	if s.checkRingOrAdd(msg.TimeStamp) {
		log.Debug().
			Str("ts", msg.TimeStamp).
			Msg("Got a duplicate message from server")
		return
	}

	isItMe := msg.BotID != "" && msg.BotID == s.myBotID
	m := s.buildMessage(msg)
	if m.Time.Before(s.lastRecieved) {
		log.Debug().
			Time("ts", m.Time).
			Interface("lastRecv", s.lastRecieved).
			Msg("Ignoring message")
		return
	}
	if err := s.logMessage(m); err != nil {
		log.Fatal().Err(err).Msg("Error logging message")
	}
	if !isItMe && msg.ThreadTimeStamp == "" {
		s.lastRecieved = m.Time
		s.event(s, bot.Message, m)
	} else if msg.ThreadTimeStamp != "" {
		//we're throwing away some information here by not parsing the correct reply object type, but that's okay
		s.event(s, bot.Reply, s.buildMessage(msg), msg.ThreadTimeStamp)
	} else if isItMe {
		s.event(s, bot.SelfMessage, m)
	} else {
		log.Debug().
			Str("text", msg.Text).
			Msg("Unknown message is hidden")
	}

	s.msgBuffer.Value = msg
	s.msgBuffer = s.msgBuffer.Next()
}

func (s *SlackApp) Send(kind bot.Kind, args ...interface{}) (string, error) {
	switch kind {
	case bot.Message:
		return s.sendMessage(args[0].(string), args[1].(string), false, args...)
	case bot.Action:
		return s.sendMessage(args[0].(string), args[1].(string), true, args...)
	case bot.Edit:
		return s.edit(args[0].(string), args[1].(string), args[2].(string))
	case bot.Reply:
		switch args[2].(type) {
		case msg.Message:
			return s.replyToMessage(args[0].(string), args[1].(string), args[2].(msg.Message))
		case string:
			return s.replyToMessageIdentifier(args[0].(string), args[1].(string), args[2].(string))
		default:
			return "", fmt.Errorf("Invalid types given to Reply")
		}
	case bot.Reaction:
		return s.react(args[0].(string), args[1].(string), args[2].(msg.Message))
	default:
	}
	return "", fmt.Errorf("No handler for message type %d", kind)
}

func (s *SlackApp) sendMessage(channel, message string, meMessage bool, args ...interface{}) (string, error) {
	ts, err := "", fmt.Errorf("")
	nick := s.config.Get("Nick", "bot")

	options := []slack.MsgOption{
		slack.MsgOptionUsername(nick),
		slack.MsgOptionText(message, false),
	}
	if meMessage {
		options = append(options, slack.MsgOptionMeMessage())
	}

	// Check for message attachments
	attachments := []slack.Attachment{}
	if len(args) > 0 {
		for _, a := range args {
			switch a := a.(type) {
			case bot.ImageAttachment:
				attachments = append(attachments, slack.Attachment{
					ImageURL: a.URL,
					Text:     a.AltTxt,
				})
			}
		}
	}

	if len(attachments) > 0 {
		options = append(options, slack.MsgOptionAttachments(attachments...))
	}

	log.Debug().
		Str("channel", channel).
		Str("message", message).
		Int("attachment count", len(attachments)).
		Int("option count", len(options)).
		Int("arg count", len(args)).
		Msg("Sending message")

	_, ts, err = s.api.PostMessage(channel, options...)

	if err != nil {
		log.Error().Err(err).Msg("Error sending message")
		return "", err
	}

	return ts, nil
}

func (s *SlackApp) replyToMessageIdentifier(channel, message, identifier string) (string, error) {
	nick := s.config.Get("Nick", "bot")

	resp, err := http.PostForm("https://slack.com/api/chat.postMessage",
		url.Values{"token": {s.botToken},
			"username":  {nick},
			"channel":   {channel},
			"text":      {message},
			"thread_ts": {identifier},
		})

	if err != nil {
		err := fmt.Errorf("Error sending Slack reply: %s", err)
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		err := fmt.Errorf("Error reading Slack API body: %s", err)
		return "", err
	}

	log.Debug().Bytes("body", body)

	type MessageResponse struct {
		OK        bool   `json:"ok"`
		Timestamp string `json:"ts"`
	}

	var mr MessageResponse
	err = json.Unmarshal(body, &mr)
	if err != nil {
		err := fmt.Errorf("Error parsing message response: %s", err)
		return "", err
	}

	if !mr.OK {
		return "", fmt.Errorf("Got !OK from slack message response")
	}

	return mr.Timestamp, err
}

func (s *SlackApp) replyToMessage(channel, message string, replyTo msg.Message) (string, error) {
	return s.replyToMessageIdentifier(channel, message, replyTo.AdditionalData["RAW_SLACK_TIMESTAMP"])
}

func (s *SlackApp) react(channel, reaction string, message msg.Message) (string, error) {
	log.Debug().
		Str("channel", channel).
		Str("reaction", reaction).
		Msg("reacting")
	ref := slack.ItemRef{
		Channel:   channel,
		Timestamp: message.AdditionalData["RAW_SLACK_TIMESTAMP"],
	}
	err := s.api.AddReaction(reaction, ref)
	return "", err
}

func (s *SlackApp) edit(channel, newMessage, identifier string) (string, error) {
	log.Debug().
		Str("channel", channel).
		Str("identifier", identifier).
		Str("newMessage", newMessage).
		Msg("editing")
	nick := s.config.Get("Nick", "bot")
	_, ts, err := s.api.PostMessage(channel,
		slack.MsgOptionUsername(nick),
		slack.MsgOptionText(newMessage, false),
		slack.MsgOptionMeMessage(),
		slack.MsgOptionUpdate(identifier))
	return ts, err
}

func (s *SlackApp) GetEmojiList() map[string]string {
	return s.emoji
}

func (s *SlackApp) populateEmojiList() {
	if s.userToken == "NONE" {
		log.Error().Msg("Cannot get emoji list without slack.usertoken")
		return
	}
	api := slack.New(s.userToken, slack.OptionDebug(false))

	em, err := api.GetEmoji()
	if err != nil {
		log.Error().Err(err).Msg("Error retrieving emoji list from Slack")
		return
	}

	s.emoji = em
}

// I think it's horseshit that I have to do this
func slackTStoTime(t string) time.Time {
	ts := strings.Split(t, ".")
	if len(ts) < 2 {
		log.Fatal().
			Str("ts", t).
			Msg("Could not parse Slack timestamp")
	}
	sec, _ := strconv.ParseInt(ts[0], 10, 64)
	nsec, _ := strconv.ParseInt(ts[1], 10, 64)
	return time.Unix(sec, nsec)
}

var urlDetector = regexp.MustCompile(`<(.+)://([^|^>]+).*>`)

// Convert a slackMessage to a msg.Message
func (s *SlackApp) buildMessage(m *slackevents.MessageEvent) msg.Message {
	text := html.UnescapeString(m.Text)

	text = fixText(s.getUser, text)

	// Slack likes to put these NBSP in and it screws with matching later
	text = strings.ReplaceAll(text, "\u00a0", " ")

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := m.SubType == "me_message"

	// We have to try a few layers to get a valid name for the user because Slack
	defaultName := "unknown"
	if m.BotID != "" {
		defaultName = m.Username
	}
	icon := ""
	log.Debug().Msgf("Getting user %s", m.User)
	name, u := s.getUser(m.User, defaultName)
	if u != nil {
		icon = u.Profile.Image192
	}
	if m.Username != "" && name == "unknown" {
		name = m.Username
	}

	chName := m.Channel
	if ch, _ := s.getChannel(m.Channel); ch != nil {
		chName = ch.Name
	}

	tstamp := slackTStoTime(m.TimeStamp)

	return msg.Message{
		ID: m.TimeStamp,
		User: &user.User{
			ID:   m.User,
			Name: name,
			Icon: icon,
		},
		Body:        text,
		Raw:         m,
		Channel:     m.Channel,
		ChannelName: chName,
		IsIM:        m.ChannelType == "im",
		Command:     isCmd,
		Action:      isAction,
		Time:        tstamp,
		AdditionalData: map[string]string{
			"RAW_SLACK_TIMESTAMP": m.TimeStamp,
		},
	}
}

func (s *SlackApp) getChannel(id string) (*slack.Channel, error) {
	if ch, ok := s.channels[id]; ok {
		return ch, nil
	}

	log.Debug().
		Str("id", id).
		Msg("Channel not known, requesting info")

	ch, err := s.api.GetChannelInfo(id)
	if err != nil {
		return nil, err
	}
	s.channels[id] = ch
	return s.channels[id], nil
}

func stringForUser(user *slack.User) string {
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	return user.Name
}

// Get username for Slack user ID
func (s *SlackApp) getUser(id, defaultName string) (string, *slack.User) {
	if u, ok := s.users[id]; ok {
		return stringForUser(u), u
	}

	log.Debug().
		Str("id", id).
		Msg("User not already found, requesting info")

	u, err := s.api.GetUserInfo(id)
	if err != nil {
		return defaultName, nil
	}
	s.users[id] = u
	return stringForUser(u), u
}

// Who gets usernames out of a channel
func (s *SlackApp) Who(id string) []string {
	if s.userToken == "NONE" {
		log.Error().Msg("Cannot get emoji list without slack.usertoken")
		return []string{s.config.Get("nick", "bot")}
	}
	api := slack.New(s.userToken, slack.OptionDebug(false))

	log.Debug().
		Str("id", id).
		Msg("Who is queried")
	// Not super sure this is the correct call
	params := &slack.GetUsersInConversationParameters{
		ChannelID: id,
		Limit:     50,
	}
	members, _, err := api.GetUsersInConversation(params)
	if err != nil {
		log.Error().Err(err)
		return []string{s.config.Get("nick", "bot")}
	}

	ret := []string{}
	for _, m := range members {
		if m == "" {
			log.Error().Msg("empty member")
		}
		u, _ := s.getUser(m, "unknown")
		if u == "unknown" {
			log.Error().
				Err(err).
				Str("user", m).
				Msg("Couldn't get user")
			continue
			/**/
		}
		ret = append(ret, u)
	}
	return ret
}

// logMessage writes to a <slackapp.logMessage.dir>/<channel>.logMessage
// Uses slackapp.logMessage.format to write entries
func (s *SlackApp) logMessage(raw msg.Message) error {
	// Do some filtering and fixing up front
	if raw.Body == "" {
		return nil
	}

	data := struct {
		msg.Message
		TopicChange bool
	}{
		Message: raw,
	}

	if strings.Contains(raw.Body, "set the channel topic: ") {
		topic := strings.SplitN(raw.Body, "set the channel topic: ", 2)
		data.Body = "changed topic to " + topic[1]
		data.TopicChange = true
	}

	buf := bytes.Buffer{}
	if err := s.logFormat.Execute(&buf, data); err != nil {
		return err
	}

	return s.log(buf.String(), raw.ChannelName)
}

func (s *SlackApp) log(msg, channel string) error {
	dir := path.Join(s.config.Get("slackapp.logMessage.dir", "logs"), channel)
	now := time.Now()
	fname := now.Format("20060102") + ".log"
	path := path.Join(dir, fname)

	log.Debug().
		Interface("raw", msg).
		Str("dir", dir).
		Msg("Slack event")

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error().
			Err(err).
			Msg("Could not create logMessage directory")
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Error opening logMessage file")
	}
	defer f.Close()

	if _, err := fmt.Fprint(f, msg); err != nil {
		return err
	}

	return f.Sync()
}

func (s *SlackApp) reactionReceived(event *slackevents.ReactionAddedEvent) error {
	log.Debug().Msgf("reactionReceived(%+v)", event)
	name, _ := s.getUser(event.User, "unknown")

	ch, err := s.getChannel(event.Item.Channel)
	if err != nil {
		return err
	}
	channel := ch.Name

	body := "unknown message"

	// Iterate through the ring and print its contents
	s.msgBuffer.Do(func(p interface{}) {
		switch m := p.(type) {
		case nil:
		case *slackevents.MessageEvent:
			if m.TimeStamp == event.Item.Timestamp {
				defaultName := "unknown"
				if m.BotID != "" {
					defaultName = m.Username
				}
				u, _ := s.getUser(m.User, defaultName)
				body = fmt.Sprintf("%s: %s", u, m.Text)
			}
		default:
			log.Debug().Interface("msg", m).Msg("Unexpected type in reaction received")
		}
	})

	tstamp := slackTStoTime(event.EventTimestamp)
	msg := fmt.Sprintf("[%s] <%s> reacted to %s with :%s:\n",
		fixDate(tstamp, "2006-01-02 15:04:05"),
		name, body, event.Reaction)

	log.Debug().Msgf("Made it to reaction received, logging %v: %v", msg, channel)

	return s.log(msg, channel)
}

func (s *SlackApp) Profile(identifier string) (user.User, error) {
	log.Debug().Msgf("Getting profile for %s", identifier)

	users, err := s.api.GetUsers()
	if err != nil {
		return user.User{}, err
	}

	for _, u := range users {
		if u.Name == identifier || u.ID == identifier {
			return user.User{
				ID:    u.ID,
				Name:  stringForUser(&u),
				Admin: false,
				Icon:  u.Profile.Image192,
			}, nil
		}
	}

	return user.User{}, fmt.Errorf("user %s not found", err)
}
