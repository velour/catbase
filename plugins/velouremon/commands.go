package velouremon

import (
	"strconv"

	"github.com/velour/catbase/bot"
)

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func (vp *VelouremonPlugin) handleStatus(c bot.Connector, player *Player) bool {
	vp.bot.Send(c, bot.Message, vp.channel, player.string())
	return true
}

func (vp *VelouremonPlugin) handleAddCreature(c bot.Connector, tokens []string) bool {
	if len(tokens) == 3 {
		name := tokens[0]
		stats := make([]int, 2)
		fail := false
		for i := range stats {
			stat, err := strconv.Atoi(tokens[i+1])
			if err != nil {
				fail = true
				break
			}
			stats[i] = min(max(stat, 0), 255)
		}
		if !fail {
			err := vp.saveNewCreature(&Creature{
				Name: name,
				Defense: stats[0],
				Attack: stats[1],
			})
			if err == nil {
				vp.bot.Send(c, bot.Message, vp.channel, "Added " + name)
				return true
			}
		}
	}

	vp.bot.Send(c, bot.Message, vp.channel, "!add_creature [name] [defense 0-255] [attack 0-255]")
	return true
}

func (vp *VelouremonPlugin) handleAddAbility(c bot.Connector, tokens []string) bool {
	if len(tokens) == 6 {
		name := tokens[0]
		stats := make([]int, 5)
		fail := false
		for i := range stats {
			stat, err := strconv.Atoi(tokens[i+1])
			if err != nil {
				fail = true
				break
			}
			stats[i] = min(max(stat, 0), 255)
		}
		if !fail {
			err := vp.saveNewAbility(&Ability{
				Name: name,
				Damage: stats[0],
				Heal: stats[1],
				Shield: stats[2],
				Weaken: stats[3],
				Critical: stats[4],
			})
			if err == nil {
				vp.bot.Send(c, bot.Message, vp.channel, "Added " + name)
				return true
			}
		}
	}

	vp.bot.Send(c, bot.Message, vp.channel, "!add_ability [name] [damage 0-255] [heal 0-255] [shield 0-255] [weaken 0-255] [critical 0-255]")
	return true
}

func (vp *VelouremonPlugin) handleBattle(c bot.Connector, player *Player) bool {
	id, _ := vp.bot.Send(c, bot.Message, vp.channel, "Let's get ready to rumble!")
	vp.threads[id] = &Interaction {
		id: id,
		players: []*Player{player},
		creatures: []*Creature{},
	}
	vp.bot.Send(c, bot.Reply, vp.channel, "Let's get ready to rumble!", id)
	return true
}
