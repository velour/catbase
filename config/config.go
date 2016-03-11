// © 2013 the CatBase Authors under the WTFPL. See AUTHORS for the list of authors.

package config

import "encoding/json"
import "fmt"
import "io/ioutil"

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	DB struct {
		File   string
		Name   string
		Server string
	}
	Channels    []string
	MainChannel string
	Plugins     []string
	Type        string
	Irc         struct {
		Server, Pass string
	}
	Slack struct {
		Token string
	}
	Nick                  string
	FullName              string
	Version               string
	CommandChar           string
	RatePerSec            float64
	QuoteChance           float64
	QuoteTime             int
	LogLength             int
	Admins                []string
	HttpAddr              string
	UntappdToken          string
	UntappdFreq           int
	UntappdChannels       []string
	EnforceNicks          bool
	WelcomeMsgs           []string
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	TwitterUserKey        string
	TwitterUserSecret     string
	StartupFact           string
	BadMsgs               []string
	Bad                   struct {
		Msgs  []string
		Nicks []string
		Hosts []string
	}
}

// Readconfig loads the config data out of a JSON file located in cfile
func Readconfig(version, cfile string) *Config {
	fmt.Printf("Using %s as config file.\n", cfile)
	file, e := ioutil.ReadFile(cfile)
	if e != nil {
		panic("Couldn't read config file!")
	}

	var c Config
	err := json.Unmarshal(file, &c)
	if err != nil {
		panic(err)
	}
	c.Version = version

	if c.Type == "" {
		c.Type = "irc"
	}

	fmt.Printf("godeepintir version %s running.\n", c.Version)

	return &c
}
