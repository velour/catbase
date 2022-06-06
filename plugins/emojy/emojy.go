package emojy

import (
	"github.com/forPelevin/gomoji"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
	"os"
	"path"
	"regexp"
	"strings"
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
			Kind: bot.Message, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(request bot.Request) bool {
				r := regexp.MustCompile(`:[a-zA-Z0-9_-]+:`)
				for _, match := range r.FindAllString(request.Msg.Body, -1) {
					log.Debug().Msgf("Emojy detected: %s", match)
					p.recordReaction(match)
				}
				return false
			},
		},
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
	URL   string `json:"url"`
	Count int    `json:"count"`
}

func (p *EmojyPlugin) allCounts() (map[string][]EmojyCount, error) {
	out := map[string][]EmojyCount{}
	q := `select emojy, count(observed) as count from emojyLog group by emojy order by count desc`
	result := []EmojyCount{}
	err := p.db.Select(&result, q)
	if err != nil {
		return nil, err
	}
	for _, e := range result {
		if isEmoji(e.Emojy) {

			out["emoji"] = append(out["emoji"], e)
		} else if ok, fname, _ := p.isKnownEmojy(e.Emojy); ok {
			e.URL = fname
			out["emojy"] = append(out["emojy"], e)
		} else {
			out["unknown"] = append(out["unknown"], e)
		}
	}
	return out, nil
}

func (p *EmojyPlugin) isKnownEmojy(name string) (bool, string, error) {
	emojyPath := p.c.Get("emojy.path", "emojy")
	baseURL := p.c.Get("emojy.baseURL", "/emojy/file")
	entries, err := os.ReadDir(emojyPath)
	if err != nil {
		return false, "", err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), name) {
			url := path.Join(baseURL, e.Name())
			return true, url, nil
		}
	}
	return false, "", nil
}

func isEmoji(in string) bool {
	return gomoji.ContainsEmoji(in)
}
