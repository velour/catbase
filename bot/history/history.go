package history

import (
	"container/ring"
	"fmt"

	"github.com/velour/catbase/bot/msg"
)

// History is a ring buffer of messages
type History struct {
	r *ring.Ring
}

// New returns a history of sz size
func New(sz int) *History {
	return &History{ring.New(sz)}
}

// Edit looks for an entry and ammends it
// returns error if the id is not found
func (h *History) Edit(id string, update *msg.Message) error {
	r := h.r
	for i := 0; i < r.Len(); i++ {
		m := r.Value.(*msg.Message)
		if m != nil && m.ID == id {
			r.Value = update
			return nil
		}
		r = r.Next()
	}
	return fmt.Errorf("entry not found")
}

// Append adds a message to the history
func (h *History) Append(m *msg.Message) {
	h.r.Value = m
	h.r = h.r.Next()
}

// Find looks for an entry by id and returns an error if not found
func (h *History) Find(id string) (*msg.Message, error) {
	r := h.r
	for i := 0; i < r.Len(); i++ {
		m := r.Value.(*msg.Message)
		if m != nil && m.ID == id {
			return m, nil
		}
		r = r.Next()
	}
	return nil, fmt.Errorf("entry not found")
}

// Last gets the last known message
func (h *History) Last() *msg.Message {
	return h.r.Prev().Value.(*msg.Message)
}

// LastInChannel searches backwards for the last message matching channel ch
func (h *History) LastInChannel(ch string) (*msg.Message, error) {
	r := h.r
	for i := 0; i < r.Len(); i++ {
		m := r.Value.(*msg.Message)
		if m != nil && m.Channel == ch {
			return m, nil
		}
		r = r.Prev()
	}
	return nil, fmt.Errorf("entry not found")
}

// InChannel returns all knows messages from channel ch in reverse order
func (h *History) InChannel(ch string) []*msg.Message {
	out := []*msg.Message{}
	r := h.r
	for i := 0; i < r.Len(); i++ {
		m := r.Value.(*msg.Message)
		if m != nil && m.Channel == ch {
			out = append(out, m)
		}
		r = r.Prev()
	}
	return out
}
