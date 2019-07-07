package velouremon

import (
	"github.com/rs/zerolog/log"
)

func (vp *VelouremonPlugin) checkAndBuildDBOrFail() {
	if _, err := vp.db.Exec(`create table if not exists velouremon_players (
			id integer primary key,
			chatid string,
			player string,
			health    integer,
			experience    integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_creature_ref (
		id integer primary key,
		player integer,
		creature integer,
		health integer,
		experience integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_creatures (
			id integer primary key,
			creature string,
			defense    integer,
			attack    integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_ability_ref (
		id integer primary key,
		creatureref integer,
		ability integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_abilities (
			id integer primary key,
			name string,
			damage int,
			heal int,
			shield int,
			weaken int,
			critical int
		);`); err != nil {
		log.Fatal().Err(err)
	}
}

func (vp *VelouremonPlugin) loadFromDB() {
	vp.loadPlayers()
	vp.loadCreatures()
	vp.loadAbilities()
}
