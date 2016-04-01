// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package msglog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot/msg"
)

func TestNew(t *testing.T) {
	in := make(chan msg.Message)
	out := make(chan msg.Messages)
	logger := New(in, out)
	assert.NotNil(t, logger)
}

func TestRunNew(t *testing.T) {
	in := make(chan msg.Message)
	out := make(chan msg.Messages)
	RunNew(in, out)

	in <- msg.Message{}
	msg := <-out
	assert.Empty(t, out)
	assert.NotNil(t, msg)
}
