package webshit

import (
	"github.com/jmoiron/sqlx"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func make(t *testing.T) *Webshit {
	db := sqlx.MustOpen("sqlite3", "file::memory:?mode=memory&cache=shared")
	w := New(db)
	if w.db != db {
		t.Fail()
	}
	return w
}

func TestWebshit_GetWeekly(t *testing.T) {
	w := make(t)
	weekly, err := w.GetWeekly()
	if err != nil {
		t.Errorf("Could not get weekly: %s", err)
		t.Fail()
	}
	if len(weekly) < 5 {
		t.Errorf("Weekly content:\n%+v", weekly)
		t.Fail()
	}
}
