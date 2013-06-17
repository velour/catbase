package bot

import (
	"code.google.com/p/velour/irc"
	"errors"
	"fmt"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(message Message) bool
	Event(kind string, message Message) bool
	BotMessage(message Message) bool
	Help(channel string, parts []string)
	RegisterWeb() *string
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
	botnick := strings.ToLower(b.Config.Nick)
	iscmd := false
	lowerMessage := strings.ToLower(message)

	if strings.HasPrefix(lowerMessage, cmdc) && len(cmdc) > 0 {
		iscmd = true
		message = message[len(cmdc):]
		// } else if match, _ := regexp.MatchString(rex, lowerMessage); match {
	} else if strings.HasPrefix(lowerMessage, botnick) &&
		len(lowerMessage) > len(botnick) &&
		(lowerMessage[len(botnick)] == ',' ||
			lowerMessage[len(botnick)] == ':') &&
		lowerMessage[len(botnick)+1] == ' ' {

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
func (b *Bot) buildMessage(conn *irc.Client, inMsg irc.Msg) Message {
	// Check for the user
	user := b.GetUser(inMsg.Origin)

	channel := inMsg.Args[0]
	if channel == b.Config.Nick {
		channel = inMsg.Args[0]
	}

	isAction := false
	var message string
	if len(inMsg.Args) > 1 {
		message = inMsg.Args[1]

		isAction = strings.HasPrefix(message, actionPrefix)
		if isAction {
			message = strings.TrimRight(message[len(actionPrefix):], "\x01")
			message = strings.TrimSpace(message)
		}

	}

	iscmd := false
	filteredMessage := message
	if !isAction {
		iscmd, filteredMessage = b.isCmd(message)
	}

	msg := Message{
		User:    user,
		Channel: channel,
		Body:    filteredMessage,
		Raw:     message,
		Command: iscmd,
		Action:  isAction,
		Time:    time.Now(),
		Host:    inMsg.Host,
	}

	return msg
}

func (b *Bot) LastMessage(channel string) (Message, error) {
	log := <-b.logOut
	if len(log) == 0 {
		return Message{}, errors.New("No messages found.")
	}
	for i := len(log) - 1; i >= 0; i-- {
		msg := log[i]
		if strings.ToLower(msg.Channel) == strings.ToLower(channel) {
			return msg, nil
		}
	}
	return Message{}, errors.New("No messages found.")
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
		nicks := b.Who(message.Channel)
		someone := nicks[rand.Intn(len(nicks))].Name
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
		"http://github.com/chrissexton/alepale", b.Version)
	b.SendMessage(channel, msg)
}

// Send our own musings to the plugins
func (b *Bot) selfSaid(channel, message string, action bool) {
	msg := Message{
		User:    &b.Me, // hack
		Channel: channel,
		Body:    message,
		Raw:     message, // hack
		Action:  action,
		Command: false,
		Time:    time.Now(),
		Host:    "0.0.0.0", // hack
	}

	for _, name := range b.PluginOrdering {
		p := b.Plugins[name]
		if p.BotMessage(msg) {
			break
		}
	}
}
