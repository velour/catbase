package velouremon

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

func (vp *VelouremonPlugin) loadAbilities() error {
	rows, err := vp.db.Queryx("select * from velouremon_abilities;")
	if err != nil {
		log.Error().Err(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		ability := &Ability{}
		err := rows.StructScan(ability)
		if err != nil {
			log.Error().Err(err)
			return err
		}
		vp.abilities = append(vp.abilities, ability)
	}
	return nil
}

func (vp *VelouremonPlugin) loadAbilitiesForCreatureRef(ref *CreatureRef) ([]*Ability, error) {
	abilityRefs, err := vp.loadAbilityRefsForCreature(ref)
	if err != nil {
		return nil, err
	}
	abilities, err := vp.loadAbilitiesFromRefs(abilityRefs)
	if err != nil {
		return nil, err
	}
	return abilities, nil
}

func (vp *VelouremonPlugin) loadAbilityRefsForCreature(ref *CreatureRef) ([]*AbilityRef, error) {
	rows, err := vp.db.Queryx(fmt.Sprintf("select * from velouremon_ability_ref where creatureref = %d;", ref.ID))
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	defer rows.Close()

	abilities := []*AbilityRef{}
	for rows.Next() {
		ability := &AbilityRef{}
		err := rows.StructScan(ability)

		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
		abilities = append(abilities, ability)
	}
	return abilities, nil
}

func (vp *VelouremonPlugin) loadAbilitiesFromRefs(refs []*AbilityRef) ([]*Ability, error) {
	abilities := []*Ability{}
	for _, ref := range refs {
		ability, err := vp.loadAbilityFromRef(ref)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}
		abilities = append(abilities, ability)
	}
	return abilities, nil
}

func (vp *VelouremonPlugin) loadAbilityFromRef(ref *AbilityRef) (*Ability, error) {
	ability := &Ability{}
	err := vp.db.QueryRowx(`select * from velouremon_abilities where id = ? LIMIT 1;`, ref.Ability).StructScan(ability)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	return ability, nil
}

func (vp *VelouremonPlugin) saveNewAbility(ability *Ability) error {
	_, err := vp.db.Exec(`insert into velouremon_abilities (name, damage, heal, shield, weaken, critical) values (?, ?, ?, ?, ?, ?);`, ability.Name, ability.Damage, ability.Heal, ability.Shield, ability.Weaken, ability.Critical)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	return nil
}
