package velouremon

import (
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot/user"
)

func (vp *VelouremonPlugin) loadPlayers() error {
	rows, err := vp.db.Queryx("select * from velouremon_players;")
	if err != nil {
		log.Error().Err(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		player := &Player{}
		err := rows.StructScan(player)
		if err != nil {
			log.Error().Err(err)
			return err
		}
		vp.players = append(vp.players, player)
		player.Creatures, err = vp.loadCreaturesForPlayer(player)
		if err != nil {
			log.Error().Err(err)
			return err
		}
	}
	return nil
}

func (vp *VelouremonPlugin) addPlayer(p *user.User) (*Player, error) {
	player := &Player{
		ChatID:     p.ID,
		Name:       p.Name,
		Health:     255,
		Experience: 0,
		Creatures:  []*Creature{},
	}

	res, err := vp.db.Exec(`insert into velouremon_players (chatid, player, health, experience) values (?, ?, ?, ?);`,
		player.ChatID, player.Name, player.Health, player.Experience)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}
	player.ID = id
	vp.players = append(vp.players, player)
	return player, nil
}

func (vp *VelouremonPlugin) savePlayer(player *Player) error {
	_, err := vp.db.Exec(`update velouremon_players set health = ?, experience = ? where id = ?;`, player.ID, player.Health, player.Experience)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	err = vp.savePlayerCreatures(player)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	return nil
}
