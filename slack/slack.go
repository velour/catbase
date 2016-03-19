// Package slack connects to slack service
package slack

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"golang.org/x/net/websocket"
)

type Slack struct {
	config *config.Config

	url string
	id  string
	ws  *websocket.Conn

	users map[string]string

	eventReceived   func(bot.Message)
	messageReceived func(bot.Message)
}

var idCounter uint64

type slackUserInfoResp struct {
	Ok   bool `json:"ok"`
	User struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"user"`
}

type slackMessage struct {
	Id      uint64 `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subtype"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
	User    string `json:"user"`
	Error   struct {
		Code uint64 `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error"`
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
		users:  make(map[string]string),
	}
}

func (s *Slack) RegisterEventReceived(f func(bot.Message)) {
	s.eventReceived = f
}

func (s *Slack) RegisterMessageReceived(f func(bot.Message)) {
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

func (s *Slack) receiveMessage() (slackMessage, error) {
	m := slackMessage{}
	err := websocket.JSON.Receive(s.ws, &m)
	return m, err
}

func (s *Slack) Serve() {
	s.connect()
	for {
		msg, err := s.receiveMessage()
		if err != nil {
			log.Fatalf("Slack API error: %s", err)
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

// Convert a slackMessage to a bot.Message
func (s *Slack) buildMessage(msg slackMessage) bot.Message {
	log.Printf("DEBUG: msg: %#v", msg)
	text := html.UnescapeString(msg.Text)

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := strings.HasPrefix(text, "/me ")
	if isAction {
		text = text[3:]
	}

	user := s.getUser(msg.User)

	return bot.Message{
		User: &bot.User{
			Name: user,
		},
		Body:    text,
		Raw:     msg.Text,
		Channel: msg.Channel,
		Command: isCmd,
		Action:  isAction,
		Host:    string(msg.Id),
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
	var userInfo slackUserInfoResp
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		log.Println("Error decoding response: ", err)
		return "UNKNOWN"
	}
	s.users[id] = userInfo.User.Name
	return s.users[id]
}
