// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package msglog

import "github.com/velour/catbase/bot/msg"

type MsgLogger struct {
	in      <-chan msg.Message
	out     chan<- msg.Messages
	entries msg.Messages
}

func New(in chan msg.Message, out chan msg.Messages) *MsgLogger {
	return &MsgLogger{in, out, make(msg.Messages, 0)}
}

func RunNew(in chan msg.Message, out chan msg.Messages) {
	logger := New(in, out)
	go logger.Run()
}

func (l *MsgLogger) sendEntries() {
	l.out <- l.entries
}

func (l *MsgLogger) Run() {
	var msg msg.Message
	for {
		select {
		case msg = <-l.in:
			l.entries = append(l.entries, msg)
		case l.out <- l.entries:
			go l.sendEntries()
		}
	}
}
