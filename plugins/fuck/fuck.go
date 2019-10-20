package fuck

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/chrissexton/gofuck"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

type Fuck struct {
	b bot.Bot
	c *config.Config
}

func New(b bot.Bot) *Fuck {
	f := &Fuck{
		b: b,
		c: b.Config(),
	}

	b.Register(f, bot.Help, f.help)
	b.Register(f, bot.Message, f.message)

	return f
}

func (p *Fuck) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	helpMsg := "Run brainfuck with: `!fuck ```<program>``` <stdin>`\n\nFor example:\n> !fuck ```,[.,]``` hello"
	p.b.Send(c, bot.Message, message.Channel, helpMsg)
	return true
}

func (p *Fuck) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	lowerBody := strings.ToLower(message.Body)
	if message.Command && strings.HasPrefix(lowerBody, "fuck") {
		fields := strings.Split(message.Body, "```")
		if len(fields) < 2 {
			return false
		}

		pgm := fields[1]
		in := strings.Join(fields[2:], "```")

		log.Debug().
			Str("pgm", pgm).
			Str("in", in).
			Msg("Asked to fuck hard")

		stdin := bytes.NewBufferString(in)
		stdout := &bytes.Buffer{}

		m := gofuck.New(stdin, stdout)

		m.InstructionLimit = p.c.GetInt("fuck.limit.instr", 100000)
		if m.InstructionLimit < 1 {
			m.InstructionLimit = 1
		}
		maxOut := p.c.GetInt("fuck.limit.output", 1000)

		err := m.Run([]byte(pgm))
		if stdout.Len() > maxOut {
			stdout.Truncate(maxOut)
		}
		if err != nil {
			p.b.Send(c, bot.Message, message.Channel, fmt.Sprintf("Error running program: %s", err))
			if stdout.Len() > 0 {
				p.b.Send(c, bot.Message, message.Channel,
					fmt.Sprintf("Here's the output so far:\n%s", stdout.String()))
			}
			return true
		}

		p.b.Send(c, bot.Message, message.Channel, stdout.String())
		return true
	}
	return false
}
