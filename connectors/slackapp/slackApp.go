package slackapp

import (
	"bytes"
	"container/ring"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
		log.Fatalf("No slack token found. Set SLACKTOKEN env.")
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
			log.Println(e)
			w.WriteHeader(http.StatusInternalServerError)
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				log.Println(err)
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
			log.Printf("Event: (%v): %+v", eventsAPIEvent.Type, eventsAPIEvent)
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
	if s.checkRingOrAdd(msg.TimeStamp) {
		log.Printf("Got a duplicate message from server: %s", msg.TimeStamp)
		return
	}
	isItMe := msg.BotID != "" && msg.BotID == s.myBotID
	if !isItMe && msg.ThreadTimeStamp == "" {
		m := s.buildMessage(msg)
		if m.Time.Before(s.lastRecieved) {
			log.Printf("Ignoring message: lastRecieved: %v msg: %v", s.lastRecieved, m.Time)
		} else {
			s.lastRecieved = m.Time
			s.event(bot.Message, m)
		}
	} else if msg.ThreadTimeStamp != "" {
		//we're throwing away some information here by not parsing the correct reply object type, but that's okay
		s.event(bot.Reply, s.buildMessage(msg), msg.ThreadTimeStamp)
	} else {
		log.Printf("THAT MESSAGE WAS HIDDEN: %+v", msg.Text)
	}
}

func (s *SlackApp) Send(kind bot.Kind, args ...interface{}) (string, error) {
	switch kind {
	case bot.Message:
		return s.sendMessage(args[0].(string), args[1].(string))
	case bot.Action:
		return s.sendAction(args[0].(string), args[1].(string))
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

func (s *SlackApp) sendMessageType(channel, message string, meMessage bool) (string, error) {
	ts, err := "", fmt.Errorf("")
	nick := s.config.Get("Nick", "bot")

	if meMessage {
		_, ts, err = s.api.PostMessage(channel,
			slack.MsgOptionUsername(nick),
			slack.MsgOptionText(message, false),
			slack.MsgOptionMeMessage())
	} else {
		_, ts, err = s.api.PostMessage(channel,
			slack.MsgOptionUsername(nick),
			slack.MsgOptionText(message, false))
	}

	if err != nil {
		log.Printf("Error sending message: %+v", err)
		return "", err
	}

	return ts, nil
}

func (s *SlackApp) sendMessage(channel, message string) (string, error) {
	log.Printf("Sending message to %s: %s", channel, message)
	identifier, err := s.sendMessageType(channel, message, false)
	return identifier, err
}

func (s *SlackApp) sendAction(channel, message string) (string, error) {
	log.Printf("Sending action to %s: %s", channel, message)
	identifier, err := s.sendMessageType(channel, "_"+message+"_", true)
	return identifier, err
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

	log.Println(string(body))

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
	log.Printf("Reacting in %s: %s", channel, reaction)
	ref := slack.ItemRef{
		Channel:   channel,
		Timestamp: message.AdditionalData["RAW_SLACK_TIMESTAMP"],
	}
	err := s.api.AddReaction(reaction, ref)
	return "", err
}

func (s *SlackApp) edit(channel, newMessage, identifier string) (string, error) {
	log.Printf("Editing in (%s) %s: %s", identifier, channel, newMessage)
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
		log.Println("Cannot get emoji list without slack.usertoken")
		return
	}
	api := slack.New(s.userToken, slack.OptionDebug(false))

	em, err := api.GetEmoji()
	if err != nil {
		log.Printf("Error retrieving emoji list from Slack: %s", err)
		return
	}

	s.emoji = em
}

// I think it's horseshit that I have to do this
func slackTStoTime(t string) time.Time {
	ts := strings.Split(t, ".")
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

	log.Printf("User %s not already found, requesting info", id)
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
		log.Println("Cannot get emoji list without slack.usertoken")
		return []string{s.config.Get("nick", "bot")}
	}
	api := slack.New(s.userToken, slack.OptionDebug(false))

	log.Println("Who is queried for ", id)
	// Not super sure this is the correct call
	params := &slack.GetUsersInConversationParameters{
		ChannelID: id,
		Limit:     50,
	}
	members, _, err := api.GetUsersInConversation(params)
	if err != nil {
		log.Println(err)
		return []string{s.config.Get("nick", "bot")}
	}

	ret := []string{}
	for _, m := range members {
		u, err := s.getUser(m)
		if err != nil {
			log.Printf("Couldn't get user %s: %s", m, err)
			continue
		}
		ret = append(ret, u)
	}
	return ret
}
