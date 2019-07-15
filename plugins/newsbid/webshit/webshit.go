package webshit

import (
	"bytes"
	"fmt"
	"github.com/PaulRosset/go-hacknews"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"
	"net/url"
)

type Webshit struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Webshit {
	w := &Webshit{db}
	w.setup()
	return w
}

// setup will create any necessary SQL tables and populate them with minimal data
func (w *Webshit) setup() {
}

// GetHeadlines will return the current possible news headlines for bidding
func (w *Webshit) GetHeadlines() ([]hacknews.Post, error) {
	news := hacknews.Initializer{Story: "topstories", NbPosts: 10}
	ids, err := news.GetCodesStory()
	if err != nil {
		return nil, err
	}
	posts, err := news.GetPostStory(ids)
	if err != nil {
		return nil, err
	}
	return posts, nil
}

type Weekly []string

// GetWeekly will return the headlines in the last webshit weekly report
func (w *Webshit) GetWeekly() (Weekly, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("http://n-gate.com/hackernews/index.rss")
	if err != nil {
		return nil, err
	}
	if len(feed.Items) <= 0 {
		return nil, fmt.Errorf("no webshit weekly found")
	}

	buf := bytes.NewBufferString(feed.Items[0].Description)
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		return nil, err
	}

	var items []string
	doc.Find(".storylink").Each(func(i int, s *goquery.Selection) {
		items = append(items, s.Find("a").Text())
	})

	return items, nil
}

// GetBalances returns the current balance for all known users
// Any unknown user has a default balance on their first bid
func (w *Webshit) GetBalances() {
}

// Bid allows a user to place a bid on a particular story
func (w *Webshit) Bid(user string, amount int, URL url.URL) error {
	return nil
}
