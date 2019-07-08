package velouremon

import (
	"fmt"
	"strings"

	"github.com/velour/catbase/bot/user"
)

type Player struct {
	ID         int64  `db:"id"`
	ChatID     string `db:"chatid"`
	Name       string `db:"name"`
	Health     int    `db:"health"`
	Experience int    `db:"experience"`
	Creatures  []*Creature
}

func (vp *VelouremonPlugin) getOrAddPlayer(u *user.User) (*Player, error) {
	var player *Player
	for _, p := range vp.players {
		if p.ChatID == u.ID {
			return player, nil
		}
	}
	return vp.addPlayer(u)
}

func (p *Player) string() string {
	message := fmt.Sprintf("%s: %d HP, %d XP\n", p.Name, p.Health, p.Experience)
	for _, creature := range p.Creatures {
		message += "\t" + strings.ReplaceAll(creature.string(), "\n", "\n\t")
		message = strings.TrimSuffix(message, "\t")
	}
	return message
}
