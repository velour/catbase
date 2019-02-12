// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/velour/catbase/bot/msg"
)

func (b *bot) Receive(kind Kind, msg msg.Message, args ...interface{}) bool {
	log.Println("Received event: ", msg)

	// msg := b.buildMessage(client, inMsg)
	// do need to look up user and fix it
	if kind == Message && strings.HasPrefix(msg.Body, "help") && msg.Command {
		parts := strings.Fields(strings.ToLower(msg.Body))
		b.checkHelp(msg.Channel, parts)
		goto RET
	}

	for _, name := range b.pluginOrdering {
		if b.runCallback(b.plugins[name], kind, msg, args...) {
			goto RET
		}
	}

RET:
	b.logIn <- msg
	return true
}

func (b *bot) runCallback(plugin Plugin, evt Kind, message msg.Message, args ...interface{}) bool {
	t := reflect.TypeOf(plugin)
	for _, cb := range b.callbacks[t][evt] {
		if cb(evt, message, args...) {
			return true
		}
	}
	return false
}

// Send a message to the connection
func (b *bot) Send(kind Kind, args ...interface{}) (string, error) {
	return b.conn.Send(kind, args...)
}

func (b *bot) GetEmojiList() map[string]string {
	return b.conn.GetEmojiList()
}

// Checks to see if the user is asking for help, returns true if so and handles the situation.
func (b *bot) checkHelp(channel string, parts []string) {
	if len(parts) == 1 {
		// just print out a list of help topics
		topics := "Help topics: about variables"
		for name, _ := range b.plugins {
			name = strings.Split(strings.TrimPrefix(name, "*"), ".")[0]
			topics = fmt.Sprintf("%s, %s", topics, name)
		}
		b.Send(Message, channel, topics)
	} else {
		// trigger the proper plugin's help response
		if parts[1] == "about" {
			b.Help(channel, parts)
			return
		}
		if parts[1] == "variables" {
			b.listVars(channel, parts)
			return
		}
		for name, plugin := range b.plugins {
			if strings.HasPrefix(name, "*"+parts[1]) {
				if b.runCallback(plugin, Help, msg.Message{Channel: channel}, channel, parts) {
					return
				} else {
					msg := fmt.Sprintf("I'm sorry, I don't know how to help you with %s.", parts[1])
					b.Send(Message, channel, msg)
					return
				}
			}
		}
		msg := fmt.Sprintf("I'm sorry, I don't know what %s is!", strings.Join(parts, " "))
		b.Send(Message, channel, msg)
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
		log.Fatal("getVar error: ", err)
	}
	return text, nil
}

func (b *bot) listVars(channel string, parts []string) {
	var variables []string
	err := b.DB().Select(&variables, `select name from variables group by name`)
	if err != nil {
		log.Fatal(err)
	}
	msg := "I know: $who, $someone, $digit, $nonzero"
	if len(variables) > 0 {
		msg += ", " + strings.Join(variables, ", ")
	}
	b.Send(Message, channel, msg)
}

func (b *bot) Help(channel string, parts []string) {
	msg := fmt.Sprintf("Hi, I'm based on godeepintir version %s. I'm written in Go, and you "+
		"can find my source code on the internet here: "+
		"http://github.com/velour/catbase", b.version)
	b.Send(Message, channel, msg)
}

// Send our own musings to the plugins
func (b *bot) selfSaid(channel, message string, action bool) {
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
		if b.runCallback(b.plugins[name], SelfMessage, msg) {
			return
		}
	}
}
