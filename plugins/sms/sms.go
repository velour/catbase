package sms

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	twilio "github.com/kevinburke/twilio-go"
	"github.com/rs/zerolog/log"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

var plugin *SMSPlugin

type SMSPlugin struct {
	b bot.Bot
	c *config.Config

	knownUsers map[string]string
}

func New(b bot.Bot) *SMSPlugin {
	if plugin != nil {
		return plugin
	}
	plugin = &SMSPlugin{
		b:          b,
		c:          b.Config(),
		knownUsers: make(map[string]string),
	}
	b.Register(plugin, bot.Message, plugin.message)
	b.Register(plugin, bot.Help, plugin.help)
	return plugin
}

func (p *SMSPlugin) checkNumber(num string) (string, error) {
	expectedLen := 10

	if len(num) != expectedLen {
		return num, fmt.Errorf("invalid number length: %d, expected %d", len(num), expectedLen)
	}

	return num, nil
}

var regRegex = regexp.MustCompile(`^my sms number is (\d+)$`)

// Send will send a text to a registered user, who
func (p *SMSPlugin) Send(who, message string) error {
	num, ok := p.knownUsers[who]
	if !ok {
		return fmt.Errorf("unknown user: %s", who)
	}
	sid := p.c.Get("TWILIOSID", "")
	token := p.c.Get("TWILIOTOKEN", "")
	myNum := p.c.Get("TWILIONUMBER", "")
	client := twilio.NewClient(sid, token, nil)

	_, err := client.Messages.SendMessage(myNum, num, message, nil)
	return err
}

func (p *SMSPlugin) message(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	lowerBody := strings.ToLower(strings.TrimSpace(message.Body))

	log.Debug().Msgf("lowerBody: %s, match: %v", lowerBody, regRegex.MatchString(lowerBody))

	if lowerBody == "send me a message" {
		err := p.Send(message.User.Name, "Hey, here's a message.")
		if err != nil {
			p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message, %s", err))
			return true
		}
		p.b.Send(c, bot.Message, ch, "I sent a message to them.")
		return true
	}

	if regRegex.MatchString(lowerBody) {
		subs := regRegex.FindStringSubmatch(lowerBody)
		if subs == nil || len(subs) != 2 {
			p.b.Send(c, bot.Message, ch, fmt.Sprintf("if you're trying to register a number, give me a message of the format: `%s`", regRegex))
			return true
		}

		num, err := p.checkNumber(subs[1])
		if err != nil {
			p.b.Send(c, bot.Message, ch, fmt.Sprintf("That number didn't make sense to me: %s", err))
			return true
		}
		me := p.b.WhoAmI()
		m := fmt.Sprintf("Hello from %s", me)
		p.knownUsers[message.User.Name] = num
		err = p.Send(message.User.Name, m)
		if err != nil {
			delete(p.knownUsers, message.User.Name)
			p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message, %s", err))
			return true
		}
		p.b.Send(c, bot.Message, ch, "I sent a message to them.")
		return true
	}
	return false
}

func (p *SMSPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	p.b.Send(c, bot.Message, ch, "There is no help for you.")
	return true
}

func (p *SMSPlugin) registerWeb() {
	http.HandleFunc("/sms/recieve", p.receive)
}

func (p *SMSPlugin) receive(w http.ResponseWriter, r *http.Request) {

}
