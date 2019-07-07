package velouremon

import (
	"fmt"
)

type Creature struct {
	ID         int    `db:"id"`
	Name       string `db:"name"`
	Health     int
	Experience int
	Defense    int `db:"defense"`
	Attack     int `db:"attack"`
	Abilities  []*Ability
}

type CreatureRef struct {
	ID         int `db:"id"`
	Creature   int `db:"creature"`
	Health     int `db:"health"`
	Experience int `db:"experience"`
}

func (c *Creature) string() string {
	message := fmt.Sprintf("%s : %d HP, %d XP, %d DEF, %d ATT\n", c.Name, c.Health, c.Experience, c.Defense, c.Attack)
	for _, ability := range c.Abilities {
		message += "\t" + ability.string()
	}
	return message
}
