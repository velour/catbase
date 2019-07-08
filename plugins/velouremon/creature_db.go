package velouremon

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func (vp *VelouremonPlugin) loadCreatures() error {
	rows, err := vp.db.Queryx("select * from velouremon_creatures;")
	if err != nil {
		log.Error().Err(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		creature := &Creature{
			Health:     255,
			Experience: 0,
		}
		err := rows.StructScan(creature)
		log.Print(err)
		if err != nil {
			log.Error().Err(err)
			return err
		}

		vp.creatures = append(vp.creatures, creature)
	}
	return nil
}

func (vp *VelouremonPlugin) loadCreaturesForPlayer(player *Player) ([]*Creature, error) {
	creatureRefs, err := vp.loadCreatureRefsForPlayer(player)
	if err != nil {
		return nil, err
	}
	creatures, err := vp.loadCreaturesFromRefs(creatureRefs)
	if err != nil {
		return nil, err
	}
	return creatures, nil
}

func (vp *VelouremonPlugin) loadCreatureRefsForPlayer(player *Player) ([]*CreatureRef, error) {
	rows, err := vp.db.Queryx(fmt.Sprintf("select * from velouremon_creature_ref where player = %d;", player.ID))
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	defer rows.Close()

	creatures := []*CreatureRef{}
	for rows.Next() {
		creature := &CreatureRef{}
		err := rows.StructScan(creature)

		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
		creatures = append(creatures, creature)
	}
	return creatures, nil
}

func (vp *VelouremonPlugin) loadCreaturesFromRefs(refs []*CreatureRef) ([]*Creature, error) {
	creatures := []*Creature{}
	for _, ref := range refs {
		creature, err := vp.loadCreaturesFromRef(ref)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
		creatures = append(creatures, creature)
	}
	return creatures, nil
}

func (vp *VelouremonPlugin) loadCreaturesFromRef(ref *CreatureRef) (*Creature, error) {
	creature := &Creature{}
	err := vp.db.QueryRowx(`select * from velouremon_creatures where id = ? LIMIT 1;`, ref.Creature).StructScan(creature)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	creature.Abilities, err = vp.loadAbilitiesForCreatureRef(ref)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	creature.Health = ref.Health
	creature.Experience = ref.Experience
	return creature, nil
}

func (vp *VelouremonPlugin) savePlayerCreatures(player *Player) error {
	for _, creature := range player.Creatures {
		err := vp.saveCreatureForPlayer(player, creature)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}
	return nil
}

func (vp *VelouremonPlugin) saveCreatureForPlayer(player *Player, creature *Creature) error {
	_, err := vp.db.Exec(`update velouremon_creature_ref set health = ?, experience = ? where id = ?;`, creature.ID, creature.Health, creature.Experience)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	return nil
}

func (vp *VelouremonPlugin) saveNewCreature(creature *Creature) error {
	_, err := vp.db.Exec(`insert into velouremon_creatures (name, defense, attack) values (?, ?, ?);`, creature.Name, creature.Defense, creature.Attack)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	return nil
}
