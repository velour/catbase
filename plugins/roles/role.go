package roles

import (
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type Role struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Offering string `json:"offering"`

	*sqlx.DB
}

func NewRole(db *sqlx.DB, name, offering string) *Role {
	return &Role{
		Name:     name,
		Offering: offering,
		DB:       db,
	}
}

func (r *Role) Save() error {
	q := `insert or replace into roles (name, offering) values (?, ?)`
	res, err := r.Exec(q, r.Name, r.Offering)
	if checkErr(err) {
		return err
	}
	id, err := res.LastInsertId()
	if checkErr(err) {
		return err
	}
	r.ID = id
	return nil
}

func getRolesByOffering(db *sqlx.DB, offering string) ([]*Role, error) {
	roles := []*Role{}
	err := db.Select(&roles, `select * from roles where offering=?`, offering)
	if checkErr(err) {
		return nil, err
	}
	for _, r := range roles {
		r.DB = db
	}
	return roles, nil
}

func getRole(db *sqlx.DB, name, offering string) (*Role, error) {
	role := &Role{}
	err := db.Get(role, `select * from roles where role=? and offering=?`, role, offering)
	if checkErr(err) {
		return nil, err
	}
	role.DB = db
	return role, nil
}

func checkErr(err error) bool {
	if err != nil {
		log.Error().Err(err).Msg("")
		return true
	}
	return false
}
