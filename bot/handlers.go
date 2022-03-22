// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot/msg"
)

func (b *bot) Receive(conn Connector, kind Kind, msg msg.Message, args ...any) bool {
	// msg := b.buildMessage(client, inMsg)
	// do need to look up user and fix it

	if kind == Edit {
		b.history.Edit(msg.ID, &msg)
	} else {
		b.history.Append(&msg)
	}

	if kind == Message && strings.HasPrefix(msg.Body, "help") && msg.Command {
		parts := strings.Fields(strings.ToLower(msg.Body))
		b.checkHelp(conn, msg.Channel, parts)
		log.Debug().Msg("Handled a help, returning")
		goto RET
	}

	for _, name := range b.pluginOrdering {
		if b.OnBlacklist(msg.Channel, pluginNameStem(name)) || !b.onWhitelist(pluginNameStem(name)) {
			continue
		}
		if b.runCallback(conn, b.plugins[name], kind, msg, args...) {
			goto RET
		}
	}

RET:
	b.logIn <- msg
	return true
}

func ParseValues(r *regexp.Regexp, body string) RegexValues {
	out := RegexValues{}
	subs := r.FindStringSubmatch(body)
	if len(subs) == 0 {
		return out
	}
	for i, n := range r.SubexpNames() {
		out[n] = subs[i]
	}
	return out
}

func (b *bot) runCallback(conn Connector, plugin Plugin, evt Kind, message msg.Message, args ...any) bool {
	t := reflect.TypeOf(plugin).String()
	for _, spec := range b.callbacks[t][evt] {
		if spec.Regex.MatchString(message.Body) {
			req := Request{
				Conn:   conn,
				Kind:   evt,
				Msg:    message,
				Values: ParseValues(spec.Regex, message.Body),
				Args:   args,
			}
			if spec.Handler(req) {
				return true
			}
		}
	}
	return false
}

// Send a message to the connection
func (b *bot) Send(conn Connector, kind Kind, args ...any) (string, error) {
	if b.quiet {
		return "", nil
	}
	return conn.Send(kind, args...)
}

func (b *bot) GetEmojiList() map[string]string {
	return b.conn.GetEmojiList()
}

// Checks to see if the user is asking for help, returns true if so and handles the situation.
func (b *bot) checkHelp(conn Connector, channel string, parts []string) {
	if len(parts) == 1 {
		// just print out a list of help topics
		topics := "Help topics: about variables"
		for _, name := range b.GetWhitelist() {
			name = pluginNameStem(name)
			topics = fmt.Sprintf("%s, %s", topics, name)
		}
		b.Send(conn, Message, channel, topics)
	} else {
		// trigger the proper plugin's help response
		if parts[1] == "about" {
			b.Help(conn, channel, parts)
			return
		}
		if parts[1] == "variables" {
			b.listVars(conn, channel, parts)
			return
		}
		for name, plugin := range b.plugins {
			if strings.HasPrefix(name, "*"+parts[1]) {
				if b.runCallback(conn, plugin, Help, msg.Message{Channel: channel}, channel, parts) {
					return
				} else {
					msg := fmt.Sprintf("I'm sorry, I don't know how to help you with %s.", parts[1])
					b.Send(conn, Message, channel, msg)
					return
				}
			}
		}
		msg := fmt.Sprintf("I'm sorry, I don't know what %s is!", strings.Join(parts, " "))
		b.Send(conn, Message, channel, msg)
	}
}

func (b *bot) LastMessage(channel string) (msg.Message, error) {
	log := <-b.logOut
	if len(log) == 0 {
		return msg.Message{}, errors.New("No messages found.")
	}
	for i := len(log) - 1; i >= 0; i-- {
		msg := log[i]
		if strings.ToLower(msg.Channel) == strings.ToLower(channel) {
			return msg, nil
		}
	}
	return msg.Message{}, errors.New("No messages found.")
}

