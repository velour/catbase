// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Package slack connects to slack service
package slack

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
	"golang.org/x/net/websocket"
)

type Slack struct {
	config *config.Config

	url string
	id  string
	ws  *websocket.Conn

	users map[string]string

	emoji map[string]string

	eventReceived   func(msg.Message)
	messageReceived func(msg.Message)
}

var idCounter uint64

type slackUserInfoResp struct {
	Ok   bool `json:"ok"`
	User struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
}

type slackChannelInfoResp struct {
	Ok      bool `json:"ok"`
	Channel struct {
		Id   string `json:"id"`
		Name string `json:"name"`

		Created int64  `json:"created"`
		Creator string `json:"creator"`

		Members []string `json:"members"`

		Topic struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int64  `json:"last_set"`
		} `json:"topic"`
	} `json:"channel"`
}

type slackMessage struct {
	Id       uint64 `json:"id"`
	Type     string `json:"type"`
	SubType  string `json:"subtype"`
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	User     string `json:"user"`
	Username string `json:"username"`
	Ts       string `json:"ts"`
	Error    struct {
		Code uint64 `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error"`
}

type slackReaction struct {
	Reaction string `json:"name"`
	Channel  string `json:"channel"`
	Timestamp  float64 `json:"timestamp"`
}

type rtmStart struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
	Url   string `json:"url"`
	Self  struct {
		Id string `json:"id"`
	} `json:"self"`
}

func New(c *config.Config) *Slack {
	return &Slack{
		config: c,
		users: make(map[string]string),
		emoji: make(map[string]string),
	}
}

func (s *Slack) RegisterEventReceived(f func(msg.Message)) {
	s.eventReceived = f
}

func (s *Slack) RegisterMessageReceived(f func(msg.Message)) {
	s.messageReceived = f
}

func (s *Slack) SendMessageType(channel, messageType, subType, message string) error {
	m := slackMessage{
		Id:      atomic.AddUint64(&idCounter, 1),
		Type:    messageType,
		SubType: subType,
		Channel: channel,
		Text:    message,
	}
	err := websocket.JSON.Send(s.ws, m)
	if err != nil {
		log.Printf("Error sending Slack message: %s", err)
	}
	return err
}

func (s *Slack) SendMessage(channel, message string) {
	log.Printf("Sending message to %s: %s", channel, message)
	s.SendMessageType(channel, "message", "", message)
}

func (s *Slack) SendAction(channel, message string) {
	log.Printf("Sending action to %s: %s", channel, message)
	s.SendMessageType(channel, "message", "me_message", "_"+message+"_")
}

func (s *Slack) React(channel, reaction string, message msg.Message) {
	log.Printf("Reacting in %s: %s", channel, reaction)
	resp, err := http.PostForm("https://slack.com/api/reactions.add",
		url.Values{ "token": {s.config.Slack.Token},
								"name": {reaction},
								"channel": {channel},
								"timestamp": {message.AdditionalData["RAW_SLACK_TIMESTAMP"]}})
	if err != nil {
		log.Printf("Error sending Slack reaction: %s", err)
	}
	log.Print(resp)
}

func (s *Slack) GetEmojiList() map[string]string {
	return s.emoji
}

