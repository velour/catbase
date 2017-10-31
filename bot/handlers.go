// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/velour/catbase/bot/msg"
)

// Handles incomming PRIVMSG requests
func (b *bot) MsgReceived(msg msg.Message) {
	log.Println("Received message: ", msg)

	// msg := b.buildMessage(client, inMsg)
	// do need to look up user and fix it

	log.Println(msg.User.Name)

	if strings.HasPrefix(msg.Body, "help ") && msg.Command {
		parts := strings.Fields(strings.ToLower(msg.Body))
		b.checkHelp(msg.Channel, parts)
		goto RET
	}

	for _, name := range b.pluginOrdering {
		p := b.plugins[name]
		if p.Message(msg) {
			break
		}
	}

RET:
	b.logIn <- msg
	return
}

// Handle incoming events
func (b *bot) EventReceived(msg msg.Message) {
	log.Println("Received event: ", msg)
	//msg := b.buildMessage(conn, inMsg)
	for _, name := range b.pluginOrdering {
		p := b.plugins[name]
		if p.Event(msg.Body, msg) { // TODO: could get rid of msg.Body
			break
		}
	}
}

func (b *bot) SendMessage(channel, message string) string {
	return b.conn.SendMessage(channel, message)
}

func (b *bot) SendAction(channel, message string) string {
	return b.conn.SendAction(channel, message)
}

func (b *bot) React(channel, reaction string, message msg.Message) bool {
	return b.conn.React(channel, reaction, message)
}

func (b *bot) Edit(channel, newMessage, identifier string) bool {
	return b.conn.Edit(channel, newMessage, identifier)
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
			topics = fmt.Sprintf("%s, %s", topics, name)
		}
		b.SendMessage(channel, topics)
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
		plugin := b.plugins[parts[1]]
		if plugin != nil {
			plugin.Help(channel, parts)
		} else {
			msg := fmt.Sprintf("I'm sorry, I don't know what %s is!", parts[1])
			b.SendMessage(channel, msg)
		}
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
	rand.Seed(time.Now().Unix())

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
	err := b.db.Get(&text, `select value from variables where name=? order by random() limit 1`, varName)
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
	err := b.db.Select(&variables, `select name from variables group by name`)
	if err != nil {
		log.Fatal(err)
	}
	msg := "I know: $who, $someone, $digit, $nonzero"
	if len(variables) > 0 {
		msg += ", " + strings.Join(variables, ", ")
	}
	b.SendMessage(channel, msg)
}

func (b *bot) Help(channel string, parts []string) {
	msg := fmt.Sprintf("Hi, I'm based on godeepintir version %s. I'm written in Go, and you "+
		"can find my source code on the internet here: "+
		"http://github.com/velour/catbase", b.version)
	b.SendMessage(channel, msg)
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
		p := b.plugins[name]
		if p.BotMessage(msg) {
			break
		}
	}
}
