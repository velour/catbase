package emojy

import (
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"regexp"
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
	p.db.MustExec(`create table if not exists emojyCounter (
		emojy text primary key,
		count integer
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
	q := `insert into emojyCounter (emojy, count) values (?, 1)
			on conflict(emojy) do update set count=count+1 where emojy=?;`
	_, err := p.db.Exec(q, emojy, emojy)
	if err != nil {
		log.Error().Err(err).Msgf("recordReaction")
		return err
	}
	return nil
}

type EmojyEntry struct {
	Emojy string `db:"emojy"`
	Count int    `db:"count"`
}

func (p *EmojyPlugin) all() ([]EmojyEntry, error) {
	q := `select emojy, count from emojyCounter order by count desc`
	result := []EmojyEntry{}
	err := p.db.Select(&result, q)
	if err != nil {
		return nil, err
	}
	return result, nil
}
