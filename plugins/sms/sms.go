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
}

func New(b bot.Bot) *SMSPlugin {
	if plugin != nil {
		return plugin
	}
	plugin = &SMSPlugin{
		b: b,
		c: b.Config(),
	}
	plugin.setup()
	plugin.registerWeb()

	b.RegisterRegex(plugin, bot.Message, deleteRegex, plugin.deleteCmd)
	b.RegisterRegex(plugin, bot.Message, sendRegex, plugin.sendCmd)
	b.RegisterRegex(plugin, bot.Message, regSomeoneRegex, plugin.registerOtherCmd)
	b.RegisterRegex(plugin, bot.Message, regSelfRegex, plugin.registerSelfCmd)

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

var regSelfRegex = regexp.MustCompile(`(?i)my sms number is (?P<number>\d+)`)
var regSomeoneRegex = regexp.MustCompile(`(?i)register sms for (?P<name>\S+) (?P<number>\d+)`)
var sendRegex = regexp.MustCompile(`(?i)send sms to (?P<name>\S+) (?P<body>.+)`)
var deleteRegex = regexp.MustCompile(`(?i)delete my sms$`)

// Send will send a text to a registered user, who
func (p *SMSPlugin) Send(who, message string) error {
	num, err := p.get(who)
	if err != nil {
		return fmt.Errorf("unknown user: %s: %w", who, err)
	}
	sid := p.c.Get("TWILIOSID", "")
	token := p.c.Get("TWILIOTOKEN", "")
	myNum := p.c.Get("TWILIONUMBER", "")

	if sid == "" || token == "" || myNum == "" {
		return fmt.Errorf("this bot is not configured for Twilio")
	}

	client := twilio.NewClient(sid, token, nil)

	_, err = client.Messages.SendMessage(myNum, num, message, nil)
	return err
}

func (p *SMSPlugin) reg(c bot.Connector, ch, who, num string) bool {
	num, err := p.checkNumber(num)
	if err != nil {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("That number didn't make sense to me: %s", err))
		return true
	}
	me := p.b.WhoAmI()
	m := fmt.Sprintf("Hello from %s. You are registered for reminders and you can chat with the channel "+
		"by talking to this number", me)
	err = p.add(who, num)
	if err != nil {
		log.Error().Err(err).Msgf("error adding user")
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message, %s", err))
		return true
	}
	err = p.Send(who, m)
	if err != nil {
		log.Error().Err(err).Msgf("error sending to user")
		if err := p.delete(who); err != nil {
			log.Error().Err(err).Msgf("error deleting user")
		}
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message, %s", err))
		return true
	}
	p.b.Send(c, bot.Message, ch, "I sent a message to them.")
	return true
}

func (p *SMSPlugin) help(c bot.Connector, kind bot.Kind, message msg.Message, args ...interface{}) bool {
	ch := message.Channel
	m := fmt.Sprintf("You can register your number with: `%s`", regSelfRegex)
	m += fmt.Sprintf("\nYou can register somebody else with: `%s`", regSomeoneRegex)
	m += fmt.Sprintf("\nYou can send a message to a user with: `%s`", sendRegex)
	m += "\nYou can deregister with: `delete my sms`"
	m += fmt.Sprintf("\nAll messages sent to %s will come to the channel.",
		p.c.Get("TWILIONUMBER", "unknown"))
	m += "\nAll reminders with registered users will be sent to their SMS number."
	p.b.Send(c, bot.Message, ch, m)
	return true
}

func (p *SMSPlugin) registerWeb() {
	http.HandleFunc("/sms/new", p.receive)
}

func (p *SMSPlugin) receive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		fmt.Fprintf(w, "Incorrect HTTP method")
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Error().Err(err).Msgf("could not parse incoming SMS")
		return
	}

	body := r.PostFormValue("Body")
	number := strings.TrimPrefix(r.PostFormValue("From"), "+1")
	from, err := p.getByNumber(number)
	if err != nil {
		log.Debug().Err(err).Msgf("could not find user")
		from = number
	}

	m := fmt.Sprintf("SMS From %s: %s", from, body)
	chs := p.c.GetArray("channels", []string{})

	log.Debug().Msgf("%s to %v", m, chs)

	for _, ch := range chs {
		p.b.Send(p.b.DefaultConnector(), bot.Message, ch, m)
	}
}

func (p *SMSPlugin) add(who, num string) error {
	tx, err := p.b.DB().Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`insert or replace into sms (who, num) values (?, ?)`, who, num)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (p *SMSPlugin) delete(who string) error {
	tx, err := p.b.DB().Beginx()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from sms where who=?`, who)
	if err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (p *SMSPlugin) get(who string) (string, error) {
	num := ""
	err := p.b.DB().Get(&num, `select num from sms where who=?`, who)
	return num, err
}

func (p *SMSPlugin) getByNumber(num string) (string, error) {
	who := ""
	err := p.b.DB().Get(&who, `select who from sms where num=?`, num)
	return who, err
}

func (p *SMSPlugin) setup() {
	_, err := p.b.DB().Exec(`create table if not exists sms (
		who string primary key,
		num string
		)`)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not create sms table")
	}
}

func (p *SMSPlugin) deleteCmd(r bot.Request) bool {
	ch := r.Msg.Channel
	if err := p.delete(r.Msg.User.Name); err != nil {
		p.b.Send(r.Conn, bot.Message, ch, "Something went wrong.")
		return true
	}
	p.b.Send(r.Conn, bot.Message, ch, "Okay.")
	return true
}

func (p *SMSPlugin) sendCmd(r bot.Request) bool {
	if r.Msg.User.Name == p.c.Get("nick", "") {
		return false
	}
	ch := r.Msg.Channel
	chName := r.Msg.ChannelName
	c := r.Conn
	if r.Values["name"] == "" || r.Values["body"] == "" {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message. Your request makes no sense."))
		return true
	}
	user, body := r.Values["name"], fmt.Sprintf("#%s: %s", chName, r.Values["body"])
	err := p.Send(user, body)
	if err != nil {
		p.b.Send(c, bot.Message, ch, fmt.Sprintf("I didn't send the message, %s", err))
		return true
	}
	p.b.Send(c, bot.Message, ch, "I sent a message to them.")
	return true
}

func (p *SMSPlugin) registerOtherCmd(r bot.Request) bool {
	ch := r.Msg.Channel
	if r.Values["name"] == "" || r.Values["number"] == "" {
		p.b.Send(r.Conn, bot.Message, ch, fmt.Sprintf("if you're trying to register somebody, give me a "+
			"message of the format: `%s`", regSomeoneRegex))
		return true
	}

	return p.reg(r.Conn, ch, r.Values["name"], r.Values["number"])
}

func (p *SMSPlugin) registerSelfCmd(r bot.Request) bool {
	ch := r.Msg.Channel
	who := r.Msg.User.Name
	if r.Values["number"] == "" {
		p.b.Send(r.Conn, bot.Message, ch, fmt.Sprintf("if you're trying to register a number, give me a "+
			"message of the format: `%s`", regSelfRegex))
	}
	return p.reg(r.Conn, ch, who, r.Values["number"])
}
