// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package msg

import (
	"time"

	"github.com/velour/catbase/bot/user"
)

type Log Messages
type Messages []Message

type Message struct {
	User          *user.User
	Channel, Body string
	Raw           string
	Command       bool
	Action        bool
	Time          time.Time
	Host          string
	AdditionalData map[string]string
}
