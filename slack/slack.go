// © 2016 the CatBase Authors under the WTFPL license. See AUTHORS for the list of authors.

// Package slack connects to slack service
package slack

import (
	"encoding/json"
	"errors"
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

	// "sync/atomic"
	"context"
	"time"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/bot/user"
	"github.com/velour/catbase/config"
	"github.com/velour/chat/websocket"
)

type Slack struct {
	config *config.Config

	url   string
	id    string
	token string
	ws    *websocket.Conn

	lastRecieved time.Time

	users map[string]string

	myBotID string

	emoji map[string]string

	event func(bot.Kind, msg.Message, ...interface{})
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
			LastSet int64  `json:"last_set"`
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
	ID       uint64 `json:"id"`
	Type     string `json:"type"`
	SubType  string `json:"subtype"`
	Hidden   bool   `json:"hidden"`
	Channel  string `json:"channel"`
	Text     string `json:"text"`
	User     string `json:"user"`
	Username string `json:"username"`
	BotID    string `json:"bot_id"`
	Ts       string `json:"ts"`
	ThreadTs string `json:"thread_ts"`
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
	URL   string `json:"url"`
	Self  struct {
		ID string `json:"id"`
	} `json:"self"`
}

func New(c *config.Config) *Slack {
	token := c.Get("slack.token", "NONE")
	if token == "NONE" {
		log.Fatalf("No slack token found. Set SLACKTOKEN env.")
	}
	return &Slack{
		config:       c,
		token:        c.Get("slack.token", ""),
		lastRecieved: time.Now(),
		users:        make(map[string]string),
		emoji:        make(map[string]string),
	}
}

