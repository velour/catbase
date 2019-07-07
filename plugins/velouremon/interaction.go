package velouremon

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/velour/catbase/bot"
)

type Interaction struct {
	players   []*Player
	creatures []*Creature
}

func randomInteraction(c bot.Connector, vp *VelouremonPlugin) {
	for {
		<-vp.timer.C
		if vp.channel != "" {
			creature := vp.creatures[rand.Intn(len(vp.creatures))]
			message := fmt.Sprintf("A wild %s appeared.", creature.Name)
			vp.bot.Send(c, bot.Message, vp.channel, message)
		}

		dur, _ := time.ParseDuration("1h")
		vp.timer.Reset(dur)
	}
}
