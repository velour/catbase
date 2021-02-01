package achievements

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

// A plugin to track our misdeeds
type AchievementsPlugin struct {
	bot bot.Bot
	cfg *config.Config
	db  *sqlx.DB
}

func New(b bot.Bot) *AchievementsPlugin {
	ap := &AchievementsPlugin{
		bot: b,
		cfg: b.Config(),
		db:  b.DB(),
	}
	err := ap.mkDB()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create achievements tables")
	}

	b.RegisterRegexCmd(ap, bot.Message, grantRegex, ap.grantCmd)
	b.RegisterRegexCmd(ap, bot.Message, createRegex, ap.createCmd)
	b.RegisterRegexCmd(ap, bot.Message, greatRegex, ap.greatCmd)

	b.Register(ap, bot.Help, ap.help)
	return ap
}

func (p *AchievementsPlugin) mkDB() error {
	trophiesTable := `create table if not exists trophies (
		emojy string primary key,
		description string,
		creator string
	);`

	awardsTable := `create table if not exists awards (
		id integer primary key,
		emojy string references trophies(emojy) on delete restrict on update cascade,
		holder string,
		amount integer default 0,
		granted timestamp CURRENT_TIMESTAMP 
	);`
	tx, err := p.db.Beginx()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(trophiesTable); err != nil {
		return err
	}
	if _, err = tx.Exec(awardsTable); err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *AchievementsPlugin) GetAwards(nick string) []Award {
	var awards []Award
	q := `select * from awards inner join trophies on awards.emojy=trophies.emojy where holder=?`
	if err := p.db.Select(&awards, q, nick); err != nil {
		log.Error().Err(err).Msg("could not select awards")
	}
	return awards
}

var grantRegex = regexp.MustCompile(`(?i)grant (?P<emojy>(?::[[:word:][:punct:]]+:\s?)+) to :?(?P<nick>[[:word:]]+):?`)
var createRegex = regexp.MustCompile(`(?i)create trophy (?P<emojy>(?::[[:word:][:punct:]]+:\s?)+) (?P<description>.+)`)
var greatRegex = regexp.MustCompile(`(?i)how great (?:am i|is :?(?P<who>[[:word:]]+))[[:punct:]]*`)

func (p *AchievementsPlugin) grantCmd(r bot.Request) bool {
	nick := r.Msg.User.Name
	emojy := r.Values["emojy"]
	receiver := r.Values["nick"]
	trophy, err := p.FindTrophy(emojy)
	if err != nil {
		log.Error().Err(err).Msg("could not find trophy")
		msg := fmt.Sprintf("The %s award doesn't exist.", emojy)
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
		return true
	}
	if nick == trophy.Creator {
		a, err := p.Grant(receiver, emojy)
		if err != nil {
			log.Error().Err(err).Msg("could not award trophy")
		}
		msg := fmt.Sprintf("Congrats %s. You just got the %s award for %s.",
			receiver, emojy, a.Description)
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	} else {
		msg := fmt.Sprintf("Sorry, %s. %s owns that trophy.", nick, trophy.Creator)
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, msg)
	}
	return true
}

func (p *AchievementsPlugin) createCmd(r bot.Request) bool {
	nick := r.Msg.User.Name
	emojy := r.Values["emojy"]
	description := r.Values["description"]
	t, err := p.Create(emojy, description, nick)
	if err != nil {
		log.Error().Err(err).Msg("could not create trophy")
		if strings.Contains(err.Error(), "exists") {
			p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, err.Error())
			return true
		}
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, "I'm too humble to ever award that trophy")
		return true
	}
	resp := fmt.Sprintf("Okay %s. I have crafted a one-of-a-kind %s trophy to give for %s",
		nick, t.Emojy, t.Description)
	p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, resp)
	return true
}

func (p *AchievementsPlugin) greatCmd(r bot.Request) bool {
	nick := r.Msg.User.Name
	who := r.Values["who"]
	if who == "" {
		who = nick
	}
	awards := p.GetAwards(who)
	if len(awards) == 0 {
		m := fmt.Sprintf("%s has no achievements to their name. They really suck.", who)
		if who == nick {
			m = fmt.Sprintf("You have no achievements to your name. "+
				"You are a sad and terrible specimen of the human condition, %s.", who)
		}
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, m)
	} else {
		m := fmt.Sprintf("Wow, let's all clap for %s. Look at these awards:", who)
		for _, a := range awards {
			m += fmt.Sprintf("\n%s - %s", a.Emojy, a.Description)
		}
		p.bot.Send(r.Conn, bot.Message, r.Msg.Channel, m)
	}
	return true
}

func (p *AchievementsPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	me := p.bot.WhoAmI()
	msg := "The achievements plugins awards trophies."
	msg += fmt.Sprintf("\nYou can create a trophy with `%s, create trophy <emojy> <description>`", me)
	msg += fmt.Sprintf("\nYou can award a trophy with `%s, grant <emojy> to <nick>`", me)
	msg += "\nYou can see others awards with `How great is <nick>`"
	msg += "\nYou can see your awards with `How great am I?`"
	p.bot.Send(c, bot.Message, ch, msg)
	return true
}

// Award is used by other plugins to register a particular award for a user
func (p *AchievementsPlugin) Grant(nick, emojy string) (Award, error) {
	empty := Award{}
	q := `insert into awards (emojy,holder) values (?, ?)`
	tx, err := p.db.Beginx()
	if err != nil {
		return empty, err
	}
	if _, err := tx.Exec(q, emojy, nick); err != nil {
		tx.Rollback()
		return empty, err
	}
	if err := tx.Commit(); err != nil {
		return empty, err
	}
	return p.FindAward(emojy)
}

func (p *AchievementsPlugin) Create(emojy, description, creator string) (Trophy, error) {
	t, err := p.FindTrophy(emojy)
	if err == nil {
		return t, fmt.Errorf("the trophy %s already exists", emojy)
	}

	q := `insert into trophies (emojy,description,creator) values (?,?,?)`
	tx, err := p.db.Beginx()
	if err != nil {
		return Trophy{}, err
	}
	_, err = tx.Exec(q, emojy, description, creator)
	if err != nil {
		tx.Rollback()
		return Trophy{}, err
	}
	err = tx.Commit()
	return Trophy{
		Emojy:       emojy,
		Description: description,
		Creator:     creator,
	}, err
}

type Trophy struct {
	Emojy       string
	Description string
	Creator     string
}

type Award struct {
	Trophy
	ID      int64
	Holder  string
	Amount  int
	Granted *time.Time
}

func (a *Award) Save() error {
	return nil
}

func (p *AchievementsPlugin) FindTrophy(emojy string) (Trophy, error) {
	q := `select * from trophies where emojy=?`
	var t Trophy
	err := p.db.Get(&t, q, emojy)
	return t, err
}

func (p *AchievementsPlugin) FindAward(emojy string) (Award, error) {
	q := `select * from awards inner join trophies on awards.emojy=trophies.emojy where trophies.emojy=?`
	var a Award
	err := p.db.Get(&a, q, emojy)
	return a, err
}
