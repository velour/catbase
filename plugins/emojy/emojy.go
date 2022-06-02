package emojy

import (
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
	"time"
)

type EmojyPlugin struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB
}

func New(b bot.Bot) *EmojyPlugin {
	log.Debug().Msgf("emojy.New")
	p := &EmojyPlugin{
		b:  b,
		c:  b.Config(),
		db: b.DB(),
	}
	p.setupDB()
	p.register()
	p.registerWeb()
	return p
}

func (p *EmojyPlugin) setupDB() {
	p.db.MustExec(`create table if not exists emojyLog (
		id integer primary key autoincrement,
		emojy text,
		observed datetime
	)`)
}

func (p *EmojyPlugin) register() {
	ht := bot.HandlerTable{
		{
			Kind: bot.Reaction, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(request bot.Request) bool {
				log.Debug().Msgf("Emojy detected: %s", request.Msg.Body)
				p.recordReaction(request.Msg.Body)
				return false
			},
		},
	}
	p.b.RegisterTable(p, ht)
}

func (p *EmojyPlugin) recordReaction(emojy string) error {
	q := `insert into emojyLog (emojy, observed) values (?, ?)`
	_, err := p.db.Exec(q, emojy, time.Now())
	if err != nil {
		log.Error().Err(err).Msgf("recordReaction")
		return err
	}
	return nil
}

type EmojyEntry struct {
	Emojy    string    `db:"emojy"`
	Observed time.Time `db:"observed"`
}

type EmojyCount struct {
	Emojy string `json:"emojy"`
	Count int    `json:"count"`
}

func (p *EmojyPlugin) allCounts() ([]EmojyCount, error) {
	q := `select emojy, count(observed) as count from emojyLog group by emojy order by count desc`
	result := []EmojyCount{}
	err := p.db.Select(&result, q)
	if err != nil {
		return nil, err
	}
	return result, nil
}
