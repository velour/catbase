package rpgORdie

import (
	"fmt"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

const (
	DUDE    = ":lion_face:"
	BOULDER = ":full_moon:"
	HOLE    = ":new_moon:"
	EMPTY   = ":white_large_square:"

	OK      = iota
	INVALID = iota
	WIN     = iota
)

type RPGPlugin struct {
	Bot       bot.Bot
	listenFor map[string]*board
}

type board struct {
	state [][]string
	x, y  int
}

func NewRandomBoard() *board {
	boardSize := 5
	b := board{
		state: make([][]string, boardSize),
		x:     boardSize - 1,
		y:     boardSize - 1,
	}
	for i := 0; i < boardSize; i++ {
		b.state[i] = make([]string, boardSize)
		for j := 0; j < boardSize; j++ {
			b.state[i][j] = ":white_large_square:"
		}
	}

	b.state[boardSize-1][boardSize-1] = DUDE
	b.state[boardSize/2][boardSize/2] = BOULDER
	b.state[0][0] = HOLE

	return &b
}

func (b *board) toMessageString() string {
	lines := make([]string, len(b.state))
	for i := 0; i < len(b.state); i++ {
		lines[i] = strings.Join(b.state[i], "")
	}
	return strings.Join(lines, "\n")
}

func (b *board) checkAndMove(dx, dy int) int {
	newX := b.x + dx
	newY := b.y + dy

	if newX < 0 || newY < 0 || newX >= len(b.state) || newY >= len(b.state) {
		return INVALID
	}

	if b.state[newY][newX] == HOLE {
		return INVALID
	}

	win := false
	if b.state[newY][newX] == BOULDER {
		newBoulderX := newX + dx
		newBoulderY := newY + dy

		if newBoulderX < 0 || newBoulderY < 0 || newBoulderX >= len(b.state) || newBoulderY >= len(b.state) {
			return INVALID
		}

		if b.state[newBoulderY][newBoulderX] != HOLE {
			b.state[newBoulderY][newBoulderX] = BOULDER
		} else {
			win = true
		}
	}

	b.state[newY][newX] = DUDE
	b.state[b.y][b.x] = EMPTY
	b.x = newX
	b.y = newY

	if win {
		return WIN
	}
	return OK
}

func New(b bot.Bot) *RPGPlugin {
	return &RPGPlugin{
		Bot:       b,
		listenFor: map[string]*board{},
	}
}

func (p *RPGPlugin) Message(message msg.Message) bool {
	if strings.ToLower(message.Body) == "start rpg" {
		b := NewRandomBoard()
		ts, _ := p.Bot.Send(bot.Message, message.Channel, b.toMessageString())
		p.listenFor[ts] = b
		p.Bot.Send(bot.Reply, message.Channel, "Over here.", ts)
		return true
	}
	return false
}

func (p *RPGPlugin) LoadData() {

}

func (p *RPGPlugin) Help(channel string, parts []string) {
	p.Bot.Send(bot.Message, channel, "Go find a walkthrough or something.")
}

func (p *RPGPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *RPGPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *RPGPlugin) RegisterWeb() *string {
	return nil
}

func (p *RPGPlugin) ReplyMessage(message msg.Message, identifier string) bool {
	if strings.ToLower(message.User.Name) != strings.ToLower(p.Bot.Config().Get("Nick", "bot")) {
		if b, ok := p.listenFor[identifier]; ok {

			var res int

			if message.Body == "left" {
				res = b.checkAndMove(-1, 0)
			} else if message.Body == "right" {
				res = b.checkAndMove(1, 0)
			} else if message.Body == "up" {
				res = b.checkAndMove(0, -1)
			} else if message.Body == "down" {
				res = b.checkAndMove(0, 1)
			} else {
				return false
			}

			switch res {
			case OK:
				p.Bot.Send(bot.Edit, message.Channel, b.toMessageString(), identifier)
			case WIN:
				p.Bot.Send(bot.Edit, message.Channel, b.toMessageString(), identifier)
				p.Bot.Send(bot.Reply, message.Channel, "congratulations, you beat the easiest level imaginable.", identifier)
			case INVALID:
				p.Bot.Send(bot.Reply, message.Channel, fmt.Sprintf("you can't move %s", message.Body), identifier)
			}
			return true
		}
	}
	return false
}
