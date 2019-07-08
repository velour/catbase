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
	lastUpdate time.Time
}

func randomInteraction(c bot.Connector, vp *VelouremonPlugin) {
	for {
		<-vp.timer.C

		fifteenMinutesAgo := time.Now().Add(-15 * time.Minute)
		remove := []string{}
		for _, i := range vp.threads {
			if i.lastUpdate.Before(fifteenMinutesAgo) {
				vp.bot.Send(c, bot.Reply, vp.channel, "Cleaning up the bodies.", i.id)
				remove = append(remove, i.id)
			}
		}
		for _, id := range remove {
			delete(vp.threads, id)
		}


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
				lastUpdate: time.Now(),
			}

			vp.bot.Send(c, bot.Reply, vp.channel, message, id)
		}

		vp.timer.Reset(time.Duration(15 + rand.Intn(30)) * time.Minute)
	}
}

func (i *Interaction) handleMessage(vp *VelouremonPlugin, c bot.Connector, player *Player, tokens []string) bool {
	i.lastUpdate = time.Now()

	if len(tokens) > 0 {

		if len(tokens) == 1 {
			command := strings.ToLower(tokens[0])
			if command == "status" {
				vp.bot.Send(c, bot.Reply, vp.channel, player.string(), i.id)
			} else if command == "join" {
				return i.handleJoin(vp, c, player)
			} else if command == "run" {
				return i.handleRun(vp, c, player)
			} else if command == "flail" {
				return i.handleFlail(vp, c, player)
			}
		} else if len(tokens) == 3 {
			src := strings.ToLower(tokens[0])
			ability := strings.ToLower(tokens[1])
			target := strings.ToLower(tokens[2])
			return i.handleAbilityUseCreature(vp, c, player, src, ability, target)
		} else if len(tokens) == 4 {
			src := strings.ToLower(tokens[0])
			ability := strings.ToLower(tokens[1])
			targetPlayer := strings.ToLower(tokens[2])
			target := strings.ToLower(tokens[3])
			return i.handleAbilityUsePlayer(vp, c, player, src, ability, targetPlayer, target)
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

func (i *Interaction) handleFlail(vp *VelouremonPlugin, c bot.Connector, player *Player) bool {
	return true

}

func (i *Interaction) handleAbilityUseCreature(vp *VelouremonPlugin, c bot.Connector, player *Player, src, ab, target string) bool {
	var creature *Creature
	for _, c := range player.Creatures {
		if strings.ToLower(c.Name) == src {
			creature = c
			 break
		}
	}
	if creature == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature source: " + src, i.id)
		return true
	}

	var ability *Ability
	for _, a := range creature.Abilities {
		if strings.ToLower(a.Name) == ab {
			ability = a
			break
		}
	}
	if ability == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature ability: " + ab, i.id)
		return true
	}

	var targetCreature *Creature
	for _, c := range i.creatures {
		if strings.ToLower(c.Name) == target {
			targetCreature = c
			 break
		}
	}

	var targetPlayer *Player
	for _, p := range i.players {
		if strings.ToLower(p.Name) == target {
			targetPlayer = p
			 break
		}
	}

	if targetCreature == nil && targetPlayer == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature/player target: " + target, i.id)
		return true
	}

	if targetCreature != nil {
		vp.bot.Send(c, bot.Reply, vp.channel, fmt.Sprintf("%s's %s used %s on %s", player.Name, creature.Name, ability.Name, targetCreature.Name), i.id)
		status := ability.applyCreature(creature, targetCreature)
		vp.bot.Send(c, bot.Reply, vp.channel, status, i.id)
	} else {
		vp.bot.Send(c, bot.Reply, vp.channel, fmt.Sprintf("%s's %s used %s on %s", player.Name, creature.Name, ability.Name, targetPlayer.Name), i.id)
		status := ability.applyPlayer(creature, targetPlayer)
		vp.bot.Send(c, bot.Reply, vp.channel, status, i.id)
	}
	return true
}

func (i *Interaction) handleAbilityUsePlayer(vp *VelouremonPlugin, c bot.Connector, player *Player, src, ab, tPlayer, target string) bool {
	var creature *Creature
	for _, c := range player.Creatures {
		if strings.ToLower(c.Name) == src {
			creature = c
			 break
		}
	}
	if creature == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature source: " + src, i.id)
		return true
	}

	var ability *Ability
	for _, a := range creature.Abilities {
		if strings.ToLower(a.Name) == ab {
			ability = a
			break
		}
	}
	if ability == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature ability: " + ab, i.id)
		return true
	}

	var targetPlayer *Player
	for _, p := range i.players {
		if strings.ToLower(p.Name) == tPlayer {
			targetPlayer = p
			 break
		}
	}
	if targetPlayer == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized player target: " + tPlayer, i.id)
		return true
	}

	var targetCreature *Creature
	for _, c := range targetPlayer.Creatures {
		if strings.ToLower(c.Name) == target {
			targetCreature = c
			 break
		}
	}
	if targetCreature == nil {
		vp.bot.Send(c, bot.Reply, vp.channel, "Unrecognized creature target: " + target, i.id)
		return true
	}

	vp.bot.Send(c, bot.Reply, vp.channel, fmt.Sprintf("%s's %s used %s on %s's %s", player.Name, creature.Name, ability.Name, targetPlayer.Name, targetCreature.Name), i.id)
	status := ability.applyCreature(creature, targetCreature)
	vp.bot.Send(c, bot.Reply, vp.channel, status, i.id)

	return true
}
