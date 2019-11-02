package newsbid

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/plugins/newsbid/webshit"
)

type NewsBid struct {
	bot bot.Bot
	db  *sqlx.DB
	ws  *webshit.Webshit
}

func New(b bot.Bot) *NewsBid {
	ws := webshit.New(b.DB())
	p := &NewsBid{
		bot: b,
		db:  b.DB(),
		ws:  ws,
	}
	p.bot.Register(p, bot.Message, p.message)
	return p
}

func (p *NewsBid) message(conn bot.Connector, k bot.Kind, message msg.Message, args ...interface{}) bool {
	body := strings.ToLower(message.Body)
	ch := message.Channel
	if message.Command && body == "balance" {
		bal := p.ws.GetBalance(message.User.Name)
		p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("%s, your current balance is %d.",
			message.User.Name, bal))
		return true
	}
	if message.Command && body == "bids" {
		bids, err := p.ws.GetAllBids()
		if err != nil {
			p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error getting bids: %s", err))
			return true
		}
		if len(bids) == 0 {
			p.bot.Send(conn, bot.Message, ch, "No bids to report.")
			return true
		}
		sort.Slice(bids, func(i, j int) bool { return bids[i].User < bids[j].User })
		out := "Bids:\n"
		for _, b := range bids {
			out += fmt.Sprintf("%s bid %d on <%s|%s> \n", b.User, b.Bid, b.URL, b.Title)
		}
		p.bot.Send(conn, bot.Message, ch, out)
		return true
	}
	if message.Command && body == "scores" {
		bals, err := p.ws.GetAllBalances()
		if err != nil {
			p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error getting bids: %s", err))
			return true
		}
		if len(bals) == 0 {
			p.bot.Send(conn, bot.Message, ch, "No balances to report.")
			return true
		}
		out := "NGate balances:\n"
		for _, b := range bals {
			out += fmt.Sprintf("%s has a total score of %d with %d left to bid this session\n", b.User, b.Score, b.Balance)
		}
		p.bot.Send(conn, bot.Message, ch, out)
		return true

	}
	if message.Command && strings.HasPrefix(body, "bid") {
		parts := strings.Fields(body)
		if len(parts) != 3 {
			p.bot.Send(conn, bot.Message, ch, "You must bid with an amount and a URL.")
			return true
		}
		amount, _ := strconv.Atoi(parts[1])
		url := parts[2]
		if bid, err := p.ws.Bid(message.User.Name, amount, url); err != nil {
			p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error placing bid: %s", err))
		} else {
			p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Your bid has been placed on %s", bid.Title))
		}
		return true
	}
	if message.Command && body == "check ngate" {
		p.check(conn, ch)
		return true
	}
	return false
}

func (p *NewsBid) check(conn bot.Connector, ch string) {
	wr, err := p.ws.Check()
	if err != nil {
		p.bot.Send(conn, bot.Message, ch, fmt.Sprintf("Error checking ngate: %s", err))
		return
	}

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
			msg += "\nWinning articles: " + res.WinningArticles.Titles()
		}
		if len(res.LosingArticles) > 0 {
			msg += "\nLosing articles: " + res.LosingArticles.Titles()
		}
		p.bot.Send(conn, bot.Message, ch, msg)
	}
}
