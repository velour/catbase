package aoc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.chrissexton.org/cws/getaoc"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type AOC struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *AOC {
	aoc := &AOC{
		b: b,
		c: b.Config(),
	}
	b.Register(aoc, bot.Message, aoc.message)
	return aoc
}

func (p *AOC) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !message.Command {
		return false
	}
	cleaned := strings.TrimSpace(strings.ToLower(message.Body))
	fields := strings.Split(cleaned, " ")
	if strings.HasPrefix(cleaned, "aoc") {
		year := time.Now().Year()
		if len(fields) > 1 {
			year, _ = strconv.Atoi(fields[1])
		}
		boardId := p.c.GetInt("aoc.board", 0)
		var err error
		if len(fields) > 2 {
			boardId, err = strconv.Atoi(fields[2])
			if err != nil {
				p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Error getting leaderboard: %s", err))
				return true
			}
		}
		session := p.c.Get("aoc.session", "")
		if session == "" {
			p.b.Send(c, bot.Message, message.Channel, "Error getting leaderboard: must have a session ID.")
			return true
		}
		if boardId == 0 {
			p.b.Send(c, bot.Message, message.Channel, "Error getting leaderboard: must have a board ID.")
			return true
		}
		board, err := getaoc.GetLeaderboard(session, year, boardId)
		if err != nil {
			p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Error getting leaderboard: %s", err))
			return true
		}

		members := []getaoc.Member{}
		for _, m := range board.Members {
			members = append(members, m)
		}
		sort.Slice(members, func(i, j int) bool {
			return members[i].LocalScore > members[j].LocalScore
		})

		gold, silver, bronze := -1, -1, -1
		goldID, silverID, bronzeID := "", "", ""
		for _, m := range members {
			if m.LocalScore > gold {
				gold = m.LocalScore
				goldID = m.ID
			}
			if m.LocalScore < gold && m.LocalScore > silver {
				silver = m.LocalScore
				silverID = m.ID
			}
			if m.LocalScore < silver && m.LocalScore > bronze {
				bronze = m.LocalScore
				bronzeID = m.ID
			}
		}

		msg := fmt.Sprintf("AoC <https://adventofcode.com/%d/leaderboard/private/view/%d|Leaderboard>:\n", year, boardId)
		for _, m := range members {
			if m.Stars == 0 {
				continue
			}
			trophy := ""
			switch m.ID {
			case goldID:
				trophy = ":gold-trophy:"
			case silverID:
				trophy = ":silver-trophy:"
			case bronzeID:
				trophy = ":bronze-trophy:"
			}
			msg += fmt.Sprintf("%s has %d :star: for a score of %d%s\n", m.Name, m.Stars, m.LocalScore, trophy)
		}
		msg = strings.TrimSpace(msg)

		p.b.Send(c, bot.Message, message.Channel, msg)
		return true
	}
	return false
}
