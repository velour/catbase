package velouremon

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
)

type VelouremonPlugin struct {
	bot       bot.Bot
	db        *sqlx.DB
	channel   string
	threads   map[string]*Interaction
	players   []*Player
	creatures []*Creature
	abilities []*Ability
	timer     *time.Timer
}

func New(b bot.Bot) *VelouremonPlugin {
	vp := &VelouremonPlugin{
		bot:       b,
		db:        b.DB(),
		channel:   "",
		threads:   map[string]*Interaction{},
		players:   []*Player{},
		creatures: []*Creature{},
		abilities: []*Ability{},
		timer:     time.NewTimer(15 * time.Minute),
	}

	vp.checkAndBuildDBOrFail()
	vp.loadFromDB()

	b.Register(vp, bot.Message, vp.message)
	b.Register(vp, bot.Reply, vp.replyMessage)
	b.Register(vp, bot.Help, vp.help)

	go randomInteraction(b.DefaultConnector(), vp)

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

	if command == "status" {
		player, err := vp.getOrAddPlayer(message.User)
		if err != nil {
			return false
		}
		return vp.handleStatus(c, player)
	} else if command == "battle" {
		player, err := vp.getOrAddPlayer(message.User)
		if err != nil {
			return false
		}
		return vp.handleBattle(c, player)
	} else if len(tokens) > 1 {
		if command == "add_creature" {
			return vp.handleAddCreature(c, tokens[1:])
		} else if command == "add_ability" {
			return vp.handleAddAbility(c, tokens[1:])
		}
	}

	return false
}

func (vp *VelouremonPlugin) replyMessage(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	if !message.Command {
		return false
	}

	identifier := args[0].(string)
	if strings.ToLower(message.User.Name) != strings.ToLower(vp.bot.Config().Get("Nick", "bot")) {
		if interaction, ok := vp.threads[identifier]; ok {
			player, err := vp.getOrAddPlayer(message.User)
			if err != nil {
				return false
			}
			tokens := strings.Fields(message.Body)
			return interaction.handleMessage(vp, c, player, tokens)
		}
	}
	return false
}


func (vp *VelouremonPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	vp.bot.Send(c, bot.Message, message.Channel, "try something else, this is too complicated for you.")
	return true
}
