package config

import (
	"bytes"
	"html/template"
	"log"
	"strings"
)

var q = `
INSERT INTO config VALUES('type','slack');
INSERT INTO config VALUES('nick','{{.Nick}}');
INSERT INTO config VALUES('channels','{{.Channel}}');
INSERT INTO config VALUES('factoid.quotetime',30);
INSERT INTO config VALUES('reaction.negativereactions','bullshit;;fake;;tableflip;;vomit');
INSERT INTO config VALUES('reaction.positivereactions','+1;;authorized;;aw_yeah;;yeah_man;;joy');
INSERT INTO config VALUES('reaction.generalchance',0.01);
INSERT INTO config VALUES('reaction.harrasschance',0.05);
INSERT INTO config VALUES('commandchar','!;;¡');
INSERT INTO config VALUES('factoid.startupfact','speed test');
INSERT INTO config VALUES('factoid.quotechance',0.99);
INSERT INTO config VALUES('factoid.minlen',4);
INSERT INTO config VALUES('untappd.channels','{{.Channel}}');
INSERT INTO config VALUES('twitch.channels','{{.Channel}}');
INSERT INTO config VALUES('twitch.{{.ChannelKey}}.users','drseabass;;phlyingpenguin;;stack5;;geoffwithaj;;msherms;;eaburns;;sheltim;;rathaus;;rcuhljr');
INSERT INTO config VALUES('twitch.freq',60);
INSERT INTO config VALUES('leftpad.maxlen',50);
INSERT INTO config VALUES('untappd.freq',60);
INSERT INTO config VALUES('your.replacements.0.freq',1);
INSERT INTO config VALUES('your.replacements.0.this','fuck');
INSERT INTO config VALUES('your.replacements.0.that','duck');
INSERT INTO config VALUES('your.replacements','0;;1;;2');
INSERT INTO config VALUES('httpaddr','127.0.0.1:1337');
INSERT INTO config VALUES('your.maxlength',140);
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
