package webshit

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	hacknews "github.com/PaulRosset/go-hacknews"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
)

type Config struct {
	HNFeed          string
	HNLimit         int
	BalanceReferesh int
}

var DefaultConfig = Config{
	HNFeed:          "topstories",
	HNLimit:         10,
	BalanceReferesh: 100,
}

type Webshit struct {
	db     *sqlx.DB
	config Config
}

type Story struct {
	Title string
	URL   string
}

type Stories []Story

func (s Stories) Titles() string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v.Title
	}
	return out
}

type Bid struct {
	ID     int
	User   string
	Title  string
	URL    string
	Bid    int
	Placed int64
}

func (b Bid) PlacedParsed() time.Time {
	return time.Unix(b.Placed, 0)
}

type Balance struct {
	User    string
	Balance int
	Score   int
}

type WeeklyResult struct {
	User            string
	Won             int
	WinningArticles Stories
	LosingArticles  Stories
	Score           int
}

func New(db *sqlx.DB) *Webshit {
	return NewConfig(db, DefaultConfig)
}

func NewConfig(db *sqlx.DB, cfg Config) *Webshit {
	w := &Webshit{db: db, config: cfg}
	w.setup()
	return w
}

// setup will create any necessary SQL tables and populate them with minimal data
func (w *Webshit) setup() {
	w.db.MustExec(`create table if not exists webshit_bids (
		id integer primary key autoincrement,
		user string,
		title string,
		url string,
		bid integer,
		placed integer
	)`)
	w.db.MustExec(`create table if not exists webshit_balances (
		user string primary key,
		balance int,
		score int
	)`)
}

func (w *Webshit) Check() ([]WeeklyResult, error) {
	stories, published, err := w.GetWeekly()
	if err != nil {
		return nil, err
	}

	var bids []Bid
	if err = w.db.Select(&bids, `select user,title,url,bid from webshit_bids where placed < ?`,
		published.Unix()); err != nil {
		return nil, err
	}

	// Assuming no bids earlier than the weekly means there hasn't been a new weekly
	if len(bids) == 0 {
		return nil, fmt.Errorf("there are no bids against the current ngate post")
	}

	storyMap := map[string]Story{}
	for _, s := range stories {
		u, err := url.Parse(s.URL)
		if err != nil {
			log.Error().Err(err).Msg("couldn't parse URL")
			continue
		}
		id := u.Query().Get("id")
		storyMap[id] = s
	}

	wr := w.checkBids(bids, storyMap)

	// Update all balance scores in a tx
	if err := w.updateScores(wr); err != nil {
		return nil, err
	}

	// Delete all those bids
	if _, err = w.db.Exec(`delete from webshit_bids where placed < ?`,
		published.Unix()); err != nil {
		return nil, err
	}

	// Set all balances to 100
	if _, err = w.db.Exec(`update webshit_balances set balance=?`,
		w.config.BalanceReferesh); err != nil {
		return nil, err
	}

	return wr, nil
}

func (w *Webshit) checkBids(bids []Bid, storyMap map[string]Story) []WeeklyResult {

	var wins []Bid
	total, totalWinning := 0.0, 0.0
	wr := map[string]WeeklyResult{}

	for _, b := range bids {
		score := w.GetScore(b.User)
		if _, ok := wr[b.User]; !ok {
			wr[b.User] = WeeklyResult{
				User:  b.User,
				Score: score,
			}
		}
		rec := wr[b.User]

		u, err := url.Parse(b.URL)
		if err != nil {
			log.Error().Err(err).Msg("couldn't parse URL")
			continue
		}
		id := u.Query().Get("id")

		if s, ok := storyMap[id]; ok {
			wins = append(wins, b)
			rec.WinningArticles = append(rec.WinningArticles, s)
			totalWinning += float64(b.Bid)
		} else {
			rec.LosingArticles = append(rec.LosingArticles, Story{b.Title, b.URL})
		}
		total += float64(b.Bid)
		wr[b.User] = rec
	}

	for _, b := range wins {
		payout := float64(b.Bid) / totalWinning * total
		rec := wr[b.User]
		rec.Won += int(payout)
		rec.Score += int(payout)
		wr[b.User] = rec
	}

	return wrMapToSlice(wr)
}

// GetHeadlines will return the current possible news headlines for bidding
func (w *Webshit) GetHeadlines() ([]Story, error) {
	news := hacknews.Initializer{Story: w.config.HNFeed, NbPosts: w.config.HNLimit}
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

	published := feed.Items[0].PublishedParsed

	buf := bytes.NewBufferString(feed.Items[0].Description)
	doc, err := goquery.NewDocumentFromReader(buf)
	if err != nil {
		return nil, nil, err
	}

	var items []Story
	doc.Find(".storylink").Each(func(i int, s *goquery.Selection) {
		story := Story{
			Title: s.Find("a").Text(),
			URL:   s.SiblingsFiltered(".small").First().Find("a").AttrOr("href", ""),
		}
		items = append(items, story)
		log.Debug().
			Str("URL", story.URL).
			Str("Title", story.Title).
			Msg("Parsed webshit story")
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

func (w *Webshit) GetScore(user string) int {
	q := `select score from webshit_balances where user=?`
	var score int
	err := w.db.Get(&score, q, user)
	if err != nil {
		return 0
	}
	return score
}

func (w *Webshit) GetAllBids() ([]Bid, error) {
	var bids []Bid
	err := w.db.Select(&bids, `select * from webshit_bids`)
	if err != nil {
		return nil, err
	}
	return bids, nil
}

func (w *Webshit) GetAllBalances() ([]Balance, error) {
	var balances []Balance
	err := w.db.Select(&balances, `select * from webshit_balances`)
	if err != nil {
		return nil, err
	}
	return balances, nil
}

// Bid allows a user to place a bid on a particular story
func (w *Webshit) Bid(user string, amount int, URL string) (Bid, error) {
	bal := w.GetBalance(user)
	if amount < 0 {
		return Bid{}, fmt.Errorf("cannot bid less than 0")
	}
	if bal < amount {
		return Bid{}, fmt.Errorf("cannot bid more than balance, %d", bal)
	}
	story, err := w.getStoryByURL(URL)
	if err != nil {
		return Bid{}, err
	}

	ts := time.Now().Unix()

	tx := w.db.MustBegin()
	_, err = tx.Exec(`insert into webshit_bids (user,title,url,bid,placed) values (?,?,?,?,?)`,
		user, story.Title, story.URL, amount, ts)
	if err != nil {
		tx.Rollback()
		return Bid{}, err
	}
	q := `insert into webshit_balances (user,balance,score) values (?,?,0)
		on conflict(user) do update  set balance=?`
	_, err = tx.Exec(q, user, bal-amount, bal-amount)
	if err != nil {
		tx.Rollback()
		return Bid{}, err
	}
	tx.Commit()

	return Bid{
		User:   user,
		Title:  story.Title,
		URL:    story.URL,
		Placed: ts,
	}, err
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

func (w *Webshit) updateScores(results []WeeklyResult) error {
	tx := w.db.MustBegin()
	for _, res := range results {
		if _, err := tx.Exec(`update webshit_balances set score=? where user=?`,
			res.Score, res.User); err != nil {
			tx.Rollback()
			return err
		}
	}
	err := tx.Commit()
	return err
}

func wrMapToSlice(wr map[string]WeeklyResult) []WeeklyResult {
	var out = []WeeklyResult{}
	for _, r := range wr {
		out = append(out, r)
	}
	return out
}
