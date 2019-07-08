package velouremon

import (
	"fmt"
)

type Ability struct {
	ID       int    `db:"id"`
	Name     string `db:"name"`
	Damage   int    `db:"damage"`
	Heal     int    `db:"heal"`
	Shield   int    `db:"shield"`
	Weaken   int    `db:"weaken"`
	Critical int    `db:"critical"`
}

type AbilityRef struct {
	ID       int `db:"id"`
	Creature int `db:"creatureref"`
	Ability  int `db:"ability"`
}

func (a *Ability) string() string {
	return fmt.Sprintf("%s : %d DM, %d HL, %d SH, %d WK, %d CR\n", a.Name, a.Damage, a.Heal, a.Shield, a.Weaken, a.Critical)
}

func (a *Ability) applyCreature(creature, targetCreature *Creature) string {
	return ""
}

func (a *Ability) applyPlayer(creature *Creature, targetPlayer *Player) string {
	return ""
}
