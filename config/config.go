package config

import "encoding/json"
import "fmt"
import "io/ioutil"

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	DbName                string
	DbServer              string
	Channels              []string
	MainChannel           string
	Plugins               []string
	Nick, Server, Pass    string
	Version               string
	CommandChar           string
	QuoteChance           float64
	QuoteTime             int
	LogLength             int
	Admins                []string
	UntappdToken          string
	UntappdFreq           int
	WelcomeMsgs           []string
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	TwitterUserKey        string
	TwitterUserSecret     string
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

	fmt.Printf("godeepintir version %s running.\n", c.Version)

	return &c
}
