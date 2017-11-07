package capturetheflag

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

const (
	TIMESTAMP = "2006-01-02 15:04:05"
)

type CaptureTheFlagPlugin struct {
	Bot bot.Bot
	db  *sqlx.DB
}

type User struct {
	UserId   int64  `db:"id"`
	UserName string `db:"user"`
}

type Flag struct {
	FlagId   int64  `db:"id"`
	FlagName string `db:"flag"`
}

type Possession struct {
	PossessionId       int64 `db:"id"`
	UserId             int64 `db:"user"`
	FlagId             int64 `db:"flag"`
	Active             bool  `db:"active"`
	LastGrab           int64 `db:"lastgrab"`
	PossessionDuration int64 `db:"duration"`
}

// NewDicePlugin creates a new DicePlugin with the Plugin interface
func New(bot bot.Bot) *CaptureTheFlagPlugin {
	rand.Seed(time.Now().Unix())

	if _, err := bot.DB().Exec(`create table if not exists ctf_user (
			id integer primary key,
			user string
		);`); err != nil {
		log.Fatal(err)
	}

	if _, err := bot.DB().Exec(`create table if not exists ctf_flag (
			id integer primary key,
			flag string
		);`); err != nil {
		log.Fatal(err)
	}

	if _, err := bot.DB().Exec(`create table if not exists ctf_possession (
			id integer primary key,
			user integer,
			flag integer,
			active integer,
			lastgrab integer,
			duration integer
		);`); err != nil {
		log.Fatal(err)
	}

	return &CaptureTheFlagPlugin{
		Bot: bot,
		db:  bot.DB(),
	}
}

func (p *CaptureTheFlagPlugin) Message(message msg.Message) bool {
	lowercase := strings.ToLower(message.Body)
	tokens := strings.Fields(lowercase)
	numTokens := len(tokens)

	log.Println(lowercase)

	if numTokens >= 2 && tokens[0] == "capture" {
		log.Printf("%s trying to capture %s.", message.User.Name, strings.Join(tokens[1:], " "))
		response, err := p.tryCapture(message.User.Name, strings.Join(tokens[1:], " "))

		if err == nil {
			log.Print("Capture success.")
			log.Print(response)
			p.Bot.SendMessage(message.Channel, response)
			return true
		} else {
			log.Print("Capture failed.")
		}
	}
	return false
}

func (p *CaptureTheFlagPlugin) Help(channel string, parts []string) {
	p.Bot.SendMessage(channel, "There are flags. Capture them.")
}

func (p *CaptureTheFlagPlugin) Event(kind string, message msg.Message) bool {
	return false
}

func (p *CaptureTheFlagPlugin) BotMessage(message msg.Message) bool {
	return false
}

func (p *CaptureTheFlagPlugin) RegisterWeb() *string {
	return nil
}

func (p *CaptureTheFlagPlugin) createUser(username string) (*User, error) {
	res, err := p.db.Exec(`insert into ctf_user (user) values (?);`, username)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &User{
		UserId:   id,
		UserName: username,
	}, nil
}

func (p *CaptureTheFlagPlugin) getOrCreateUser(username string) (*User, bool, error) {
	var usr User
	err := p.db.QueryRowx(`select * from ctf_user where user = ? LIMIT 1;`, username).StructScan(&usr)
	if err != nil {
		if err == sql.ErrNoRows {
			usrPtr, err := p.createUser(username)
			return usrPtr, true, err
		}
		log.Print(err)
		return nil, false, err
	}
	return &usr, false, nil
}

func (p *CaptureTheFlagPlugin) getUserById(id int64) (*User, error) {
	var usr User
	err := p.db.QueryRowx(`select * from ctf_user where id = ? LIMIT 1;`, id).StructScan(&usr)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &usr, nil
}

func (p *CaptureTheFlagPlugin) createFlag(flagName string) (*Flag, error) {
	res, err := p.db.Exec(`insert into ctf_flag (flag) values (?);`, flagName)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &Flag{
		FlagId:   id,
		FlagName: flagName,
	}, nil
}

