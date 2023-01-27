package newsbid

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/plugins/newsbid/webshit"
)

type NewsBid struct {
	bot bot.Bot
	db  *sqlx.DB
	ws  *webshit.Webshit
}

func New(b bot.Bot) *NewsBid {
	ws := webshit.NewConfig(b.DB(), webshit.Config{
		HNFeed:          b.Config().GetString("webshit.hnfeed", "topstories"),
		HNLimit:         b.Config().GetInt("webshit.hnlimit", 10),
		BalanceReferesh: b.Config().GetInt("webshit.balancerefresh", 100),
		HouseName:       b.Config().GetString("webshit.housename", "house"),
	})
	p := &NewsBid{
		bot: b,
		db:  b.DB(),
		ws:  ws,
	}

	p.bot.RegisterRegexCmd(p, bot.Message, balanceRegex, p.balanceCmd)
	p.bot.RegisterRegexCmd(p, bot.Message, bidsRegex, p.bidsCmd)
	p.bot.RegisterRegexCmd(p, bot.Message, scoresRegex, p.scoresCmd)
	p.bot.RegisterRegexCmd(p, bot.Message, bidRegex, p.bidCmd)
	p.bot.RegisterRegexCmd(p, bot.Message, checkRegex, p.checkCmd)

	return p
}

var balanceRegex = regexp.MustCompile(`(?i)^balance$`)
var bidsRegex = regexp.MustCompile(`(?i)^bids$`)
var scoresRegex = regexp.MustCompile(`(?i)^scores$`)
var bidRegex = regexp.MustCompile(`(?i)^bid (?P<amount>\S+) (?P<url>\S+)\s?(?P<comment>.+)?$`)
var checkRegex = regexp.MustCompile(`(?i)^check ngate$`)

func (p *NewsBid) balanceCmd(r bot.Request) bool {
	if !r.Msg.Command {
		return false
	}

	bal := p.ws.GetBalance(r.Msg.User.Name)
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("%s, your current balance is %d.",
		r.Msg.User.Name, bal))
	return true
}

func (p *NewsBid) bidsCmd(r bot.Request) bool {
	ch := r.Msg.Channel
	bids, err := p.ws.GetAllBids()
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, ch, fmt.Sprintf("Error getting bids: %s", err))
		return true
	}
	if len(bids) == 0 {
		p.bot.Send(r.Conn, bot.Message, ch, "No bids to report.")
		return true
	}
	sort.Slice(bids, func(i, j int) bool {
		if bids[i].User == bids[j].User {
			return bids[i].Bid > bids[j].Bid
		}
		return bids[i].User < bids[j].User
	})
	out := "Bids:\n"
	for _, b := range bids {
		hnURL := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", b.HNID)
		out += fmt.Sprintf("• %s bid %s on %s (%s)\n", b.User, b.BidStr, b.Title, hnURL)
	}
	p.bot.Send(r.Conn, bot.Message, ch, out)
	return true
}

func (p *NewsBid) scoresCmd(r bot.Request) bool {

	ch := r.Msg.Channel
	bals, err := p.ws.GetAllBalances()
	if err != nil {
		p.bot.Send(r.Conn, bot.Message, ch, fmt.Sprintf("Error getting bids: %s", err))
		return true
	}
	if len(bals) == 0 {
		p.bot.Send(r.Conn, bot.Message, ch, "No balances to report.")
		return true
	}
	out := "NGate balances:\n"
	sort.Sort(bals)
	for _, b := range bals {
		if b.Score > 0 {
			out += fmt.Sprintf("%s has a total score of %d with %d left to bid this session\n", b.User, b.Score, b.Balance)
		}
	}
	p.bot.Send(r.Conn, bot.Message, ch, out)
	return true
}

func (p *NewsBid) bidCmd(r bot.Request) bool {
	body := strings.ToLower(r.Msg.Body)
	ch := r.Msg.Channel
	conn := r.Conn

	parts := strings.Fields(body)
	if len(parts) != 3 {
		p.bot.Send(conn, bot.Message, ch, "You must bid with an amount and a URL.")
		return true
	}

	amount, _ := strconv.Atoi(parts[1])
	url := parts[2]
	log.Debug().Msgf("URL: %s", url)
	if id, err := strconv.Atoi(url); err == nil {
		url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", id)
		log.Debug().Msgf("New URL: %s", url)
	}
	log.Debug().Msgf("URL: %s", url)

	if bid, err := p.ws.Bid(r.Msg.User.Name, amount, parts[1], url); err != nil {
		p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error placing bid: %s", err))
	} else {
		p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Your bid has been placed on %s", bid.Title))
	}
	return true
}

func (p *NewsBid) checkCmd(r bot.Request) bool {
	ch := r.Msg.Channel
	conn := r.Conn
	last := p.bot.Config().GetInt64("newsbid.lastprocessed", 0)
	wr, pubTime, err := p.ws.Check(last)
	if err != nil {
		p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error checking ngate: %s", err))
		return true
	}
	p.bot.Config().Set("newsbid.lastprocessed", strconv.FormatInt(pubTime, 10))

	topWon := 0
	topSpread := 0

	for _, res := range wr {
		if res.Won > topWon {
			topWon = res.Won
		}
		if len(res.WinningArticles) > topSpread {
			topSpread = len(res.WinningArticles)
		}
	}

	for _, res := range wr {
		icon := ""
		if res.Won == topWon {
			icon += "🏆 "
		}
		if len(res.WinningArticles) == topSpread {
			icon += "⭐️ "
		}
		msg := fmt.Sprintf("%s%s won %d for a score of %d",
			icon, res.User, res.Won, res.Score)
		if len(res.WinningArticles) > 0 {
			msg += "\nWinning articles: \n" + res.WinningArticles.Titles()
		}
		if len(res.LosingArticles) > 0 {
			msg += "\nLosing articles: \n" + res.LosingArticles.Titles()
		}
		p.bot.Send(conn, bot.Message, ch, msg)
	}
	return true
}
