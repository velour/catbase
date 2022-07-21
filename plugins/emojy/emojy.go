package emojy

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/gabriel-vasile/mimetype"

	"github.com/forPelevin/gomoji"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/config"
)

type EmojyPlugin struct {
	b  bot.Bot
	c  *config.Config
	db *sqlx.DB
}

const maxLen = 32

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
					match = strings.Trim(match, ":")
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
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^swapemojy (?P<old>.+) (?P<new>.+)$`),
			Handler: func(r bot.Request) bool {
				old := sanitizeName(r.Values["old"])
				new := sanitizeName(r.Values["new"])
				p.rmEmojy(r, old)
				p.addEmojy(r, new)
				return true
			},
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^addemojy (?P<name>.+)$`),
			Handler: func(r bot.Request) bool {
				name := sanitizeName(r.Values["name"])
				return p.addEmojy(r, name)
			},
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^rmemojy (?P<name>.+)$`),
			Handler: func(r bot.Request) bool {
				name := sanitizeName(r.Values["name"])
				return p.rmEmojy(r, name)
			},
		},
	}
	p.b.RegisterTable(p, ht)
}

func (p *EmojyPlugin) rmEmojy(r bot.Request, name string) bool {
	onServerList := invertEmojyList(p.b.GetEmojiList(false))
	if _, ok := onServerList[name]; !ok {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Emoji does not exist")
		return true
	}
	if err := r.Conn.DeleteEmojy(name); err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "error "+err.Error())
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "removed emojy "+name)
	return true
}

func (p *EmojyPlugin) addEmojy(r bot.Request, name string) bool {
	onServerList := invertEmojyList(p.b.GetEmojiList(false))
	if _, ok := onServerList[name]; ok {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Emoji already exists")
		return true
	}
	if err := p.uploadEmojy(r.Conn, name); err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("error adding emojy: %v", err))
		return true
	}
	list := r.Conn.GetEmojiList(true)
	for k, v := range list {
		if v == name {
			p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("added emojy: %s <:%s:%s>", name, name, k))
			break
		}
	}
	return true
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
	Emojy    string `json:"emojy"`
	URL      string `json:"url"`
	Count    int    `json:"count"`
	OnServer bool   `json:"onServer"`
}

func invertEmojyList(emojy map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range emojy {
		out[v] = k
	}
	return out
}

func (p *EmojyPlugin) allCounts() (map[string][]EmojyCount, error) {
	out := map[string][]EmojyCount{}
	onServerList := invertEmojyList(p.b.GetEmojiList(true))
	q := `select emojy, count(observed) as count from emojyLog group by emojy order by count desc`
	result := []EmojyCount{}
	err := p.db.Select(&result, q)
	if err != nil {
		return nil, err
	}
	for _, e := range result {
		_, e.OnServer = onServerList[e.Emojy]
		if isEmoji(e.Emojy) {
			out["emoji"] = append(out["emoji"], e)
		} else if ok, _, eURL, _ := p.isKnownEmojy(e.Emojy); ok {
			e.URL = eURL
			out["emojy"] = append(out["emojy"], e)
		} else {
			out["unknown"] = append(out["unknown"], e)
		}
	}
	return out, nil
}

func AllFiles(c *config.Config) (map[string]string, map[string]string, error) {
	files := map[string]string{}
	urls := map[string]string{}
	emojyPath := c.Get("emojy.path", "emojy")
	baseURL := c.Get("emojy.baseURL", "/emojy/file")
	entries, err := os.ReadDir(emojyPath)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			baseName := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			url := path.Join(baseURL, e.Name())
			files[baseName] = path.Join(emojyPath, e.Name())
			urls[baseName] = url
		}
	}
	return files, urls, nil
}

func (p *EmojyPlugin) isKnownEmojy(name string) (bool, string, string, error) {
	allFiles, allURLs, err := AllFiles(p.c)
	if err != nil {
		return false, "", "", err
	}
	if _, ok := allURLs[name]; !ok {
		return false, "", "", nil
	}
	return true, allFiles[name], allURLs[name], nil
}

func isEmoji(in string) bool {
	return gomoji.ContainsEmoji(in)
}

func (p *EmojyPlugin) uploadEmojy(c bot.Connector, name string) error {
	maxEmojySz := p.c.GetFloat64("emoji.maxsize", 128.0)
	ok, fname, _, err := p.isKnownEmojy(name)
	if !ok || err != nil {
		u := p.c.Get("baseurl", "")
		u = u + "/emojy"
		return fmt.Errorf("error getting emojy, the known emojy list can be found at: %s", u)
	}
	i, err := gg.LoadImage(fname)
	if err != nil {
		return err
	}
	ctx := gg.NewContextForImage(i)
	max := math.Max(float64(ctx.Width()), float64(ctx.Height()))
	ctx.Scale(maxEmojySz/max, maxEmojySz/max)
	w := bytes.NewBuffer([]byte{})
	err = ctx.EncodePNG(w)
	if err != nil {
		return err
	}
	mtype := mimetype.Detect(w.Bytes())
	base64Img := "data:" + mtype.String() + ";base64," + base64.StdEncoding.EncodeToString(w.Bytes())
	if err := c.UploadEmojy(name, base64Img); err != nil {
		return err
	}
	return nil
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	nameLen := len(name)
	if nameLen > maxLen {
		nameLen = maxLen
	}
	return name[:nameLen]
}