// Take an input string and mutate it based on $vars in the string
func (b *bot) Filter(message msg.Message, input string) string {
	if strings.Contains(input, "$NICK") {
		nick := strings.ToUpper(message.User.Name)
		input = strings.Replace(input, "$NICK", nick, -1)
	}

	// Let's be bucket compatible for this var
	input = strings.Replace(input, "$who", "$nick", -1)
	if strings.Contains(input, "$nick") {
		nick := message.User.Name
		input = strings.Replace(input, "$nick", nick, -1)
	}

	for strings.Contains(input, "$someone") {
		nicks := b.Who(message.Channel)
		someone := nicks[rand.Intn(len(nicks))].Name
		input = strings.Replace(input, "$someone", someone, 1)
	}

	for strings.Contains(input, "$digit") {
		num := strconv.Itoa(rand.Intn(9))
		input = strings.Replace(input, "$digit", num, 1)
	}

	for strings.Contains(input, "$nonzero") {
		num := strconv.Itoa(rand.Intn(8) + 1)
		input = strings.Replace(input, "$nonzero", num, 1)
	}

	for strings.Contains(input, "$time") {
		hrs := rand.Intn(23)
		mins := rand.Intn(59)
		input = strings.Replace(input, "$time", fmt.Sprintf("%0.2d:%0.2d", hrs, mins), 1)
	}

	for strings.Contains(input, "$now") {
		now := time.Now().Format("15:04")
		input = strings.Replace(input, "$now", now, 1)
	}

	for strings.Contains(input, "$msg") {
		msg := message.Body
		input = strings.Replace(input, "$msg", msg, 1)
	}

	r, err := regexp.Compile("\\$[A-z]+")
	if err != nil {
		panic(err)
	}

	for _, f := range b.filters {
		input = f(input)
	}

	varname := r.FindString(input)
	blacklist := make(map[string]bool)
	blacklist["$and"] = true
	for len(varname) > 0 && !blacklist[varname] {
		text, err := b.getVar(varname)
		if err != nil {
			blacklist[varname] = true
			continue
		}
		input = strings.Replace(input, varname, text, 1)
		varname = r.FindString(input)
	}

	return input
}

func (b *bot) getVar(varName string) (string, error) {
	var text string
	err := b.DB().Get(&text, `select value from variables where name=? order by random() limit 1`, varName)
	switch {
	case err == sql.ErrNoRows:
		return "", fmt.Errorf("No factoid found")
	case err != nil:
		log.Fatal().Err(err).Msg("getVar error")
	}
	return text, nil
}

func (b *bot) listVars(conn Connector, channel string, parts []string) {
	var variables []string
	err := b.DB().Select(&variables, `select name from variables group by name`)
	if err != nil {
		log.Fatal().Err(err)
	}
	msg := "I know: $who, $someone, $digit, $nonzero, $time, $now, $msg"
	if len(variables) > 0 {
		msg += ", " + strings.Join(variables, ", ")
	}
	b.Send(conn, Message, channel, msg)
}

func (b *bot) Help(conn Connector, channel string, parts []string) {
	msg := fmt.Sprintf("Hi, I'm based on godeepintir version %s. I'm written in Go, and you "+
		"can find my source code on the internet here: "+
		"http://github.com/velour/catbase", b.version)
	b.Send(conn, Message, channel, msg)
}

// Send our own musings to the plugins
func (b *bot) selfSaid(conn Connector, channel, message string, action bool) {
	msg := msg.Message{
		User:    &b.me, // hack
		Channel: channel,
		Body:    message,
		Raw:     message, // hack
		Action:  action,
		Command: false,
		Time:    time.Now(),
		Host:    "0.0.0.0", // hack
	}

	for _, name := range b.pluginOrdering {
		if b.runCallback(conn, b.plugins[name], SelfMessage, msg) {
			return
		}
	}
}

// PubToASub sends updates to subscribed URLs
func (b *bot) PubToASub(subject string, payload any) {
	key := fmt.Sprintf("pubsub.%s.url", subject)
	subs := b.config.GetArray(key, []string{})
	if len(subs) == 0 {
		return
	}
	encodedBody, _ := json.Marshal(struct {
		Payload any `json:"payload"`
	}{payload})
	body := bytes.NewBuffer(encodedBody)
	for _, url := range subs {
		r, err := http.Post(url, "text/json", body)
		if err != nil {
			log.Error().Err(err).Msg("")
		} else {
			body, _ := ioutil.ReadAll(r.Body)
			log.Debug().Msgf("Response body: %s", string(body))
		}
	}
}
