package goals

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
	"github.com/velour/catbase/plugins/counter"
)

type GoalsPlugin struct {
	b   bot.Bot
	cfg *config.Config
	db  *sqlx.DB
}

func New(b bot.Bot) *GoalsPlugin {
	p := &GoalsPlugin{
		b:   b,
		cfg: b.Config(),
		db:  b.DB(),
	}
	p.mkDB()
	b.Register(p, bot.Message, p.message)
	b.Register(p, bot.Help, p.help)
	counter.RegisterUpdate(p.update)
	return p
}

func (p *GoalsPlugin) mkDB() {
	_, err := p.db.Exec(`create table if not exists goals (
		id integer primary key,
		kind string not null,
		who string not null,
		what string not null,
		amount integer,
		
		unique (who, what, kind)
	)`)
	if err != nil {
		log.Fatal().Msgf("could not create goals db: %s", err)
	}
}

var registerSelf = regexp.MustCompile(`(?i)^register (?P<type>competition|goal) (?P<what>[[:punct:][:alnum:]]+)\s?(?P<amount>[[:digit:]]+)?`)
var registerOther = regexp.MustCompile(`(?i)^register (?P<type>competition|goal) for (?P<who>[[:punct:][:alnum:]]+) (?P<what>[[:punct:][:alnum:]]+)\s?(?P<amount>[[:digit:]]+)?`)
var deRegisterSelf = regexp.MustCompile(`(?i)^deregister (?P<type>competition|goal) (?P<what>[[:punct:][:alnum:]]+)`)
var deRegisterOther = regexp.MustCompile(`(?i)^deregister (?P<type>competition|goal) for (?P<who>[[:punct:][:alnum:]]+) (?P<what>.*)`)
var checkSelf = regexp.MustCompile(`(?i)^check (?P<type>competition|goal) (?P<what>[[:punct:][:alnum:]]+)`)
var checkOther = regexp.MustCompile(`(?i)^check (?P<type>competition|goal) for (?P<who>[[:punct:][:alnum:]]+) (?P<what>[[:punct:][:alnum:]]+)`)

func (p *GoalsPlugin) message(conn bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	body := strings.TrimSpace(message.Body)
	who := message.User.Name
	ch := message.Channel

	if registerOther.MatchString(body) {
		c := parseCmd(registerOther, body)
		amount, _ := strconv.Atoi(c["amount"])
		p.register(conn, ch, c["type"], c["what"], c["who"], amount)
	} else if registerSelf.MatchString(body) {
		c := parseCmd(registerSelf, body)
		amount, _ := strconv.Atoi(c["amount"])
		p.register(conn, ch, c["type"], c["what"], who, amount)
	} else if deRegisterOther.MatchString(body) {
		c := parseCmd(deRegisterOther, body)
		p.deregister(conn, ch, c["type"], c["what"], c["who"])
	} else if deRegisterSelf.MatchString(body) {
		c := parseCmd(deRegisterSelf, body)
		p.deregister(conn, ch, c["type"], c["what"], who)
	} else if checkOther.MatchString(body) {
		c := parseCmd(checkOther, body)
		p.check(conn, ch, c["type"], c["what"], c["who"])
	} else if checkSelf.MatchString(body) {
		c := parseCmd(checkSelf, body)
		p.check(conn, ch, c["type"], c["what"], who)
	} else {
		return false
	}
	return true
}

func (p *GoalsPlugin) register(c bot.Connector, ch, kind, what, who string, howMuch int) {
	if kind == "goal" && howMuch == 0 {
		p.b.Send(c, bot.Message, ch,
			fmt.Sprintf("%s, you need to have a goal amount if you want to have a goal for %s.", who, what))
		return
	}
	g := p.newGoal(kind, who, what, howMuch)
	err := g.Save()
	if err != nil {
		log.Error().Err(err).Msgf("could not create goal")
		p.b.Send(c, bot.Message, ch, "I couldn't create that goal for some reason.")
		return
	}
	p.b.Send(c, bot.Message, ch, fmt.Sprintf("%s created", kind))
}

func (p *GoalsPlugin) deregister(c bot.Connector, ch, kind, what, who string) {
	g, err := p.getGoalKind(kind, who, what)
	if err != nil {
		log.Error().Err(err).Msgf("could not find goal to delete")
		p.b.Send(c, bot.Message, ch, "I couldn't find that item to deregister.")
		return
	}
	err = g.Delete()
	if err != nil {
		log.Error().Err(err).Msgf("could not delete goal")
		p.b.Send(c, bot.Message, ch, "I couldn't deregister that.")
		return
	}
	p.b.Send(c, bot.Message, ch, fmt.Sprintf("%s %s deregistered", kind, what))
}

func (p *GoalsPlugin) check(c bot.Connector, ch, kind, what, who string) {
	if kind == "goal" {
		p.checkGoal(c, ch, what, who)
		return
	}
	p.checkCompetition(c, ch, what, who)
}

func (p *GoalsPlugin) checkCompetition(c bot.Connector, ch, what, who string) {
	items, err := counter.GetItem(p.db, what)
	if err != nil || len(items) == 0 {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I couldn't find any %s", what))
		return
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count && who == items[i].Nick {
			return true
		}
		if items[i].Count > items[j].Count {
			return true
		}
		return false
	})

	if items[0].Nick == who && len(items) > 1 && items[1].Count == items[0].Count {
		p.b.Send(c, bot.Message, ch,
			fmt.Sprintf("Congratulations! You're in the lead for %s with %d, but you're tied with %s",
				what, items[0].Count, items[1].Nick))
		return
	}

	if items[0].Nick == who {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("Congratulations! You're in the lead for %s with %d.",
			what, items[0].Count))
		return
	}

	count := 0
	for _, i := range items {
		if i.Nick == who {
			count = i.Count
		}
	}

	p.b.Send(c, bot.Message, ch, fmt.Sprintf("%s is in the lead for %s with %d. You have %d to catch up.",
		items[0].Nick, what, items[0].Count, items[0].Count-count))
}

