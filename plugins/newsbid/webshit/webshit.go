package webshit

import (
	"bytes"
	"fmt"
	"github.com/PaulRosset/go-hacknews"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"strings"
)

type Webshit struct {
	db *sqlx.DB
}

type Weekly []string

type Story struct {
	Title string
	URL   string
}

type Bid struct {
	ID    int
	User  string
	Title string
	URL   string
	Bid   int
}

func New(db *sqlx.DB) *Webshit {
	w := &Webshit{db}
	w.setup()
	return w
}

// setup will create any necessary SQL tables and populate them with minimal data
func (w *Webshit) setup() {
	if _, err := w.db.Exec(`create table if not exists webshit_bids (
		id integer primary key,
		user string,
		title string,
		url string,
		bid integer
	)`); err != nil {
		log.Fatal().Err(err)
	}
	if _, err := w.db.Exec(`create table if not exists webshit_balances (
		user string primary key,
		balance int,
		score int
	)`); err != nil {
		log.Fatal().Err(err)
	}
}

// GetHeadlines will return the current possible news headlines for bidding
func (w *Webshit) GetHeadlines() ([]Story, error) {
	news := hacknews.Initializer{Story: "topstories", NbPosts: 10}
	ids, err := news.GetCodesStory()
	if err != nil {
		return nil, err
	}
	posts, err := news.GetPostStory(ids)
	if err != nil {
		return nil, err
	}
	var stories []Story
	for _, p := range posts {
		stories = append(stories, Story{
			Title: p.Title,
			URL:   p.Url,
		})
	}
	return stories, nil
}

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
func (w *Webshit) GetBalance(user string) int {
	q := `select balance from webshit_balances where user=?`
	var balance int
	err := w.db.Get(&balance, q, user)
	if err != nil {
		return 100
	}
	return balance
}

// Bid allows a user to place a bid on a particular story
func (w *Webshit) Bid(user string, amount int, URL string) error {
	bal := w.GetBalance(user)
	if bal < amount {
		return fmt.Errorf("cannot bid more than balance, %d", bal)
	}
	story, err := w.getStoryByURL(URL)
	if err != nil {
		return err
	}

	// Need a transaction here to deduct from the users balance (or create it)
	_, err = w.db.Exec(`insert into webshit_bids (user,title,url,bid) values (?,?,?,?)`,
		user, story.Title, story.URL, amount)

	return err
}

// getStoryByURL scrapes the URL for a title
func (w *Webshit) getStoryByURL(URL string) (Story, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return Story{}, err
	}
	if u.Host != "news.ycombinator.com" {
		return Story{}, fmt.Errorf("expected HN link")
	}
	res, err := http.Get(URL)
	if err != nil {
		return Story{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return Story{}, fmt.Errorf("bad response code: %d", res.StatusCode)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return Story{}, err
	}

	// Find the review items
	title := doc.Find("title").Text()
	title = strings.ReplaceAll(title, " | Hacker News", "")
	return Story{
		Title: title,
		URL:   URL,
	}, nil
}
