package rest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/rs/zerolog/log"

	"github.com/jmoiron/sqlx"
	"github.com/velour/catbase/bot"
)

type RestPlugin struct {
	b  bot.Bot
	db *sqlx.DB

	handlers bot.HandlerTable
}

type postProcessor func(interface{}) string

var postProcessors = map[string]postProcessor{
	"gpt2": func(input interface{}) string {
		values := input.(map[string]interface{})
		text := values["text"].(string)
		lastStop := strings.LastIndexAny(text, ".!?")
		if lastStop > 0 {
			text = text[:lastStop+1]
		}
		eot := strings.LastIndex(text, "<|endoftext|>")
		if eot > 0 {
			text = text[:eot]
		}
		return text
	},
}

func New(b bot.Bot) *RestPlugin {
	p := &RestPlugin{
		b:        b,
		db:       b.DB(),
		handlers: bot.HandlerTable{},
	}
	p.setupDB()
	p.register()
	return p
}

func (p *RestPlugin) setupDB() {
	tx := p.db.MustBegin()
	tx.MustExec(`
		create table if not exists wires (
		id integer primary key autoincrement,
		url text not null,
		parse_regex text not null,
		return_field text not null
	)`)
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

func (p *RestPlugin) register() {
	p.handlers = bot.HandlerTable{
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile("(?i)^list wires$"),
			HelpText: "Lists all REST functions",
			Handler:  p.listWires},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile(`(?i)^rm wire (?P<id>\d+)$`),
			HelpText: "Removes a wire by ID (use list to view)",
			Handler:  p.rmWire},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile("(?i)^testwire `(?P<parse>[^`]+)` to (?P<url>\\S+) `(?P<returnField>[^`]+)` => (?P<text>.*)$"),
			HelpText: "Tests a new REST function",
			Handler:  p.handleTestWire},
		bot.HandlerSpec{Kind: bot.Message, IsCmd: true,
			Regex:    regexp.MustCompile("(?i)^wire `(?P<parse>[^`]+)` to (?P<url>\\S+) `(?P<returnField>[^`]+)`$"),
			HelpText: "Registers a new REST function",
			Handler:  p.handleWire},
	}
	p.b.RegisterTable(p, p.handlers)
	wires, err := p.getWires()
	if err != nil {
		panic(err)
	}
	for _, w := range wires {
		p.b.RegisterRegex(p, bot.Message, w.ParseRegex.Regexp, p.mkHandler(w))
	}
}

type ScanableRegexp struct {
	*regexp.Regexp
}

func (s *ScanableRegexp) Scan(src interface{}) error {
	var source string
	switch src.(type) {
	case string:
		source = src.(string)
	default:
		return errors.New("incompatible type for ScanableRegexp")
	}
	r, err := regexp.Compile(source)
	if err != nil {
		return err
	}
	s.Regexp = r
	return nil
}

type ScanableURL struct {
	*url.URL
}

func (s *ScanableURL) Scan(src interface{}) error {
	var source string
	switch src.(type) {
	case string:
		source = src.(string)
	default:
		return errors.New("incompatible type for ScanableURL")
	}
	u, err := url.Parse(source)
	if err != nil {
		return err
	}
	s.URL = u
	return nil
}

type wire struct {
	// ID
	ID sql.NullInt64
	// The URL to make a request to
	URL ScanableURL
	// The regex which will trigger this REST action
	ParseRegex ScanableRegexp `db:"parse_regex"`
	// The JSON field that will contain the REST return value
	ReturnField string `db:"return_field"`
}

func (w wire) String() string {
	msg := "Wire:"
	msg += fmt.Sprintf("\nURL: %s", w.URL)
	msg += fmt.Sprintf("\nParsing to trigger: `%s`", w.ParseRegex)
	msg += fmt.Sprintf("\nReturn field: `%s`", w.ReturnField)
	return msg
}

func (p *RestPlugin) getWires() ([]wire, error) {
	wires := []wire{}
	err := p.db.Select(&wires, `select * from wires`)
	return wires, err
}

func (p *RestPlugin) deleteWire(id int64) error {
	_, err := p.db.Exec(`delete from wires where id=?`, id)
	return err
}

func (w *wire) Update(db *sqlx.DB) error {
	if !w.ID.Valid {
		return w.Save(db)
	}
	id, _ := w.ID.Value()
	_, err := db.Exec(`update wires set url=?, parse_regex=?, return_field=? where id=?`,
		w.URL.String(), w.ParseRegex.String(), w.ReturnField, id)
	return err
}

