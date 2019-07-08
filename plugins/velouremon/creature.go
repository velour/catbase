package velouremon

import (
	"math/rand"
	"math"
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

func (vp *VelouremonPlugin) buildOutCreature(c *Creature) *Creature {
	creature := &Creature{
		ID: c.ID,
		Name: c.Name,
		Health: 255,
		Experience: int(math.Max(1, rand.NormFloat64() * 100 + 250)),
		Defense: c.Defense,
		Attack: c.Attack,
		Abilities: make([]*Ability, rand.Intn(4)),
	}

	used := map[int]int{}
	for i := range creature.Abilities {
		index := rand.Intn(len(vp.abilities))
		for _, ok := used[index]; ok; _, ok = used[index] {
			index = rand.Intn(len(vp.abilities))
		}
		creature.Abilities[i] = vp.abilities[index]
	}
	return creature
}
