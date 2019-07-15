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
	"time"
)

type Webshit struct {
	db *sqlx.DB
}

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

type WeeklyResult struct {
	User string
	Won  int
	Lost int
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
		bid integer,
		created integer
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

func (w *Webshit) Check() (map[string]WeeklyResult, error) {
	stories, published, err := w.GetWeekly()
	if err != nil {
		return nil, err
	}

	var bids []Bid
	if err = w.db.Get(&bids, `select * from webshit_bids where created < ?`,
		published.Unix()); err != nil {
		return nil, err
	}

	// Assuming no bids earlier than the weekly means there hasn't been a new weekly
	if len(bids) == 0 {
		return nil, nil
	}

	storyMap := map[string]Story{}
	for _, s := range stories {
		storyMap[s.Title] = s
	}

	wr := w.checkBids(bids, storyMap)

	// Update all balance scores in a tx
	if err := w.updateScores(wr); err != nil {
		return nil, err
	}

	// Delete all those bids
	if _, err = w.db.Exec(`delete from webshit_bids where created < ?`,
		published.Unix()); err != nil {
		return nil, err
	}

	// Set all balances to 100
	if _, err = w.db.Exec(`update webshit_balances set balance=100`); err != nil {
		return nil, err
	}

	return wr, nil
}

func (w *Webshit) checkBids(bids []Bid, storyMap map[string]Story) map[string]WeeklyResult {
	wr := map[string]WeeklyResult{}
	for _, b := range bids {
		win, loss := 0, 0
		if s, ok := storyMap[b.Title]; ok {
			log.Info().Interface("story", s).Msg("won bid")
			win = b.Bid
		} else {
			log.Info().Interface("story", s).Msg("lost bid")
			loss = b.Bid
		}
		if res, ok := wr[b.User]; !ok {
			wr[b.User] = WeeklyResult{
				User: b.User,
				Won:  win,
				Lost: loss,
			}
		} else {
			res.Won = win
			res.Lost = loss
			wr[b.User] = res
		}
	}
	return wr
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
func (w *Webshit) GetWeekly() ([]Story, *time.Time, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("http://n-gate.com/hackernews/index.rss")
	if err != nil {
		return nil, nil, err
	}
	if len(feed.Items) <= 0 {
		return nil, nil, fmt.Errorf("no webshit weekly found")
	}

	published := feed.PublishedParsed

	buf := bytes.NewBufferString(feed.Items[0].Description)
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		return nil, nil, err
	}

	var items []Story
	doc.Find(".storylink").Each(func(i int, s *goquery.Selection) {
		story := Story{
			Title: s.Find("a").Text(),
			URL:   s.Find("a").AttrOr("src", ""),
		}
		items = append(items, story)
	})

	return items, published, nil
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

	tx := w.db.MustBegin()
	_, err = tx.Exec(`insert into webshit_bids (user,title,url,bid,created) values (?,?,?,?,?)`,
		user, story.Title, story.URL, amount, time.Now().Unix())
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec(`update webshit_balances set balance=? where user=?`,
		bal-amount, user)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

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

func (w *Webshit) updateScores(results map[string]WeeklyResult) error {
	tx := w.db.MustBegin()
	for _, res := range results {
		if _, err := tx.Exec(`update webshit_balances set score=score+? where user=?`,
			res.Won-res.Lost, res.User); err != nil {
			tx.Rollback()
			return err
		}
	}
	err := tx.Commit()
	return err
}
