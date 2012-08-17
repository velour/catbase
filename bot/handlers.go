package bot

import (
	"fmt"
)
import irc "github.com/fluffle/goirc/client"

// Interface used for compatibility with the Plugin interface
type Handler interface {
	Message(user *User, channel, message string) bool
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

// Handles incomming PRIVMSG requests
func (b *Bot) Msg_recieved(conn *irc.Conn, line *irc.Line) {
	// Check for the user
	user := b.checkuser(line.Nick)

	channel := line.Args[0]
	message := line.Args[1]

	user.MessageLog = append(user.MessageLog, message)

	fmt.Printf("In %s, %s said: '%s'\n", channel, line.Nick, message)
	for _, p := range b.Plugins {
		if p.Message(user, channel, message) {
			break
		}
	}
}

// Take an input string and mutate it based on $vars in the string
func (b *Bot) filter(input string) string {
	return input
}

// Sends message to channel
func (b *Bot) SendMessage(channel, message string) {
	b.Conn.Privmsg(channel, message)
}
