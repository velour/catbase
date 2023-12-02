package aoc

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.chrissexton.org/cws/getaoc"

	"github.com/velour/catbase/bot"
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
	b.RegisterRegexCmd(aoc, bot.Message, aocRegex, aoc.aocCmd)
	return aoc
}

var aocRegex = regexp.MustCompile(`(?i)^aoc\s?(?P<year>\S+)?$`)

func (p *AOC) aocCmd(r bot.Request) bool {
	year := time.Now().Year()
	if time.Now().Month() < 11 {
		year--
	}
	if r.Values["year"] != "" {
		year, _ = strconv.Atoi(r.Values["year"])
	}
	boardId := p.c.GetInt("aoc.board", 0)
	var err error

	session := p.c.Get("aoc.session", "")
	if session == "" {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Error getting leaderboard: must have a session ID.")
		return true
	}
	if boardId == 0 {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Error getting leaderboard: must have a board ID.")
		return true
	}
	board, err := getaoc.GetLeaderboard(session, year, boardId)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Error getting leaderboard: %s", err))
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
	goldID, silverID, bronzeID := -1, -1, -1
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

	link := r.Conn.URLFormat("leaderboard", fmt.Sprintf("https://adventofcode.com/%d/leaderboard/private/view/%d", year, boardId))
	msg := fmt.Sprintf("AoC %s:\n", link)
	for _, m := range members {
		if m.Stars == 0 {
			continue
		}
		trophy := ""
		switch m.ID {
		case goldID:
			trophy = r.Conn.Emojy(":gold-trophy:")
		case silverID:
			trophy = r.Conn.Emojy(":silver-trophy:")
		case bronzeID:
			trophy = r.Conn.Emojy(":bronze-trophy:")
		}
		msg += fmt.Sprintf("%s has %d :star: for a score of %d%s\n", m.Name, m.Stars, m.LocalScore, trophy)
	}
	msg = strings.TrimSpace(msg)

	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}
