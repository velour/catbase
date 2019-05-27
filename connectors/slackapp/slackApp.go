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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
)

const DEFAULT_RING = 5

type SlackApp struct {
	bot    bot.Bot
	config *config.Config
	api    *slack.Client

	botToken     string
	userToken    string
	verification string
	id           string

	lastRecieved time.Time

	myBotID string
	users   map[string]string
	emoji   map[string]string

	event bot.Callback

	msgIDBuffer *ring.Ring
}

func New(c *config.Config) *SlackApp {
	token := c.Get("slack.token", "NONE")
	if token == "NONE" {
		log.Fatal().Msg("No slack token found. Set SLACKTOKEN env.")
	}

	api := slack.New(token, slack.OptionDebug(false))

	idBuf := ring.New(c.GetInt("ringSize", DEFAULT_RING))
	for i := 0; i < idBuf.Len(); i++ {
		idBuf.Value = ""
		idBuf = idBuf.Next()
	}

	return &SlackApp{
		api:          api,
		config:       c,
		botToken:     token,
		userToken:    c.Get("slack.usertoken", "NONE"),
		verification: c.Get("slack.verification", "NONE"),
		myBotID:      c.Get("slack.botid", ""),
		lastRecieved: time.Now(),
		users:        make(map[string]string),
		emoji:        make(map[string]string),
		msgIDBuffer:  idBuf,
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
			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				// This is a bit of a problem. AppMentionEvent also needs to
				// End up in msgReceived
				//s.msgReceivd(ev)
			case *slackevents.MessageEvent:
				s.msgReceivd(ev)
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
		Str("type", msg.SubType).
		Msg("accepting a message")

	if s.checkRingOrAdd(msg.TimeStamp) {
		log.Debug().
			Str("ts", msg.TimeStamp).
			Msg("Got a duplicate message from server")
		return
	}
	isItMe := msg.BotID != "" && msg.BotID == s.myBotID
	if !isItMe && msg.ThreadTimeStamp == "" {
		m := s.buildMessage(msg)
		if m.Time.Before(s.lastRecieved) {
			log.Debug().
				Time("ts", m.Time).
				Interface("lastRecv", s.lastRecieved).
				Msg("Ignoring message")
		} else {
			s.lastRecieved = m.Time
			s.event(s, bot.Message, m)
		}
	} else if msg.ThreadTimeStamp != "" {
		//we're throwing away some information here by not parsing the correct reply object type, but that's okay
		s.event(s, bot.Reply, s.buildMessage(msg), msg.ThreadTimeStamp)
	} else {
		log.Debug().
			Str("text", msg.Text).
			Msg("THAT MESSAGE WAS HIDDEN")
	}
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
	icon := s.config.Get("IconURL", "https://placekitten.com/128/128")

	resp, err := http.PostForm("https://slack.com/api/chat.postMessage",
		url.Values{"token": {s.botToken},
			"username":  {nick},
			"icon_url":  {icon},
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

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := m.SubType == "me_message"

	u, _ := s.getUser(m.User)
	if m.Username != "" {
		u = m.Username
	}

	tstamp := slackTStoTime(m.TimeStamp)

	return msg.Message{
		User: &user.User{
			ID:   m.User,
			Name: u,
		},
		Body:    text,
		Raw:     m.Text,
		Channel: m.Channel,
		Command: isCmd,
		Action:  isAction,
		Time:    tstamp,
		AdditionalData: map[string]string{
			"RAW_SLACK_TIMESTAMP": m.TimeStamp,
		},
	}
}

// Get username for Slack user ID
func (s *SlackApp) getUser(id string) (string, error) {
	if name, ok := s.users[id]; ok {
		return name, nil
	}

	log.Debug().
		Str("id", id).
		Msg("User not already found, requesting info")
	u, err := s.api.GetUserInfo(id)
	if err != nil {
		return "UNKNOWN", err
	}
	s.users[id] = u.Name
	return s.users[id], nil
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
		u, err := s.getUser(m)
		if err != nil {
			log.Error().
				Err(err).
				Str("user", m).
				Msg("Couldn't get user")
			continue
		}
		ret = append(ret, u)
	}
	return ret
}