func (p *CaptureTheFlagPlugin) getOrCreateFlag(flagName string) (*Flag, bool, error) {
	var flag Flag
	err := p.db.QueryRowx(`select * from ctf_flag where flag = ? LIMIT 1;`, flagName).StructScan(&flag)
	if err != nil {
		if err == sql.ErrNoRows {
			flagPtr, err := p.createFlag(flagName)
			return flagPtr, true, err
		}
		log.Print(err)
		return nil, false, err
	}
	return &flag, false, nil
}

func (p *CaptureTheFlagPlugin) transferActivePossession(pos *Possession, usr *User, flag *Flag) (string, error) {
	extraTime := time.Now().Unix() - pos.LastGrab
	newTime := extraTime + pos.PossessionDuration

	_, err := p.db.Exec(`update ctf_possession set active = 0, duration = ? where id = ?;`, newTime, pos.PossessionId)
	if err != nil {
		log.Print(err)
		return "", err
	}

	last, err := p.getUserById(pos.UserId)
	if err != nil {
		log.Print(err)
		return "Flag possession dropped! (failed db query)", err
	}

	myPos, err := p.getPossession(usr, flag)
	if err != nil {
		log.Print(err)
		return "Flag possession dropped! (failed db query)", err
	}
	if myPos == nil {
		err = p.createFirstPossession(usr, flag)
		if err != nil {
			log.Print(err)
			return "Flag possession dropped! (failed db query)", err
		}
	} else {
		_, err = p.db.Exec(`update ctf_possession set active = 1, lastgrab = ? where id = ?;`, time.Now().Unix(), myPos.PossessionId)
		if err != nil {
			log.Print(err)
			return "Flag possession dropped! (failed db query)", err
		}
	}

	return fmt.Sprintf("%s now has '%s'. %s had it last for %d seconds (total %d seconds).", usr.UserName, flag.FlagName, last.UserName, extraTime, newTime), nil
}

func (p *CaptureTheFlagPlugin) createFirstPossession(usr *User, flag *Flag) error {
	_, err := p.db.Exec(`insert into ctf_possession (user, flag, active, lastgrab, duration) values (?, ?, ?, ?, ?);`, usr.UserId, flag.FlagId,
		true, time.Now().Unix(), 0)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (p *CaptureTheFlagPlugin) getActivePossession(flag *Flag) (*Possession, error) {
	var pos Possession
	err := p.db.QueryRowx(`select * from ctf_possession where active = 1 and flag = ? LIMIT 1;`, flag.FlagId).StructScan(&pos)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Print(err)
		return nil, err
	}
	return &pos, nil
}

func (p *CaptureTheFlagPlugin) getPossession(usr *User, flag *Flag) (*Possession, error) {
	var pos Possession
	err := p.db.QueryRowx(`select * from ctf_possession where user = ? and flag = ? LIMIT 1;`, usr.UserId, flag.FlagId).StructScan(&pos)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Print(err)
		return nil, err
	}
	return &pos, nil
}

func (p *CaptureTheFlagPlugin) tryCapture(username, what string) (string, error) {
	usr, _, err := p.getOrCreateUser(username)
	if err != nil {
		log.Print(err)
		return "", err
	}

	flag, newFlag, err := p.getOrCreateFlag(what)
	if err != nil {
		log.Print(err)
		return "", err
	}

	if !newFlag {
		pos, err := p.getActivePossession(flag)
		if err != nil {
			log.Print(err)
			return "", err
		}
		if pos == nil {
			err = p.createFirstPossession(usr, flag)
			if err != nil {
				log.Print(err)
				return "", err
			}
			return fmt.Sprintf("You now have '%s'", what), nil
		} else if pos.UserId == usr.UserId {
			return fmt.Sprintf("You already have '%s'", what), nil
		}
		response, err := p.transferActivePossession(pos, usr, flag)
		if err != nil {
			log.Print(err)
			return "", err
		}
		return response, nil
	} else {
		err = p.createFirstPossession(usr, flag)
		if err != nil {
			log.Print(err)
			return "", err
		}
		return fmt.Sprintf("You now have '%s'", what), nil
	}
}
