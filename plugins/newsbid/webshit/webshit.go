package webshit

import (
	"bytes"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/velour/catbase/plugins/newsbid/webshit/hn"

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

type Bid struct {
	ID             int
	User           string
	Title          string
	URL            string
	Bid            int
	PlacedScore    int
	ProcessedScore int
	Placed         int64
	Processed      int64
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
	WinningArticles hn.Items
	LosingArticles  hn.Items
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
		placed_score integer,
		processed_score integer,
		placed integer,
		processed integer
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
	if err = w.db.Select(&bids, `select user,title,url,bid from webshit_bids where placed < ? and processed=0`,
		published.Unix()); err != nil {
		return nil, err
	}

	// Assuming no bids earlier than the weekly means there hasn't been a new weekly
	if len(bids) == 0 {
		return nil, fmt.Errorf("there are no bids against the current ngate post")
	}

	storyMap := map[string]hn.Item{}
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
	if _, err = w.db.Exec(`update webshit_bids set processed=? where placed < ?`,
		time.Now().Unix(), published.Unix()); err != nil {
		return nil, err
	}

	// Set all balances to 100
	if _, err = w.db.Exec(`update webshit_balances set balance=?`,
		w.config.BalanceReferesh); err != nil {
		return nil, err
	}

	return wr, nil
}

func (w *Webshit) checkBids(bids []Bid, storyMap map[string]hn.Item) []WeeklyResult {

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
			rec.LosingArticles = append(rec.LosingArticles, hn.Item{Title: b.Title, URL: b.URL})
		}
		total += float64(b.Bid)
		wr[b.User] = rec
	}

	for _, b := range wins {
		u, _ := url.Parse(b.URL)
		id, _ := strconv.Atoi(u.Query().Get("id"))
		item, err := hn.GetItem(id)
		score := item.Score
		comments := item.Descendants
		ratio := 1.0
		if err != nil {
			ratio = float64(score) / math.Max(float64(comments), 1.0)
		}
		payout := float64(b.Bid) / totalWinning * total * ratio
		rec := wr[b.User]
		rec.Won += int(payout)
		rec.Score += int(payout)
		wr[b.User] = rec
	}

	return wrMapToSlice(wr)
}

// GetWeekly will return the headlines in the last webshit weekly report
func (w *Webshit) GetWeekly() (hn.Items, *time.Time, error) {
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

	var items hn.Items
	doc.Find(".storylink").Each(func(i int, s *goquery.Selection) {
		story := hn.Item{
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
	err := w.db.Select(&bids, `select * from webshit_bids where processed=0`)
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
	_, err = tx.Exec(`insert into webshit_bids (user,title,url,bid,placed,processed) values (?,?,?,?,?,0)`,
		user, story.Title, story.URL, amount, ts)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return Bid{}, err
		}
		return Bid{}, err
	}
	q := `insert into webshit_balances (user,balance,score) values (?,?,0)
		on conflict(user) do update  set balance=?`
	_, err = tx.Exec(q, user, bal-amount, bal-amount)
	if err != nil {
		tx.Rollback()
		return Bid{}, err
	}
	err = tx.Commit()

	return Bid{
		User:   user,
		Title:  story.Title,
		URL:    story.URL,
		Placed: ts,
	}, err
}

// getStoryByURL scrapes the URL for a title
func (w *Webshit) getStoryByURL(URL string) (hn.Item, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return hn.Item{}, err
	}
	if u.Host != "news.ycombinator.com" {
		return hn.Item{}, fmt.Errorf("expected HN link")
	}
	id, err := strconv.Atoi(u.Query().Get("id"))
	if id == 0 || err != nil {
		return hn.Item{}, fmt.Errorf("invalid item ID")
	}
	return hn.GetItem(id)
}

func (w *Webshit) updateScores(results []WeeklyResult) error {
	tx := w.db.MustBegin()
	for _, res := range results {
		if _, err := tx.Exec(`update webshit_balances set score=? where user=?`,
			res.Score, res.User); err != nil {
			if err := tx.Rollback(); err != nil {
				return err
			}
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