func (s *Slack) populateEmojiList() {
	resp, err := http.PostForm("https://slack.com/api/emoji.list",
		url.Values{ "token": {s.config.Slack.Token}})
	if err != nil {
		log.Printf("Error retrieving emoji list from Slack: %s", err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("Error reading Slack API body: %s", err)
	}

	type EmojiListResponse struct {
		OK bool `json:ok`
		Emoji map[string]string `json:emoji`
	}

	var list EmojiListResponse
	err = json.Unmarshal(body, &list)
	if err != nil {
		log.Fatalf("Error parsing emoji list: %s", err)
	}
	s.emoji = list.Emoji
}

func (s *Slack) receiveMessage() (slackMessage, error) {
	var msg []byte
	m := slackMessage{}
	//err := websocket.JSON.Receive(s.ws, &m)
	err := websocket.Message.Receive(s.ws, &msg)
	if err != nil {
		log.Println("Error decoding WS message")
		return m, err
	}
	log.Printf("Raw response from Slack: %s", msg)
	err2 := json.Unmarshal(msg, &m)
	return m, err2
}

func (s *Slack) Serve() {
	s.connect()
	s.populateEmojiList()
	for {
		msg, err := s.receiveMessage()
		if err != nil && err == io.EOF {
			log.Fatalf("Slack API EOF")
		} else if err != nil {
			log.Printf("Slack API error: %s", err)
			continue
		}
		switch msg.Type {
		case "message":
			s.messageReceived(s.buildMessage(msg))
		case "error":
			log.Printf("Slack error, code: %d, message: %s", msg.Error.Code, msg.Error.Msg)
		case "": // what even is this?
		case "hello":
		case "presence_change":
		case "user_typing":
		case "reconnect_url":
			// squeltch this stuff
		default:
			log.Printf("Unhandled Slack message type: '%s'", msg.Type)
		}
	}
}

var urlDetector = regexp.MustCompile(`<(.+)://([^|^>]+).*>`)

// Convert a slackMessage to a msg.Message
func (s *Slack) buildMessage(m slackMessage) msg.Message {
	log.Printf("DEBUG: msg: %#v", m)
	text := html.UnescapeString(m.Text)

	// remove <> from URLs, URLs may also be <url|description>
	text = urlDetector.ReplaceAllString(text, "${1}://${2}")

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := strings.HasPrefix(text, "/me ")
	if isAction {
		text = text[3:]
	}

	u := s.getUser(m.User)
	if m.Username != "" {
		u = m.Username
	}

	// I think it's horseshit that I have to do this
	ts := strings.Split(m.Ts, ".")
	sec, _ := strconv.ParseInt(ts[0], 10, 64)
	nsec, _ := strconv.ParseInt(ts[1], 10, 64)
	tstamp := time.Unix(sec, nsec)

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
		Host:    string(m.Id),
		Time:    tstamp,
		AdditionalData: map[string]string{
			"RAW_SLACK_TIMESTAMP": m.Ts,
		},
	}
}

func (s *Slack) connect() {
	token := s.config.Slack.Token
	url := fmt.Sprintf("https://slack.com/api/rtm.start?token=%s", token)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		log.Fatalf("Slack API failed. Code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("Error reading Slack API body: %s", err)
	}
	var rtm rtmStart
	err = json.Unmarshal(body, &rtm)
	if err != nil {
		return
	}

	if !rtm.Ok {
		log.Fatalf("Slack error: %s", rtm.Error)
	}

	s.url = "https://slack.com/api/"
	s.id = rtm.Self.Id

	s.ws, err = websocket.Dial(rtm.Url, "", s.url)
	if err != nil {
		log.Fatal(err)
	}
}

// Get username for Slack user ID
func (s *Slack) getUser(id string) string {
	if name, ok := s.users[id]; ok {
		return name
	}

	log.Printf("User %s not already found, requesting info", id)
	u := s.url + "users.info"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.config.Slack.Token}, "user": {id}})
	if err != nil || resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d %s",
			resp.StatusCode, err)
		return "UNKNOWN"
	}
	defer resp.Body.Close()
	var userInfo slackUserInfoResp
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		log.Println("Error decoding response: ", err)
		return "UNKNOWN"
	}
	s.users[id] = userInfo.User.Name
	return s.users[id]
}

// Who gets usernames out of a channel
func (s *Slack) Who(id string) []string {
	log.Println("Who is queried for ", id)
	u := s.url + "channels.info"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.config.Slack.Token}, "channel": {id}})
	if err != nil {
		log.Printf("Error posting user info request: %s",
			err)
		return []string{}
	}
	if resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d",
			resp.StatusCode)
		return []string{}
	}
	defer resp.Body.Close()
	var chanInfo slackChannelInfoResp
	err = json.NewDecoder(resp.Body).Decode(&chanInfo)
	if err != nil || !chanInfo.Ok {
		log.Println("Error decoding response: ", err)
		return []string{}
	}

	log.Printf("%#v", chanInfo.Channel)

	handles := []string{}
	for _, member := range chanInfo.Channel.Members {
		handles = append(handles, s.getUser(member))
	}
	log.Printf("Returning %d handles", len(handles))
	return handles
}
