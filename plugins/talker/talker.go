// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package talker

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/velour/catbase/bot"
	"github.com/velour/catbase/bot/msg"
	"github.com/velour/catbase/config"
)

var goatse []string = []string{
	"```* g o a t s e x * g o a t s e x * g o a t s e x *",
	"g                                               g",
	"o /     \\             \\            /    \\       o",
	"a|       |             \\          |      |      a",
	"t|       `.             |         |       :     t",
	"s`        |             |        \\|       |     s",
	"e \\       | /       /  \\\\\\   --__ \\\\       :    e",
	"x  \\      \\/   _--~~          ~--__| \\     |    x",
	"*   \\      \\_-~                    ~-_\\    |    *",
	"g    \\_     \\        _.--------.______\\|   |    g",
	"o      \\     \\______// _ ___ _ \\_\\__>  \\   |    o",
	"a       \\   .  C ___)  ______ \\_\\____>  |  /    a",
	"t       /\\ |   C ____)/      \\ \\_____>  |_/     t",
	"s      / /\\|   C_____)       |  \\___>   /  \\    s",
	"e     |   \\   _C_____)\\______/  // _/ /     \\   e",
	"x     |    \\  |__   \\\\_________// \\__/       |  x",
	"*    | \\    \\____)   `----   --'             |  *",
	"g    |  \\_          ___\\       /_          _/ | g",
	"o   |              /    |     |  \\            | o",
	"a   |             |    /       \\  \\           | a",
	"t   |          / /    |{nick}|  \\           |t",
	"s   |         / /      \\__/\\___/    |          |s",
	"e  |           /        |    |       |         |e",
	"x  |          |         |    |       |         |x",
	"* g o a t s e x * g o a t s e x * g o a t s e x *```",
}

type TalkerPlugin struct {
	Bot     bot.Bot
	config  *config.Config
	sayings []string
}

func New(b bot.Bot) *TalkerPlugin {
	tp := &TalkerPlugin{
		Bot:    b,
		config: b.Config(),
	}
	b.Register(tp, bot.Message, tp.message)
	b.Register(tp, bot.Help, tp.help)
	tp.registerWeb()
	return tp
}

func (p *TalkerPlugin) message(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	channel := message.Channel
	body := message.Body
	lowermessage := strings.ToLower(body)

	if message.Command && strings.HasPrefix(lowermessage, "cowsay") {
		msg, err := p.cowSay(strings.TrimPrefix(message.Body, "cowsay "))
		if err != nil {
			p.Bot.Send(bot.Message, channel, "Error running cowsay: %s", err)
			return true
		}
		p.Bot.Send(bot.Message, channel, msg)
		return true
	}

	if message.Command && strings.HasPrefix(lowermessage, "list cows") {
		cows := p.allCows()
		m := fmt.Sprintf("Cows: %s", strings.Join(cows, ", "))
		p.Bot.Send(bot.Message, channel, m)
		return true
	}

	// TODO: This ought to be space split afterwards to remove any punctuation
	if message.Command && strings.HasPrefix(lowermessage, "say") {
		msg := strings.TrimSpace(body[3:])
		p.Bot.Send(bot.Message, channel, msg)
		return true
	}

	if message.Command && strings.HasPrefix(lowermessage, "goatse") {
		nick := message.User.Name
		if parts := strings.Fields(message.Body); len(parts) > 1 {
			nick = parts[1]
		}

		output := ""
		for _, line := range goatse {
			nick = fmt.Sprintf("%9.9s", nick)
			line = strings.Replace(line, "{nick}", nick, 1)
			output += line + "\n"
		}
		p.Bot.Send(bot.Message, channel, output)
		return true
	}

	return false
}

func (p *TalkerPlugin) help(kind bot.Kind, message msg.Message, args ...interface{}) bool {
	p.Bot.Send(bot.Message, message.Channel, "Hi, this is talker. I like to talk about FredFelps!")
	return true
}

func (p *TalkerPlugin) cowSay(text string) (string, error) {
	fields := strings.Split(text, " ")
	cow := "default"
	if len(fields) > 1 && p.hasCow(fields[0]) {
		cow = fields[0]
		text = strings.Join(fields[1:], " ")
	}
	cmd := exec.Command("cowsay", "-f", cow, text)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err = cmd.Start(); err != nil {
		return "", err
	}

	output, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("```%s```", output), nil
}

func (p *TalkerPlugin) hasCow(cow string) bool {
	cows := p.allCows()
	for _, c := range cows {
		if strings.ToLower(cow) == c {
			return true
		}
	}
	return false
}

func (p *TalkerPlugin) allCows() []string {
	f, err := os.Open(p.config.Get("talker.cowpath", "/usr/local/share/cows"))
	if err != nil {
		return []string{"default"}
	}

	files, err := f.Readdir(0)
	if err != nil {
		return []string{"default"}
	}

	cows := []string{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".cow") {
			cows = append(cows, strings.TrimSuffix(f.Name(), ".cow"))
		}
	}
	return cows
}

func (p *TalkerPlugin) registerWeb() {
	http.HandleFunc("/slash/cowsay", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		log.Printf("Cowsay:\n%+v", r.PostForm.Get("text"))
		msg, err := p.cowSay(r.PostForm.Get("text"))
		if err != nil {
			fmt.Fprintf(w, "Error running cowsay: %s", err)
			return
		}
		fmt.Fprintf(w, "%s", msg)
	})
}
