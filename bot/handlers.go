package bot

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
	// "labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"regexp"
)
import irc "github.com/fluffle/goirc/client"

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(message Message) bool
	Event(kind string, message Message) bool
	Help(channel string, parts []string)
}

// Checks to see if our user exists and if any changes have occured to it
// This uses a linear scan for now, oh well.
func (b *Bot) checkuser(nick string) *User {
	var user *User = nil
	for _, usr := range b.Users {
		if usr.Name == nick {
			user = &usr
			break
		}
	}
	if user == nil {
		isadmin := false
		for _, u := range b.Config.Admins {
			if nick == u {
				isadmin = true
			}
		}
		user = &User{
			Name:       nick,
			Alts:       make([]string, 1),
			MessageLog: make([]string, 50),
			Admin:      isadmin,
		}
		b.Users = append(b.Users, *user)
	}

	return user
}

// Checks to see if the user is asking for help, returns true if so and handles the situation.
func (b *Bot) checkHelp(channel string, parts []string) {
	if len(parts) == 1 {
		// just print out a list of help topics
		topics := "Help topics: about variables"
		for name, _ := range b.Plugins {
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
		plugin := b.Plugins[parts[1]]
		if plugin != nil {
			plugin.Help(channel, parts)
		} else {
			msg := fmt.Sprintf("I'm sorry, I don't know what %s is!", parts[1])
			b.SendMessage(channel, msg)
		}
	}
}

// Checks if message is a command and returns its curtailed version
func (b *Bot) isCmd(message string) (bool, string) {
	cmdc := b.Config.CommandChar
	botnick := b.Conn.Me.Nick
	iscmd := false

	if strings.HasPrefix(message, cmdc) && len(cmdc) > 0 {
		iscmd = true
		message = message[len(cmdc):]
	} else if strings.HasPrefix(message, botnick) {
		iscmd = true
		message = message[len(botnick):]

		// trim off the customary addressing punctuation
		if message[0] == ':' || message[0] == ',' {
			message = message[1:]
		}
	}

	// trim off any whitespace left on the message
	message = strings.TrimSpace(message)

	return iscmd, message
}

// Builds our internal message type out of a Conn & Line from irc
func (b *Bot) buildMessage(conn *irc.Conn, line *irc.Line) Message {
	// Check for the user
	user := b.checkuser(line.Nick)

	channel := line.Args[0]
	if channel == conn.Me.Nick {
		channel = line.Nick
	}

	isaction := line.Cmd == "ACTION"

	var message string
	if len(line.Args) > 1 {
		message = line.Args[1]
	}
	iscmd := false
	filteredMessage := message
	if !isaction {
		iscmd, filteredMessage = b.isCmd(message)
	}

	user.MessageLog = append(user.MessageLog, message)

	msg := Message{
		User:    user,
		Channel: channel,
		Body:    filteredMessage,
		Raw:     message,
		Command: iscmd,
		Action:  isaction,
		Time:    time.Now(),
	}
	return msg
}

func (b *Bot) LastMessage() (Message, error) {
	log := <-b.logOut
	if len(log) == 0 {
		return Message{}, errors.New("No messages found.")
	}
	return log[len(log)-1], nil
}

// Handles incomming PRIVMSG requests
func (b *Bot) MsgRecieved(conn *irc.Conn, line *irc.Line) {
	msg := b.buildMessage(conn, line)

	if strings.HasPrefix(msg.Body, "help") && msg.Command {
		parts := strings.Fields(strings.ToLower(msg.Body))
		b.checkHelp(msg.Channel, parts)
		goto RET
	}

	for _, name := range b.PluginOrdering {
		p := b.Plugins[name]
		if p.Message(msg) {
			break
		}
	}

RET:
	b.logIn <- msg
	return
}

// Take an input string and mutate it based on $vars in the string
func (b *Bot) Filter(message Message, input string) string {
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

	if strings.Contains(input, "$someone") {
		someone := b.Users[rand.Intn(len(b.Users))].Name
		input = strings.Replace(input, "$someone", someone, -1)
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

	varname := r.FindString(input)
	blacklist := make(map[string]bool)
	blacklist["$and"] = true
	for len(varname) > 0 && !blacklist[varname] {
		var result []Variable
		b.varColl.Find(bson.M{"variable": varname}).All(&result)
		if len(result) == 0 {
			blacklist[varname] = true
			continue
		}
		variable := result[rand.Intn(len(result))]
		input = strings.Replace(input, varname, variable.Value, 1)
		varname = r.FindString(input)
	}

	return input
}

func (b *Bot) listVars(channel string, parts []string) {
	var result []string
	err := b.varColl.Find(bson.M{}).Distinct("variable", &result)
	if err != nil {
		panic(err)
	}
	msg := "I know: $who, $someone, $digit, $nonzero"
	for _, variable := range result {
		msg = fmt.Sprintf("%s, %s", msg, variable)
	}
	b.SendMessage(channel, msg)
}

func (b *Bot) Help(channel string, parts []string) {
	msg := fmt.Sprintf("Hi, I'm based on godeepintir version %s. I'm written in Go, and you "+
		"can find my source code on the internet here: "+
		"http://bitbucket.org/phlyingpenguin/godeepintir", b.Version)
	b.SendMessage(channel, msg)
}

func (b *Bot) UserJoined(conn *irc.Conn, line *irc.Line) {
	msg := b.buildMessage(conn, line)
	for _, name := range b.PluginOrdering {
		p := b.Plugins[name]
		if p.Event(line.Cmd, msg) {
			break
		}
	}
}
