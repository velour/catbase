package stats

import (
	"time"
)

type Stats struct {
	MessagesSent int
	MessagesRcv  int
	startTime    time.Time
}

func New() *Stats {
	return &Stats{startTime: time.Now()}
}

func (s Stats) Uptime() string {
	return time.Now().Sub(s.startTime).Truncate(time.Second).String()
}
