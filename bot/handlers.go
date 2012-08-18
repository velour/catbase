package bot

import (
	"fmt"
	"strings"
)
import irc "github.com/fluffle/goirc/client"

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(message Message) bool
	Help(channel string, parts []string)
}

// Checks to see if our user exists and if any changes have occured to it
// This uses a linear scan for now, oh well.
func (b *Bot) checkuser(nick string) *User {
	var user *User = nil
	for _, usr := range b.Users {
		if usr.Name == nick {
			user = &usr
		}
	}
	if user == nil {
		fmt.Println("Making a new user")
		user = &User{
			Name:       nick,
			Alts:       make([]string, 1),
			MessageLog: make([]string, 50),
		}
		b.Users = append(b.Users, *user)
	}

	return user
}

// Checks to see if the user is asking for help, returns true if so and handles the situation.
func (b *Bot) checkHelp(channel string, parts []string) {
	if len(parts) == 1 {
		// just print out a list of help topics
		topics := "Help topics: about"
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
		fmt.Println("Got a commandchar prefix")
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

// Handles incomming PRIVMSG requests
func (b *Bot) MsgRecieved(conn *irc.Conn, line *irc.Line) {
	// Check for the user
	user := b.checkuser(line.Nick)

	channel := line.Args[0]
	if channel == conn.Me.Nick {
		channel = line.Nick
	}
	message := line.Args[1]
	iscmd := false
	iscmd, message = b.isCmd(message)
	parts := strings.Fields(strings.ToLower(message))

	if iscmd {
		fmt.Println("Hey, I got a command!")
	}

	user.MessageLog = append(user.MessageLog, message)

	if strings.HasPrefix(message, "help") {
		b.checkHelp(channel, parts)
		return
	}

	fmt.Printf("In %s, %s said: '%s'\n", channel, line.Nick, message)
	for _, p := range b.Plugins {
		msg := Message{
			User:    user,
			Channel: channel,
			Body:    message,
			Command: iscmd,
		}
		if p.Message(msg) {
			break
		}
	}
}

// Take an input string and mutate it based on $vars in the string
func (b *Bot) Filter(message Message, input string) string {
	if strings.Contains(input, "$NICK") {
		nick := strings.ToUpper(message.User.Name)
		input = strings.Replace(input, "$NICK", nick, -1)
	} else if strings.Contains(input, "$nick") {
		nick := message.User.Name
		input = strings.Replace(input, "$nick", nick, -1)
	}
	return input
}

// Sends message to channel
func (b *Bot) SendMessage(channel, message string) {
	b.Conn.Privmsg(channel, message)
}

func (b *Bot) Help(channel string, parts []string) {
	msg := fmt.Sprintf("Hi, I'm based on godeepintir version %s. I'm written in Go, and you "+
		"can find my source code on the internet here: "+
		"http://bitbucket.org/phlyingpenguin/godeepintir", b.Version)
	b.SendMessage(channel, msg)
}
