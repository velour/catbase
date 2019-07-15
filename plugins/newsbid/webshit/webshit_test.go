package webshit

import (
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func make(t *testing.T) *Webshit {
	db := sqlx.MustOpen("sqlite3", "file::memory:?mode=memory&cache=shared")
	w := New(db)
	assert.Equal(t, w.db, db)
	return w
}

func TestWebshit_GetWeekly(t *testing.T) {
	w := make(t)
	weekly, err := w.GetWeekly()
	assert.Nil(t, err)
	assert.NotEmpty(t, weekly)
}

func TestWebshit_GetHeadlines(t *testing.T) {
	w := make(t)
	headlines, err := w.GetHeadlines()
	assert.Nil(t, err)
	assert.NotEmpty(t, headlines)
}

func TestWebshit_getStoryByURL(t *testing.T) {
	w := make(t)
	expected := "Developer Tropes: “Google Does It”"
	s, err := w.getStoryByURL("https://news.ycombinator.com/item?id=20432887")
	assert.Nil(t, err)
	assert.Equal(t, s.Title, expected)
}

func TestWebshit_getStoryByURL_BadURL(t *testing.T) {
	w := make(t)
	_, err := w.getStoryByURL("https://google.com")
	assert.Error(t, err)
}

func TestWebshit_GetBalance(t *testing.T) {
	w := make(t)
	expected := 100
	actual := w.GetBalance("foo")
	assert.Equal(t, expected, actual)
}
