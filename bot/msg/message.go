// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package msg

import (
	"time"

	"github.com/velour/catbase/bot/user"
)

type Log Messages
type Messages []Message

type Message struct {
	User *user.User
	// With Slack, channel is the ID of a channel
	Channel string
	// With slack, channelName is the nice name of a channel
	ChannelName    string
	Body           string
	IsIM           bool
	Raw            interface{}
	Command        bool
	Action         bool
	Time           time.Time
	Host           string
	AdditionalData map[string]string
}
