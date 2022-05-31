package fact

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"time"
)

// Factoid stores info about our factoid for lookup and later interaction
type Factoid struct {
	ID       sql.NullInt64
	Fact     string
	Tidbit   string
	Verb     string
	Owner    string
	Created  time.Time
	Accessed time.Time
	Count    int
}

type alias struct {
	Fact string
	Next string
}

func (a *alias) resolve(db *sqlx.DB) (*Factoid, error) {
	// perform db query to fill the To field
	q := `select fact, next from factoid_alias where fact=?`
	var next alias
	err := db.Get(&next, q, a.Next)
	if err != nil {
		// we hit the end of the chain, get a factoid named Next
		fact, err := GetSingleFact(db, a.Next)
		if err != nil {
			err := fmt.Errorf("Error resolvig alias %v: %v", a, err)
			return nil, err
		}
		return fact, nil
	}
	return next.resolve(db)
}

func findAlias(db *sqlx.DB, fact string) (bool, *Factoid) {
	q := `select * from factoid_alias where fact=?`
	var a alias
	err := db.Get(&a, q, fact)
	if err != nil {
		return false, nil
	}
	f, err := a.resolve(db)
	return err == nil, f
}

func (a *alias) save(db *sqlx.DB) error {
	q := `select * from factoid_alias where fact=?`
	var offender alias
	err := db.Get(&offender, q, a.Next)
	if err == nil {
		return fmt.Errorf("DANGER: an opposite alias already exists")
	}
	_, err = a.resolve(db)
	if err != nil {
		return fmt.Errorf("there is no fact at that destination")
	}
	q = `insert or replace into factoid_alias (fact, next) values (?, ?)`
	_, err = db.Exec(q, a.Fact, a.Next)
	if err != nil {
		return err
	}
	return nil
}

func aliasFromStrings(from, to string) *alias {
	return &alias{from, to}
}

func (f *Factoid) Save(db *sqlx.DB) error {
	var err error
	if f.ID.Valid {
		// update
		_, err = db.Exec(`update factoid set
			fact=?,
			tidbit=?,
			verb=?,
			owner=?,
			accessed=?,
			count=?
		where id=?`,
			f.Fact,
			f.Tidbit,
			f.Verb,
			f.Owner,
			f.Accessed.Unix(),
			f.Count,
			f.ID.Int64)
	} else {
		f.Created = time.Now()
		f.Accessed = time.Now()
		// insert
		res, err := db.Exec(`insert into factoid (
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		) values (?, ?, ?, ?, ?, ?, ?);`,
			f.Fact,
			f.Tidbit,
			f.Verb,
			f.Owner,
			f.Created.Unix(),
			f.Accessed.Unix(),
			f.Count,
		)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		// hackhackhack?
		f.ID.Int64 = id
		f.ID.Valid = true
	}
	return err
}

func (f *Factoid) delete(db *sqlx.DB) error {
	var err error
	if f.ID.Valid {
		_, err = db.Exec(`delete from factoid where id=?`, f.ID)
	}
	f.ID.Valid = false
	return err
}

func getFacts(db *sqlx.DB, fact string, tidbit string) ([]*Factoid, error) {
	var fs []*Factoid
	query := `select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		where fact like ?
		and tidbit like ?;`
	rows, err := db.Query(query,
		"%"+fact+"%", "%"+tidbit+"%")
	if err != nil {
		log.Error().Err(err).Msg("Error regexping for facts")
		return nil, err
	}
	for rows.Next() {
		var f Factoid
		var tmpCreated int64
		var tmpAccessed int64
		err := rows.Scan(
			&f.ID,
			&f.Fact,
			&f.Tidbit,
			&f.Verb,
			&f.Owner,
			&tmpCreated,
			&tmpAccessed,
			&f.Count,
		)
		if err != nil {
			return nil, err
		}
		f.Created = time.Unix(tmpCreated, 0)
		f.Accessed = time.Unix(tmpAccessed, 0)
		fs = append(fs, &f)
	}
	return fs, err
}

func GetSingle(db *sqlx.DB) (*Factoid, error) {
	var f Factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := db.QueryRow(`select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		order by random() limit 1;`).Scan(
		&f.ID,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}

func GetSingleFact(db *sqlx.DB, fact string) (*Factoid, error) {
	var f Factoid
	var tmpCreated int64
	var tmpAccessed int64
	err := db.QueryRow(`select
			id,
			fact,
			tidbit,
			verb,
			owner,
			created,
			accessed,
			count
		from factoid
		where fact like ?
		order by random() limit 1;`,
		fact).Scan(
		&f.ID,
		&f.Fact,
		&f.Tidbit,
		&f.Verb,
		&f.Owner,
		&tmpCreated,
		&tmpAccessed,
		&f.Count,
	)
	f.Created = time.Unix(tmpCreated, 0)
	f.Accessed = time.Unix(tmpAccessed, 0)
	return &f, err
}
