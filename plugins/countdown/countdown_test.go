package countdown

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/velour/catbase/bot"
)

func TestScheduleNewYears(t *testing.T) {
	mb := bot.NewMockBot()
	p := New(mb)
	functionTripped := false
	f := func() {
		functionTripped = true
	}

	// this is lazy, but I don't feel like fixing it
	nextYear = time.Now().Add(time.Millisecond * 5)
	p.scheduleNYE(f)
	time.Sleep(time.Millisecond * 10)

	assert.True(t, functionTripped)
}
