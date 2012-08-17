package config

import "encoding/json"
import "fmt"
import "io/ioutil"

// Config stores any system-wide startup information that cannot be easily configured via
// the database
type Config struct {
	DbName             string
	DbServer           string
	Channels           []string
	Plugins            []string
	Nick, Server, Pass string
}

// Readconfig loads the config data out of a JSON file located in cfile
func Readconfig(cfile string) *Config {
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
	return &c
}
