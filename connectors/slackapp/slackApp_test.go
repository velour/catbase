package slackapp

import (
	"container/ring"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupeNoDupes(t *testing.T) {
	buf := ring.New(3)
	for i := 0; i < 3; i++ {
		buf.Value = ""
		buf = buf.Next()
	}
	s := SlackApp{msgIDBuffer: buf}
	expected := []bool{
		false,
		false,
		false,
		false,
		false,
	}

	actuals := []bool{}
	actuals = append(actuals, s.checkRingOrAdd("a"))
	actuals = append(actuals, s.checkRingOrAdd("b"))
	actuals = append(actuals, s.checkRingOrAdd("c"))
	actuals = append(actuals, s.checkRingOrAdd("d"))
	actuals = append(actuals, s.checkRingOrAdd("e"))

	assert.ElementsMatch(t, expected, actuals)
}

func TestDedupeWithDupes(t *testing.T) {
	buf := ring.New(3)
	for i := 0; i < 3; i++ {
		buf.Value = ""
		buf = buf.Next()
	}
	s := SlackApp{msgIDBuffer: buf}
	expected := []bool{
		false,
		false,
		true,
		false,
		true,
	}

	actuals := []bool{}
	actuals = append(actuals, s.checkRingOrAdd("a"))
	actuals = append(actuals, s.checkRingOrAdd("b"))
	actuals = append(actuals, s.checkRingOrAdd("a"))
	actuals = append(actuals, s.checkRingOrAdd("d"))
	actuals = append(actuals, s.checkRingOrAdd("d"))

	assert.ElementsMatch(t, expected, actuals)
}
