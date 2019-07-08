package velouremon

import (
	"fmt"
	"strings"
	"math/rand"
	"time"

	"github.com/velour/catbase/bot"
)

type Interaction struct {
	id        string
	players   []*Player
	creatures []*Creature
	started     bool
}

func randomInteraction(c bot.Connector, vp *VelouremonPlugin) {
	for {
		<-vp.timer.C
		if vp.channel != "" {
			creature := vp.creatures[rand.Intn(len(vp.creatures))]
			message := fmt.Sprintf("A wild %s appeared.", creature.Name)
			id, _ := vp.bot.Send(c, bot.Message, vp.channel, message)

			vp.threads[id] = &Interaction {
				id: id,
				players: []*Player{},
				creatures: []*Creature{
					vp.buildOutCreature(creature),
				},
				started: false,
			}

			vp.bot.Send(c, bot.Reply, vp.channel, "A wild %s appeared.", id)
		}

		vp.timer.Reset(1 * time.Hour)
	}
}

func (i *Interaction) handleMessage(vp *VelouremonPlugin, c bot.Connector, player *Player, tokens []string) bool {
	if len(tokens) > 0 {
		command := strings.ToLower(tokens[0])
		if command == "join" {
			return i.handleJoin(vp, c, player)
		} else if command == "run" {
			return i.handleRun(vp, c, player)
		}
	}
	return false
}

func (i *Interaction) handleJoin(vp *VelouremonPlugin, c bot.Connector, player *Player) bool {
	for _, p := range i.players {
		if player == p {
			vp.bot.Send(c, bot.Reply, vp.channel, player.Name + " is already in the party.", i.id)
			return true
		}
	}
	i.players = append(i.players, player)
	vp.bot.Send(c, bot.Reply, vp.channel, player.Name + " has just joined the party.", i.id)
	return true
}

func (i *Interaction) handleRun(vp *VelouremonPlugin, c bot.Connector, player *Player) bool {
	for index, p := range i.players {
		if player == p {
			i.players = append(i.players[:index], i.players[index+1:]...)
			vp.bot.Send(c, bot.Reply, vp.channel, player.Name + " has just left the party.", i.id)
			return true
		}
	}
	vp.bot.Send(c, bot.Reply, vp.channel, player.Name + " is not currently in the party.", i.id)
	return true
}
