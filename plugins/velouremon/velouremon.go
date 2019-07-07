package velouremon

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type VelouremonHandler func(bot.Connector, *Player, []string) bool

type VelouremonPlugin struct {
	bot       bot.Bot
	db        *sqlx.DB
	channel   string
	handlers  map[string]VelouremonHandler
	threads   map[string]*Interaction
	players   []*Player
	creatures []*Creature
	abilities []*Ability
	timer     *time.Timer
}

func New(b bot.Bot) *VelouremonPlugin {
	dur, _ := time.ParseDuration("15m")
	timer := time.NewTimer(dur)

	vp := &VelouremonPlugin{
		bot:       b,
		db:        b.DB(),
		channel:   "",
		handlers:  map[string]VelouremonHandler{},
		threads:   map[string]*Interaction{},
		players:   []*Player{},
		creatures: []*Creature{},
		abilities: []*Ability{},
		timer:     timer,
	}

	vp.checkAndBuildDBOrFail()
	vp.loadFromDB()

	vp.handlers["status"] = vp.handleStatus

	b.Register(vp, bot.Message, vp.message)
	b.Register(vp, bot.Help, vp.help)

	return vp
}

func (vp *VelouremonPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !message.Command {
		return false
	}

	if vp.channel == "" {
		vp.channel = message.Channel
	}

	tokens := strings.Fields(message.Body)
	command := strings.ToLower(tokens[0])

	if fun, ok := vp.handlers[command]; ok {
		player, err := vp.getOrAddPlayer(message.User)
		if err != nil {
			return false
		}
		return fun(c, player, tokens[1:])
	}

	return false
}

func (vp *VelouremonPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	vp.bot.Send(c, bot.Message, message.Channel, "try something else, this is too complicated for you.")
	return true
}
