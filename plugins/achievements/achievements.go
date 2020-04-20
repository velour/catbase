package achievements

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

// I feel dirty about this, but ;shrug
var instance *AchievementsPlugin

type Category int

const (
	Daily = iota
	Monthly
	Yearly
	Forever
)

// A plugin to track our misdeeds
type AchievementsPlugin struct {
	bot bot.Bot
	cfg *config.Config
	db  *sqlx.DB
}

func New(b bot.Bot) *AchievementsPlugin {
	if instance == nil {
		ap := &AchievementsPlugin{
			bot: b,
			cfg: b.Config(),
			db:  b.DB(),
		}
		instance = ap
		ap.mkDB()
		b.Register(ap, bot.Message, ap.message)
		b.Register(ap, bot.Help, ap.help)
	}
	return instance
}

func (p *AchievementsPlugin) mkDB() error {
	q := `
	create table if not exists achievements
	`
	tx, err := p.db.Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(q)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (p *AchievementsPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	return false
}

func (p *AchievementsPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	me := p.bot.WhoAmI()
	msg := fmt.Sprintf("%s helps those who help themselves.", me)
	p.bot.Send(c, bot.Message, ch, msg)
	return true
}

// Award is used by other plugins to register a particular award for a user
func Award(nick, thing string, category Category) error {
	return nil
}
