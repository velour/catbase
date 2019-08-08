package webshit

import (
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func init() {
	log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func makeWS(t *testing.T) *Webshit {
	db := sqlx.MustOpen("sqlite3", "file::memory:?mode=memory&cache=shared")
	w := New(db)
	assert.Equal(t, w.db, db)
	return w
}

func TestWebshit_GetWeekly(t *testing.T) {
	w := makeWS(t)
	weekly, pub, err := w.GetWeekly()
	t.Logf("Pub: %v", pub)
	assert.NotNil(t, pub)
	assert.Nil(t, err)
	assert.NotEmpty(t, weekly)
}

func TestWebshit_GetHeadlines(t *testing.T) {
	w := makeWS(t)
	headlines, err := w.GetHeadlines()
	assert.Nil(t, err)
	assert.NotEmpty(t, headlines)
}

func TestWebshit_getStoryByURL(t *testing.T) {
	w := makeWS(t)
	expected := "Developer Tropes: “Google Does It”"
	s, err := w.getStoryByURL("https://news.ycombinator.com/item?id=20432887")
	assert.Nil(t, err)
	assert.Equal(t, s.Title, expected)
}

func TestWebshit_getStoryByURL_BadURL(t *testing.T) {
	w := makeWS(t)
	_, err := w.getStoryByURL("https://google.com")
	assert.Error(t, err)
}

func TestWebshit_GetBalance(t *testing.T) {
	w := makeWS(t)
	expected := 100
	actual := w.GetBalance("foo")
	assert.Equal(t, expected, actual)
}

func TestWebshit_checkBids(t *testing.T) {
	w := makeWS(t)
	bids := []Bid{
		Bid{User: "foo", Title: "bar", URL: "https://baz/?id=1", Bid: 10},
		Bid{User: "foo", Title: "bar2", URL: "http://baz/?id=2", Bid: 10},
	}
	storyMap := map[string]Story{
		"1": Story{Title: "bar", URL: "http://baz/?id=1"},
	}
	result := w.checkBids(bids, storyMap)
	assert.Len(t, result, 1)
	if len(result) > 0 {
		assert.Len(t, result[0].WinningArticles, 1)
		assert.Len(t, result[0].LosingArticles, 1)
	}
}
