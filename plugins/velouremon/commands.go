package velouremon

import (
	"github.com/velour/catbase/bot"
)

func (vp *VelouremonPlugin) handleStatus(c bot.Connector, player *Player, tokens []string) bool {
	vp.bot.Send(c, bot.Message, vp.channel, player.string())
	return true
}
