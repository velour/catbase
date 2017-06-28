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

type slackChannelListItem struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	IsChannel      bool     `json:"is_channel"`
	Created        int      `json:"created"`
	Creator        string   `json:"creator"`
	IsArchived     bool     `json:"is_archived"`
	IsGeneral      bool     `json:"is_general"`
	NameNormalized string   `json:"name_normalized"`
	IsShared       bool     `json:"is_shared"`
	IsOrgShared    bool     `json:"is_org_shared"`
	IsMember       bool     `json:"is_member"`
	Members        []string `json:"members"`
	Topic          struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int    `json:"last_set"`
	} `json:"topic"`
	Purpose struct {
		Value   string `json:"value"`
		Creator string `json:"creator"`
		LastSet int    `json:"last_set"`
	} `json:"purpose"`
	PreviousNames []interface{} `json:"previous_names"`
	NumMembers    int           `json:"num_members"`
}

type slackChannelListResp struct {
	Ok       bool                   `json:"ok"`
	Channels []slackChannelListItem `json:"channels"`
}

type slackChannelInfoResp struct {
	Ok      bool `json:"ok"`
	Channel struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		IsChannel      bool   `json:"is_channel"`
		Created        int    `json:"created"`
		Creator        string `json:"creator"`
		IsArchived     bool   `json:"is_archived"`
		IsGeneral      bool   `json:"is_general"`
		NameNormalized string `json:"name_normalized"`
		IsReadOnly     bool   `json:"is_read_only"`
		IsShared       bool   `json:"is_shared"`
		IsOrgShared    bool   `json:"is_org_shared"`
		IsMember       bool   `json:"is_member"`
		LastRead       string `json:"last_read"`
		Latest         struct {
			Type string `json:"type"`
			User string `json:"user"`
			Text string `json:"text"`
			Ts   string `json:"ts"`
		} `json:"latest"`
		UnreadCount        int      `json:"unread_count"`
		UnreadCountDisplay int      `json:"unread_count_display"`
		Members            []string `json:"members"`
		Topic              struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"topic"`
		Purpose struct {
			Value   string `json:"value"`
			Creator string `json:"creator"`
			LastSet int    `json:"last_set"`
		} `json:"purpose"`
		PreviousNames []string `json:"previous_names"`
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
	Reaction  string  `json:"name"`
	Channel   string  `json:"channel"`
	Timestamp float64 `json:"timestamp"`
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
		url.Values{"token": {s.config.Slack.Token},
			"name":      {reaction},
			"channel":   {channel},
			"timestamp": {message.AdditionalData["RAW_SLACK_TIMESTAMP"]}})
	if err != nil {
		log.Printf("Error sending Slack reaction: %s", err)
	}
	log.Print(resp)
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

// markAllChannelsRead gets a list of all channels and marks each as read
func (s *Slack) markAllChannelsRead() {
	chs := s.getAllChannels()
	log.Printf("Got list of channels to mark read: %+v", chs)
	for _, ch := range chs {
		s.markChannelAsRead(ch.ID)
	}
	log.Printf("Finished marking channels read")
}

// getAllChannels returns info for all channels joined
func (s *Slack) getAllChannels() []slackChannelListItem {
	u := s.url + "channels.list"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.config.Slack.Token}})
	if err != nil {
		log.Printf("Error posting user info request: %s",
			err)
		return nil
	}
	if resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d",
			resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()
	var chanInfo slackChannelListResp
	err = json.NewDecoder(resp.Body).Decode(&chanInfo)
	if err != nil || !chanInfo.Ok {
		log.Println("Error decoding response: ", err)
		return nil
	}
	return chanInfo.Channels
}

// markAsRead marks a channel read
func (s *Slack) markChannelAsRead(slackChanId string) error {
	u := s.url + "channels.info"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.config.Slack.Token}, "channel": {slackChanId}})
	if err != nil {
		log.Printf("Error posting user info request: %s",
			err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d",
			resp.StatusCode)
		return err
	}
	defer resp.Body.Close()
	var chanInfo slackChannelInfoResp
	err = json.NewDecoder(resp.Body).Decode(&chanInfo)
	log.Printf("%+v, %+v", err, chanInfo)
	if err != nil || !chanInfo.Ok {
		log.Println("Error decoding response: ", err)
		return err
	}

	u = s.url + "channels.mark"
	resp, err = http.PostForm(u,
		url.Values{"token": {s.config.Slack.Token}, "channel": {slackChanId}, "ts": {chanInfo.Channel.Latest.Ts}})
	if err != nil {
		log.Printf("Error posting user info request: %s",
			err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d",
			resp.StatusCode)
		return err
	}
	defer resp.Body.Close()
	var markInfo map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&markInfo)
	log.Printf("%+v, %+v", err, markInfo)
	if err != nil {
		log.Println("Error decoding response: ", err)
		return err
	}

	log.Printf("Marked %s as read", slackChanId)
	return nil
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

	s.markAllChannelsRead()

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
