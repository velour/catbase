package config

import (
	"bytes"
	"html/template"
	"log"
	"strings"
)

var q = `
INSERT INTO config VALUES('nick','{{.Nick}}');
INSERT INTO config VALUES('channels','{{.Channel}}');
INSERT INTO config VALUES('untappd.channels','{{.Channel}}');
INSERT INTO config VALUES('twitch.channels','{{.Channel}}');
INSERT INTO config VALUES('init',1);
`

func (c *Config) SetDefaults(mainChannel, nick string) {
	if nick == mainChannel && nick == "" {
		log.Fatalf("You must provide a nick and a mainChannel")
	}
	t := template.Must(template.New("query").Parse(q))
	vals := struct {
		Nick       string
		Channel    string
		ChannelKey string
	}{
		nick,
		mainChannel,
		strings.ToLower(mainChannel),
	}
	var buf bytes.Buffer
	t.Execute(&buf, vals)
	c.MustExec(`delete from config;`)
	c.MustExec(buf.String())
	log.Println("Configuration initialized.")
}