func (p *GoalsPlugin) checkGoal(c bot.Connector, ch, what, who string) {
	g, err := p.getGoalKind("goal", who, what)
	if err != nil {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I couldn't find %s", what))
	}

	item, err := counter.GetUserItem(p.db, who, what)
	if err != nil {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I couldn't find any %s", what))
		return
	}

	perc := float64(item.Count) / float64(g.Amount) * 100.0
	remaining := p.remainingText(item, g)

	if perc >= 100 {
		p.deregister(c, ch, g.Kind, g.What, g.Who)
		m := fmt.Sprintf("You made it! You have %.2f%% of %s and now it's done.", perc, what)
		p.b.Send(c, bot.Message, ch, m)
	} else {
		m := fmt.Sprintf("You have %d out of %d for %s. You're %.2f%% of the way there! %s",
			item.Count, g.Amount, what, perc, remaining)
		p.b.Send(c, bot.Message, ch, m)
	}

	return
}

func (p *GoalsPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	msg := "Goals can set goals and competition for your counters."
	msg += fmt.Sprintf("\nRegister with `%s` for yourself", registerSelf)
	msg += fmt.Sprintf("\nRegister with `%s` for other people", registerOther)
	msg += fmt.Sprintf("\nDeregister with `%s` for yourself", deRegisterSelf)
	msg += fmt.Sprintf("\nDeregister with `%s` for other people", deRegisterOther)
	msg += fmt.Sprintf("\nCheck with `%s` for yourself", checkSelf)
	msg += fmt.Sprintf("\nCheck with `%s` for other people", checkOther)
	p.b.Send(c, bot.Message, ch, msg)
	return true
}

type cmd map[string]string

func parseCmd(r *regexp.Regexp, body string) cmd {
	out := cmd{}
	subs := r.FindStringSubmatch(body)
	if len(subs) == 0 {
		return out
	}
	for i, n := range r.SubexpNames() {
		out[n] = subs[i]
	}
	return out
}

type goal struct {
	ID     int64
	Kind   string
	Who    string
	What   string
	Amount int

	gp *GoalsPlugin
}

func (p *GoalsPlugin) newGoal(kind, who, what string, amount int) goal {
	return goal{
		ID:     -1,
		Kind:   kind,
		Who:    who,
		What:   what,
		Amount: amount,
		gp:     p,
	}
}

func (p *GoalsPlugin) getGoal(who, what string) ([]*goal, error) {
	gs := []*goal{}
	err := p.db.Select(&gs, `select * from goals where who = ? and what = ?`,
		who, what)
	if err != nil {
		return nil, err
	}
	for _, g := range gs {
		g.gp = p
	}
	return gs, nil
}

func (p *GoalsPlugin) getGoalKind(kind, who, what string) (*goal, error) {
	g := &goal{gp: p}
	err := p.db.Get(g, `select * from goals where kind = ? and who = ? and what = ?`,
		kind, who, what)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *goal) Save() error {
	res, err := g.gp.db.Exec(`insert or replace into goals (who, what, kind, amount) values (?, ?, ? ,?)`,
		g.Who, g.What, g.Kind, g.Amount)
	if err != nil {
		return err
	}
	dbID, err := res.LastInsertId()
	if err == nil && dbID != g.ID && g.ID == 0 {
		g.ID = dbID
	}
	return nil
}

func (g goal) Delete() error {
	if g.ID == -1 {
		return nil
	}
	_, err := g.gp.db.Exec(`delete from goals where id = ?`, g.ID)
	return err
}

func (p *GoalsPlugin) update(u counter.Update) {
	log.Debug().Msgf("entered update for %#v", u)
	gs, err := p.getGoal(u.Who, u.What)
	if err != nil {
		log.Error().Err(err).Msgf("could not get goal for %#v", u)
		return
	}
	chs := p.cfg.GetArray("channels", []string{})
	c := p.b.DefaultConnector()
	for _, g := range gs {
		for _, ch := range chs {
			if g.Kind == "goal" {
				p.checkGoal(c, ch, u.What, u.Who)
			} else {
				p.checkCompetition(c, ch, u.What, u.Who)
			}
		}
	}
}

var now = time.Now

func (p *GoalsPlugin) calculateRemaining(i counter.Item, g *goal) int {
	today := float64(now().YearDay())
	thisYear := time.Date(now().Year(), 0, 0, 0, 0, 0, 0, time.UTC)
	nextYear := time.Date(now().Year()+1, 0, 0, 0, 0, 0, 0, time.UTC)
	days := nextYear.Sub(thisYear).Hours() / 24.0 // hopefully either a leap year on not

	perc := today / days
	shouldHave := float64(g.Amount) * perc
	diff := int(shouldHave) - i.Count

	log.Printf("Today is the %f-th day with %f days in the year", today, days)

	return diff
}

func (p *GoalsPlugin) remainingText(i counter.Item, g *goal) string {
	remaining := p.calculateRemaining(i, g)
	txt := ""
	if remaining < 0 {
		txt = fmt.Sprintf("You're ahead by %d!", -1*remaining)
	} else if remaining > 0 {
		txt = fmt.Sprintf("You need %d to get back on track.", remaining)
	}
	return txt
}
