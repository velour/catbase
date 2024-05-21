package emojy

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/velour/catbase/connectors/discord"
	"image"
	"image/draw"
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

	emojyPath string
	baseURL   string
}

const maxLen = 32

func New(b bot.Bot) *EmojyPlugin {
	p := NewAPI(b)
	p.register()
	p.registerWeb()
	return p
}

// NewAPI creates a version used only for API purposes (no callbacks registered)
func NewAPI(b bot.Bot) *EmojyPlugin {
	emojyPath := b.Config().Get("emojy.path", "emojy")
	baseURL := b.Config().Get("emojy.baseURL", "/emojy/file")
	p := &EmojyPlugin{
		b:         b,
		c:         b.Config(),
		db:        b.DB(),
		emojyPath: emojyPath,
		baseURL:   baseURL,
	}
	p.setupDB()
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
			Kind: bot.Startup, IsCmd: false,
			Regex: regexp.MustCompile(`.*`),
			Handler: func(r bot.Request) bool {
				switch conn := r.Conn.(type) {
				case *discord.Discord:
					p.registerCmds(conn)
				}
				return false
			},
		},
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
				old := SanitizeName(r.Values["old"])
				new := SanitizeName(r.Values["new"])
				p.rmEmojyHandler(r, old)
				p.addEmojyHandler(r, new)
				return true
			},
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^addemojy (?P<name>.+)$`),
			Handler: func(r bot.Request) bool {
				name := SanitizeName(r.Values["name"])
				return p.addEmojyHandler(r, name)
			},
		},
		{
			Kind: bot.Message, IsCmd: true,
			Regex: regexp.MustCompile(`(?i)^rmemojy (?P<name>.+)$`),
			Handler: func(r bot.Request) bool {
				name := SanitizeName(r.Values["name"])
				return p.rmEmojyHandler(r, name)
			},
		},
	}
	p.b.RegisterTable(p, ht)
}

func (p *EmojyPlugin) RmEmojy(c bot.Connector, name string) error {
	onServerList := InvertEmojyList(p.b.GetEmojiList(false))
	// Call a non-existent emojy a successful remove
	if _, ok := onServerList[name]; !ok {
		return fmt.Errorf("could not find emojy %s", name)
	}
	if err := c.DeleteEmojy(name); err != nil {
		return err
	}
	return nil
}

func (p *EmojyPlugin) rmEmojyHandler(r bot.Request, name string) bool {
	err := p.RmEmojy(r.Conn, name)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Emoji does not exist")
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "removed emojy "+name)
	return true
}

func (p *EmojyPlugin) AddEmojy(c bot.Connector, name string) error {
	onServerList := InvertEmojyList(p.b.GetEmojiList(false))
	if _, ok := onServerList[name]; ok {
		return fmt.Errorf("emojy already exists")
	}
	if err := p.UploadEmojyInCache(c, name); err != nil {
		return err
	}
	return nil
}

func (p *EmojyPlugin) addEmojyHandler(r bot.Request, name string) bool {
	p.AddEmojy(r.Conn, name)
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

func InvertEmojyList(emojy map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range emojy {
		out[v] = k
	}
	return out
}

func (p *EmojyPlugin) allCountsFlat(threshold int) ([]EmojyCount, error) {
	emoji, err := p.allCounts(threshold)
	if err != nil {
		return nil, err
	}
	output := []EmojyCount{}
	for _, v := range emoji {
		output = append(output, v...)
	}
	return output, nil
}

func (p *EmojyPlugin) allCounts(threshold int) (map[string][]EmojyCount, error) {
	out := map[string][]EmojyCount{}
	onServerList := InvertEmojyList(p.b.GetEmojiList(true))
	q := `select emojy, count(observed) as count from emojyLog group by emojy having count(observed) > ? order by count desc`
	result := []EmojyCount{}
	err := p.db.Select(&result, q, threshold)
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

func AllFiles(emojyPath, baseURL string) (map[string]string, map[string]string, error) {
	files := map[string]string{}
	urls := map[string]string{}
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
	allFiles, allURLs, err := AllFiles(p.emojyPath, p.baseURL)
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

func (p *EmojyPlugin) UploadEmojyInCache(c bot.Connector, name string) error {
	ok, fname, _, err := p.isKnownEmojy(name)
	if !ok || err != nil {
		u := p.c.Get("baseurl", "")
		u = u + "/emojy"
		return fmt.Errorf("error getting emojy, the known emojy list can be found at: %s", u)
	}
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	return p.UploadEmojyImage(c, name, img)
}

func imageToRGBA(src image.Image) *image.RGBA {
	// No conversion needed if image is an *image.RGBA.
	if dst, ok := src.(*image.RGBA); ok {
		return dst
	}

	// Use the image/draw package to convert to *image.RGBA.
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

func (p *EmojyPlugin) UploadEmojyImage(c bot.Connector, name string, data image.Image) error {
	maxEmojySz := p.c.GetFloat64("emoji.maxsize", 128.0)
	ctx := gg.NewContextForRGBA(imageToRGBA(data))
	max := math.Max(float64(ctx.Width()), float64(ctx.Height()))
	ctx.Scale(maxEmojySz/max, maxEmojySz/max)
	w := bytes.NewBuffer([]byte{})
	err := ctx.EncodePNG(w)
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

func SanitizeName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	nameLen := len(name)
	if nameLen > maxLen {
		nameLen = maxLen
	}
	return name[:nameLen]
}