func (s *Slack) Send(kind bot.Kind, args ...interface{}) (string, error) {
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

func checkReturnStatus(response *http.Response) error {
	type Response struct {
		OK bool `json:"ok"`
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		err := fmt.Errorf("Error reading Slack API body: %s", err)
		return err
	}

	var resp Response
	err = json.Unmarshal(body, &resp)
	if err != nil {
		err := fmt.Errorf("Error parsing message response: %s", err)
		return err
	}
	return nil
}

func (s *Slack) RegisterEvent(f func(bot.Kind, msg.Message, ...interface{})) {
	s.event = f
}

func (s *Slack) sendMessageType(channel, message string, meMessage bool) (string, error) {
	postUrl := "https://slack.com/api/chat.postMessage"
	if meMessage {
		postUrl = "https://slack.com/api/chat.meMessage"
	}

	nick := s.config.Get("Nick", "bot")
	icon := s.config.Get("IconURL", "https://placekitten.com/128/128")

	resp, err := http.PostForm(postUrl,
		url.Values{"token": {s.token},
			"username": {nick},
			"icon_url": {icon},
			"channel":  {channel},
			"text":     {message},
		})

	if err != nil {
		log.Printf("Error sending Slack message: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatalf("Error reading Slack API body: %s", err)
	}

	log.Println(string(body))

	type MessageResponse struct {
		OK        bool   `json:"ok"`
		Timestamp string `json:"ts"`
		Message   struct {
			BotID string `json:"bot_id"`
		} `json:"message"`
	}

	var mr MessageResponse
	err = json.Unmarshal(body, &mr)
	if err != nil {
		log.Fatalf("Error parsing message response: %s", err)
	}

	if !mr.OK {
		return "", errors.New("failure response received")
	}

	s.myBotID = mr.Message.BotID

	return mr.Timestamp, err
}

func (s *Slack) sendMessage(channel, message string) (string, error) {
	log.Printf("Sending message to %s: %s", channel, message)
	identifier, err := s.sendMessageType(channel, message, false)
	return identifier, err
}

func (s *Slack) sendAction(channel, message string) (string, error) {
	log.Printf("Sending action to %s: %s", channel, message)
	identifier, err := s.sendMessageType(channel, "_"+message+"_", true)
	return identifier, err
}

func (s *Slack) replyToMessageIdentifier(channel, message, identifier string) (string, error) {
	nick := s.config.Get("Nick", "bot")
	icon := s.config.Get("IconURL", "https://placekitten.com/128/128")

	resp, err := http.PostForm("https://slack.com/api/chat.postMessage",
		url.Values{"token": {s.token},
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

func (s *Slack) replyToMessage(channel, message string, replyTo msg.Message) (string, error) {
	return s.replyToMessageIdentifier(channel, message, replyTo.AdditionalData["RAW_SLACK_TIMESTAMP"])
}

func (s *Slack) react(channel, reaction string, message msg.Message) (string, error) {
	log.Printf("Reacting in %s: %s", channel, reaction)
	resp, err := http.PostForm("https://slack.com/api/reactions.add",
		url.Values{"token": {s.token},
			"name":      {reaction},
			"channel":   {channel},
			"timestamp": {message.AdditionalData["RAW_SLACK_TIMESTAMP"]}})
	if err != nil {
		err := fmt.Errorf("reaction failed: %s", err)
		return "", err
	}
	return "", checkReturnStatus(resp)
}

func (s *Slack) edit(channel, newMessage, identifier string) (string, error) {
	log.Printf("Editing in (%s) %s: %s", identifier, channel, newMessage)
	resp, err := http.PostForm("https://slack.com/api/chat.update",
		url.Values{"token": {s.token},
			"channel": {channel},
			"text":    {newMessage},
			"ts":      {identifier}})
	if err != nil {
		err := fmt.Errorf("edit failed: %s", err)
		return "", err
	}
	return "", checkReturnStatus(resp)
}

func (s *Slack) GetEmojiList() map[string]string {
	return s.emoji
}

func (s *Slack) populateEmojiList() {
	resp, err := http.PostForm("https://slack.com/api/emoji.list",
		url.Values{"token": {s.token}})
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
		OK    bool              `json:"ok"`
		Emoji map[string]string `json:"emoji"`
	}

	var list EmojiListResponse
	err = json.Unmarshal(body, &list)
	if err != nil {
		log.Fatalf("Error parsing emoji list: %s", err)
	}
	s.emoji = list.Emoji
}

func (s *Slack) ping(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping := map[string]interface{}{"type": "ping", "time": time.Now().UnixNano()}
			if err := s.ws.Send(context.TODO(), ping); err != nil {
				panic(err)
			}
		}
	}
}

func (s *Slack) receiveMessage() (slackMessage, error) {
	m := slackMessage{}
	err := s.ws.Recv(context.TODO(), &m)
	if err != nil {
		log.Println("Error decoding WS message")
		panic(fmt.Errorf("%v\n%v", m, err))
	}
	return m, nil
}

// I think it's horseshit that I have to do this
func slackTStoTime(t string) time.Time {
	ts := strings.Split(t, ".")
	sec, _ := strconv.ParseInt(ts[0], 10, 64)
	nsec, _ := strconv.ParseInt(ts[1], 10, 64)
	return time.Unix(sec, nsec)
}

func (s *Slack) Serve() error {
	s.connect()
	s.populateEmojiList()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.ping(ctx)

	for {
		msg, err := s.receiveMessage()
		if err != nil && err == io.EOF {
			log.Fatalf("Slack API EOF")
		} else if err != nil {
			return fmt.Errorf("Slack API error: %s", err)
		}
		switch msg.Type {
		case "message":
			isItMe := msg.BotID != "" && msg.BotID == s.myBotID
			if !isItMe && !msg.Hidden && msg.ThreadTs == "" {
				m := s.buildMessage(msg)
				if m.Time.Before(s.lastRecieved) {
					log.Printf("Ignoring message: %+v\nlastRecieved: %v msg: %v", msg.ID, s.lastRecieved, m.Time)
				} else {
					s.lastRecieved = m.Time
					s.event(bot.Message, m)
				}
			} else if msg.ThreadTs != "" {
				//we're throwing away some information here by not parsing the correct reply object type, but that's okay
				s.event(bot.Reply, s.buildLightReplyMessage(msg), msg.ThreadTs)
			} else {
				log.Printf("THAT MESSAGE WAS HIDDEN: %+v", msg.ID)
			}
		case "error":
			log.Printf("Slack error, code: %d, message: %s", msg.Error.Code, msg.Error.Msg)
		case "": // what even is this?
		case "hello":
		case "presence_change":
		case "user_typing":
		case "reconnect_url":
		case "desktop_notification":
		case "pong":
			// squeltch this stuff
			continue
		default:
			log.Printf("Unhandled Slack message type: '%s'", msg.Type)
		}
	}
}

var urlDetector = regexp.MustCompile(`<(.+)://([^|^>]+).*>`)

// Convert a slackMessage to a msg.Message
func (s *Slack) buildMessage(m slackMessage) msg.Message {
	text := html.UnescapeString(m.Text)

	text = fixText(s.getUser, text)

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := m.SubType == "me_message"

	u, _ := s.getUser(m.User)
	if m.Username != "" {
		u = m.Username
	}

	tstamp := slackTStoTime(m.Ts)

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
		Host:    string(m.ID),
		Time:    tstamp,
		AdditionalData: map[string]string{
			"RAW_SLACK_TIMESTAMP": m.Ts,
		},
	}
}

func (s *Slack) buildLightReplyMessage(m slackMessage) msg.Message {
	text := html.UnescapeString(m.Text)

	text = fixText(s.getUser, text)

	isCmd, text := bot.IsCmd(s.config, text)

	isAction := m.SubType == "me_message"

	u, _ := s.getUser(m.User)
	if m.Username != "" {
		u = m.Username
	}

	tstamp := slackTStoTime(m.Ts)

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
		Host:    string(m.ID),
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
		url.Values{"token": {s.token}})
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
		url.Values{"token": {s.token}, "channel": {slackChanId}})
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
		url.Values{"token": {s.token}, "channel": {slackChanId}, "ts": {chanInfo.Channel.Latest.Ts}})
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
	token := s.token
	u := fmt.Sprintf("https://slack.com/api/rtm.connect?token=%s", token)
	resp, err := http.Get(u)
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
	s.id = rtm.Self.ID

	// This is hitting the rate limit, and it may not be needed
	//s.markAllChannelsRead()

	rtmURL, _ := url.Parse(rtm.URL)
	s.ws, err = websocket.Dial(context.TODO(), rtmURL)
	if err != nil {
		log.Fatal(err)
	}
}

// Get username for Slack user ID
func (s *Slack) getUser(id string) (string, bool) {
	if name, ok := s.users[id]; ok {
		return name, true
	}

	log.Printf("User %s not already found, requesting info", id)
	u := s.url + "users.info"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.token}, "user": {id}})
	if err != nil || resp.StatusCode != 200 {
		log.Printf("Error posting user info request: %d %s",
			resp.StatusCode, err)
		return "UNKNOWN", false
	}
	defer resp.Body.Close()
	var userInfo slackUserInfoResp
	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		log.Println("Error decoding response: ", err)
		return "UNKNOWN", false
	}
	s.users[id] = userInfo.User.Name
	return s.users[id], true
}

// Who gets usernames out of a channel
func (s *Slack) Who(id string) []string {
	log.Println("Who is queried for ", id)
	u := s.url + "channels.info"
	resp, err := http.PostForm(u,
		url.Values{"token": {s.token}, "channel": {id}})
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
		u, _ := s.getUser(member)
		handles = append(handles, u)
	}
	log.Printf("Returning %d handles", len(handles))
	return handles
}
