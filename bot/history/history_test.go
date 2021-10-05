package history

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/velour/catbase/bot/msg"
)

var sampleMessages []*msg.Message

func init() {
	sampleMessages = []*msg.Message{}
	for i := 0; i < 100; i++ {
		txt := fmt.Sprintf("Message #%d", i)
		m := &msg.Message{
			ID:   txt,
			Body: txt,
		}

		sampleMessages = append(sampleMessages, m)
	}
}

func TestAppend(t *testing.T) {
	h := New(10)
	for i := 0; i < 10; i++ {
		h.Append(sampleMessages[i])
	}
	previous := h.r.Prev().Value.(*msg.Message)
	assert.Equal(t, sampleMessages[9].ID, previous.ID)
}

func TestFindExists(t *testing.T) {
	h := New(10)
	for i := 0; i < 10; i++ {
		h.Append(sampleMessages[i])
	}
	id := sampleMessages[5].ID
	elt, err := h.Find(id)
	assert.Nil(t, err)
	assert.Equal(t, id, elt.ID)
}

func TestFindMissing(t *testing.T) {
	h := New(10)
	for i := 0; i < 10; i++ {
		h.Append(sampleMessages[i])
	}
	id := sampleMessages[15].ID
	elt, err := h.Find(id)
	assert.NotNil(t, err)
	assert.Nil(t, elt)
}

func TestEditExists(t *testing.T) {
	h := New(10)
	for i := 0; i < 10; i++ {
		h.Append(sampleMessages[i])
	}
	id := sampleMessages[5].ID
	m := sampleMessages[15]
	m.ID = id
	err := h.Edit(id, m)
	assert.Nil(t, err)
	actual, err := h.Find(id)
	assert.Nil(t, err)
	assert.Equal(t, m.Body, actual.Body)
}

func TestEditMissing(t *testing.T) {
	h := New(10)
	for i := 0; i < 10; i++ {
		h.Append(sampleMessages[i])
	}
	id := sampleMessages[10].ID
	m := sampleMessages[6]
	t.Logf("id: %s, editID: %s", id, m.ID)
	err := h.Edit(id, m)
	assert.NotNil(t, err)
}
