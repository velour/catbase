package velouremon

import (
	"github.com/rs/zerolog/log"
)

func (vp *VelouremonPlugin) checkAndBuildDBOrFail() {
	dbNeedsPopulating := false
	if rows, err := vp.db.Queryx(`select name from sqlite_master where type='table' and name='velouremon_players';`); err != nil {
		log.Fatal().Err(err)
	} else {
		dbNeedsPopulating = rows.Next()
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_players (
			id         integer primary key,
			chatid     string,
			player     string,
			health     integer,
			experience integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_creature_ref (
		id         integer primary key,
		player     integer,
		creature   integer,
		health     integer,
		experience integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_creatures (
			id      integer primary key,
			name    string,
			defense integer,
			attack  integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_ability_ref (
		id          integer primary key,
		creatureref integer,
		ability     integer
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if _, err := vp.db.Exec(`create table if not exists velouremon_abilities (
			id       integer primary key,
			name     string,
			damage   int,
			heal     int,
			shield   int,
			weaken   int,
			critical int
		);`); err != nil {
		log.Fatal().Err(err)
	}

	if dbNeedsPopulating {
		vp.populateDBWithBaseData()
	}
}

func (vp *VelouremonPlugin) loadFromDB() {
	vp.loadPlayers()
	vp.loadCreatures()
	vp.loadAbilities()
}
