package goals

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/plugins/counter"
)

func TestRemainingSomeToGo(t *testing.T) {
	mb := bot.NewMockBot()

	i := counter.Item{
		DB:    mb.DB(),
		ID:    0,
		Nick:  "",
		Item:  "",
		Count: 190,
	}

	g := goal{
		ID:     0,
		Kind:   "",
		Who:    "",
		What:   "",
		Amount: 366,
		gp:     nil,
	}

	now = func() time.Time {
		return time.Date(2020, 07, 13, 11, 23, 00, 00, time.UTC)
	}

	p := New(mb)

	expected := 5
	actual := p.calculateRemaining(i, &g)

	assert.Equal(t, expected, actual)
}

func TestRemainingAheadOfCurve(t *testing.T) {
	mb := bot.NewMockBot()

	i := counter.Item{
		DB:    mb.DB(),
		ID:    0,
		Nick:  "",
		Item:  "",
		Count: 200,
	}

	g := goal{
		ID:     0,
		Kind:   "",
		Who:    "",
		What:   "",
		Amount: 366,
		gp:     nil,
	}

	now = func() time.Time {
		return time.Date(2020, 07, 13, 11, 23, 00, 00, time.UTC)
	}

	p := New(mb)

	expected := -5
	actual := p.calculateRemaining(i, &g)

	assert.Equal(t, expected, actual)
}
