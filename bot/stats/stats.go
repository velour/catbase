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

func (s Stats) Uptime() time.Duration {
	return time.Now().Sub(s.startTime)
}