func (w *wire) Save(db *sqlx.DB) error {
	if w.ID.Valid {
		return w.Update(db)
	}
	res, err := db.Exec(`insert into wires (url, parse_regex, return_field) values (?, ?, ?)`,
		w.URL.String(), w.ParseRegex.String(), w.ReturnField)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	_ = w.ID.Scan(id)
	return nil
}

func (p *RestPlugin) listWires(r bot.Request) bool {
	var msg string
	wires, err := p.getWires()
	if err != nil {
		msg = err.Error()
		goto SEND
	}
	msg = "Current wires:"
	for _, w := range wires {
		id, _ := w.ID.Value()
		msg += fmt.Sprintf("\n\t%d: `%s` => %s", id, w.ParseRegex, w.URL)
	}
SEND:
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}

func (p *RestPlugin) rmWire(r bot.Request) bool {
	id, _ := strconv.ParseInt(r.Values["id"], 10, 64)
	err := p.deleteWire(id)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, "Could not delete wire: "+err.Error())
		return true
	}
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Deleted wire: %d", id))
	return true
}

func (p *RestPlugin) mkWire(r bot.Request) (wire, error) {
	var w wire
	var err error
	w.ParseRegex.Regexp, err = regexp.Compile(r.Values["parse"])
	if err != nil {
		return w, err
	}
	w.URL.URL, err = url.Parse(r.Values["url"])
	if err != nil {
		return w, err
	}
	w.ReturnField = r.Values["returnField"]
	return w, nil
}

func (p *RestPlugin) handleWire(r bot.Request) bool {
	var w wire
	var msg string
	var err error
	w, err = p.mkWire(r)
	err = w.Save(p.db)
	if err != nil {
		msg = err.Error()
		goto SEND
	}
	p.b.RegisterRegex(p, bot.Message, w.ParseRegex.Regexp, p.mkHandler(w))
	msg = fmt.Sprintf("Saved %s", w)
SEND:
	p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	return true
}

func (p *RestPlugin) handleTestWire(r bot.Request) bool {
	text := r.Values["text"]
	w, err := p.mkWire(r)
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, err)
		return true
	}
	h := p.mkHandler(w)
	r.Values = bot.ParseValues(w.ParseRegex.Regexp, text)
	return h(r)
}

func (p *RestPlugin) mkHandler(w wire) bot.ResponseHandler {
	return func(r bot.Request) bool {
		if r.Msg.User.Name == p.b.Config().GetString("nick", "") {
			return false
		}
		values := bot.RegexValues{}
		for _, s := range p.b.Config().GetAllSecrets() {
			values[s.Key] = s.Value
		}
		log.Debug().Interface("values", values).Msgf("secrets")
		for k := range r.Values {
			values[k] = url.QueryEscape(r.Values[k])
		}
		log.Debug().Interface("values", values).Msgf("r.Values")
		urlStr := w.URL.String()
		parse, err := template.New(urlStr).Parse(urlStr)
		if p.handleErr(err, r) {
			return true
		}
		buf := bytes.Buffer{}
		err = parse.Execute(&buf, values)
		if p.handleErr(err, r) {
			return true
		}
		newURL, err := url.Parse(buf.String())
		log.Debug().
			Interface("values", values).
			Str("URL", buf.String()).
			Msg("Querying URL with values")
		if p.handleErr(err, r) {
			return true
		}
		resp, err := http.Get(newURL.String())
		if p.handleErr(err, r) {
			return true
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Got a status %d: %s",
				resp.StatusCode, resp.Status))
		}
		body, err := ioutil.ReadAll(resp.Body)
		if p.handleErr(err, r) {
			return true
		}
		var returnValues interface{}
		json.Unmarshal(body, &returnValues)

		var msg string
		if pp, ok := postProcessors[w.ReturnField]; ok {
			msg = pp(returnValues)
		} else {
			query, err := gojq.Parse(w.ReturnField)
			if err != nil {
				msg := fmt.Sprintf("Wire handler did not find return value: %s => `%s`", w.URL, w.ReturnField)
				p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
				return true
			}

			iter := query.Run(returnValues) // or query.RunWithContext

			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					return p.handleErr(err, r)
				}
				msg += fmt.Sprintf("%s\n", v)
			}
		}

		msg = strings.TrimSpace(msg)
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
		return true
	}
}

func (p *RestPlugin) handleErr(err error, r bot.Request) bool {
	if err != nil {
		p.b.Send(r.Conn, bot.Message, r.Msg.Channel, fmt.Sprintf("Error: %s", err))
		return true
	}
	return false
}
